package agent

import (
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"

	"Stowaway/node"
	"Stowaway/share"
	"Stowaway/utils"
)

//普通节点代码与startnode端代码绝大部分相同，这里仅是为了将角色代码独立开，方便修改，故而不复用代码，看起来清楚一点

// HandleSimpleNodeConn 启动普通节点
func HandleSimpleNodeConn(connToUpperNode *net.Conn, NODEID string) {
	go HandleConnFromUpperNode(connToUpperNode, NODEID)
	go HandleConnToUpperNode(connToUpperNode)
}

// HandleConnToUpperNode 处理发往上一级节点的控制信道
func HandleConnToUpperNode(connToUpperNode *net.Conn) {
	for {
		proxyData := <-ProxyChan.ProxyChanToUpperNode
		_, err := (*connToUpperNode).Write(proxyData)
		if err != nil {
			continue
		}
	}
}

// HandleConnFromUpperNode 处理来自上一级节点的控制信道
func HandleConnFromUpperNode(connToUpperNode *net.Conn, NODEID string) {
	var (
		CannotRead = make(chan bool, 1)
		GetName    = make(chan bool, 1)
		stdin      io.Writer
		stdout     io.Reader
	)
	for {
		command, err := utils.ExtractPayload(*connToUpperNode, AgentStatus.AESKey, NODEID, false)
		if err != nil {
			node.NodeStuff.Offline = true
			WaitingAdmin(NODEID) //上一级节点间网络连接断开后不掉线，等待上级节点重连回来
			go SendInfo(NODEID)  //重连后发送自身信息
			go SendNote(NODEID)  //重连后发送admin设置的备忘
			continue
		}
		if command.NodeId == NODEID {
			switch command.Type {
			case "COMMAND":
				switch command.Command {
				case "SHELL":
					switch command.Info {
					case "":
						stdout, stdin = CreatInteractiveShell()
						go func() {
							StartShell("", stdin, stdout, NODEID)
						}()
					case "exit\n":
						fallthrough
					default:
						go func() {
							StartShell(command.Info, stdin, stdout, NODEID)
						}()
					}
				case "SOCKS":
					socksinfo := strings.Split(command.Info, ":::")
					SocksInfo.SocksUsername = socksinfo[1]
					SocksInfo.SocksPass = socksinfo[2]
					StartSocks()
				case "SOCKSOFF":
				case "SSH":
					err := StartSSH(command.Info, NODEID)
					if err == nil {
						go ReadCommand()
					} else {
						break
					}
				case "SSHCOMMAND":
					go WriteCommand(command.Info)
				case "SSHTUNNEL":
					err := SSHTunnelNextNode(command.Info, NODEID)
					if err != nil {
						fmt.Println("[*]", err)
						break
					}
				case "CONNECT":
					var status bool = false
					command := strings.Split(command.Info, ":::")
					addr := command[0]
					choice := command[1]
					if choice == "1" {
						status = node.ConnectNextNodeReuse(addr, NODEID, AgentStatus.AESKey)
					} else {
						status = node.ConnectNextNode(addr, NODEID, AgentStatus.AESKey)
					}
					if !status {
						message, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "NODECONNECTFAIL", " ", "", 0, NODEID, AgentStatus.AESKey, false)
						ProxyChan.ProxyChanToUpperNode <- message
					}
				case "FILENAME":
					UploadFile, err := os.Create(command.Info)
					if err != nil {
						respComm, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "CREATEFAIL", " ", " ", 0, NODEID, AgentStatus.AESKey, false)
						ProxyChan.ProxyChanToUpperNode <- respComm
					} else {
						respComm, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "NAMECONFIRM", " ", " ", 0, NODEID, AgentStatus.AESKey, false)
						ProxyChan.ProxyChanToUpperNode <- respComm
						go share.ReceiveFile("", connToUpperNode, FileDataMap, CannotRead, UploadFile, AgentStatus.AESKey, false, NODEID)
					}
				case "FILESIZE":
					share.File.FileSize, _ = strconv.ParseInt(command.Info, 10, 64)
					respComm, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "FILESIZECONFIRM", " ", " ", 0, NODEID, AgentStatus.AESKey, false)
					ProxyChan.ProxyChanToUpperNode <- respComm
					share.File.ReceiveFileSize <- true
				case "FILESLICENUM":
					share.File.TotalSilceNum, _ = strconv.Atoi(command.Info)
					respComm, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "FILESLICENUMCONFIRM", " ", " ", 0, NODEID, AgentStatus.AESKey, false)
					ProxyChan.ProxyChanToUpperNode <- respComm
					share.File.ReceiveFileSliceNum <- true
				case "FILESLICENUMCONFIRM":
					share.File.TotalConfirm <- true
				case "FILESIZECONFIRM":
					share.File.TotalConfirm <- true
				case "DOWNLOADFILE":
					go share.UploadFile("", command.Info, connToUpperNode, utils.AdminId, GetName, AgentStatus.AESKey, NODEID, false)
				case "NAMECONFIRM":
					GetName <- true
				case "CREATEFAIL":
					GetName <- false
				case "CANNOTREAD":
					CannotRead <- true
					share.File.ReceiveFileSliceNum <- false
					os.Remove(command.Info)
				case "FORWARDTEST":
					go TestForward(command.Info)
				case "REFLECTTEST":
					go TestReflect(command.Info)
				case "REFLECTNUM":
					ReflectStatus.ReflectNum <- command.Clientid
				case "STOPREFLECT":
					ReflectConnMap.Lock()
					for key, conn := range ReflectConnMap.Payload {
						conn.Close()
						delete(ForwardConnMap.Payload, key)
					}
					ReflectConnMap.Unlock()

					for _, listener := range CurrentPortReflectListener {
						listener.Close()
					}

				case "RECONN":
					respCommand, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "RECONNID", " ", "", 0, NODEID, AgentStatus.AESKey, false)
					ProxyChan.ProxyChanToUpperNode <- respCommand
					go SendInfo(NODEID) //重连后发送自身信息
					go SendNote(NODEID) //重连后发送admin设置的备忘
					if AgentStatus.NotLastOne {
						BroadCast("RECONN")
					}
				case "CLEAR":
					ClearAllConn()
					SocksDataChanMap = utils.NewUint32ChanStrMap()
					if AgentStatus.NotLastOne {
						BroadCast("CLEAR")
					}
				case "LISTEN":
					err := TestListen(command.Info)
					if err != nil {
						respComm, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "LISTENRESP", " ", "FAILED", 0, NODEID, AgentStatus.AESKey, false)
						ProxyChan.ProxyChanToUpperNode <- respComm
					} else {
						respComm, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "LISTENRESP", " ", "SUCCESS", 0, NODEID, AgentStatus.AESKey, false)
						ProxyChan.ProxyChanToUpperNode <- respComm
						go node.StartNodeListen(command.Info, NODEID, AgentStatus.AESKey)
					}
				case "YOURINFO":
					AgentStatus.NodeNote = command.Info
				case "KEEPALIVE":
				default:
					continue
				}
			case "DATA":
				switch command.Command {
				case "SOCKSDATA":
					SocksDataChanMap.Lock()
					if _, ok := SocksDataChanMap.Payload[command.Clientid]; ok {
						SocksDataChanMap.Payload[command.Clientid] <- command.Info
					} else {
						SocksDataChanMap.Payload[command.Clientid] = make(chan string, 1)
						go HanleClientSocksConn(SocksDataChanMap.Payload[command.Clientid], SocksInfo.SocksUsername, SocksInfo.SocksPass, command.Clientid, NODEID)
						SocksDataChanMap.Payload[command.Clientid] <- command.Info
					}
					SocksDataChanMap.Unlock()
				case "FINOK":
					SocksDataChanMap.Lock()
					if _, ok := SocksDataChanMap.Payload[command.Clientid]; ok {
						if !utils.IsClosed(SocksDataChanMap.Payload[command.Clientid]) {
							close(SocksDataChanMap.Payload[command.Clientid])
						}
						delete(SocksDataChanMap.Payload, command.Clientid)
					}
					SocksDataChanMap.Unlock()
				case "FILEDATA": //接收文件内容
					slicenum, _ := strconv.Atoi(command.FileSliceNum)
					FileDataMap.Lock()
					FileDataMap.Payload[slicenum] = command.Info
					FileDataMap.Unlock()
				case "FIN":
					CurrentConn.Lock()
					if _, ok := CurrentConn.Payload[command.Clientid]; ok {
						CurrentConn.Payload[command.Clientid].Close()
						delete(CurrentConn.Payload, command.Clientid)
					}
					CurrentConn.Unlock()
					SocksDataChanMap.Lock()
					if _, ok := SocksDataChanMap.Payload[command.Clientid]; ok {
						if !utils.IsClosed(SocksDataChanMap.Payload[command.Clientid]) {
							close(SocksDataChanMap.Payload[command.Clientid])
						}
						delete(SocksDataChanMap.Payload, command.Clientid)
					}
					SocksDataChanMap.Unlock()
				case "FORWARD": //连接指定需要映射的端口
					TryForward(command.Info, command.Clientid)
				case "FORWARDDATA":
					ForwardConnMap.RLock()
					if _, ok := ForwardConnMap.Payload[command.Clientid]; ok {
						PortFowardMap.Lock()
						if _, ok := PortFowardMap.Payload[command.Clientid]; ok {
							PortFowardMap.Payload[command.Clientid] <- command.Info
						} else {
							PortFowardMap.Payload[command.Clientid] = make(chan string, 1)
							go HandleForward(PortFowardMap.Payload[command.Clientid], command.Clientid)
							PortFowardMap.Payload[command.Clientid] <- command.Info
						}
						PortFowardMap.Unlock()
					}
					ForwardConnMap.RUnlock()
				case "FORWARDFIN":
					ForwardConnMap.Lock()
					if _, ok := ForwardConnMap.Payload[command.Clientid]; ok {
						ForwardConnMap.Payload[command.Clientid].Close()
						delete(ForwardConnMap.Payload, command.Clientid)
					}
					ForwardConnMap.Unlock()
					PortFowardMap.Lock()
					if _, ok := PortFowardMap.Payload[command.Clientid]; ok {
						if !utils.IsClosed(PortFowardMap.Payload[command.Clientid]) {
							close(PortFowardMap.Payload[command.Clientid])
						}
					}
					PortFowardMap.Unlock()
				case "REFLECTDATARESP":
					ReflectConnMap.Lock()
					ReflectConnMap.Payload[command.Clientid].Write([]byte(command.Info))
					ReflectConnMap.Unlock()
				case "REFLECTTIMEOUT":
					fallthrough
				case "REFLECTOFFLINE":
					ReflectConnMap.Lock()
					if _, ok := ReflectConnMap.Payload[command.Clientid]; ok {
						ReflectConnMap.Payload[command.Clientid].Close()
						delete(ReflectConnMap.Payload, command.Clientid)
					}
					ReflectConnMap.Unlock()
				default:
					continue
				}
			}
		} else {
			//判断是不是admin下发的ID包，是的话提取其中的id，将其记录至自己的子节点记录
			if command.Route == "" && command.Command == "ID" {
				AgentStatus.WaitForIDAllocate <- command.NodeId
				node.NodeInfo.LowerNode.Lock()
				node.NodeInfo.LowerNode.Payload[command.NodeId] = node.NodeInfo.LowerNode.Payload[utils.AdminId]
				node.NodeInfo.LowerNode.Unlock()
			}

			routeid := ChangeRoute(command)
			proxyData, _ := utils.ConstructPayload(command.NodeId, command.Route, command.Type, command.Command, command.FileSliceNum, command.Info, command.Clientid, command.CurrentId, AgentStatus.AESKey, true)
			//新建包结构体
			passToLowerData := utils.NewPassToLowerNodeData()
			//如果返回的routeid是空，说明目标节点就是自身的子节点，不需要多轮递送
			if routeid == "" {
				passToLowerData.Route = command.NodeId
			} else {
				passToLowerData.Route = routeid
			}
			//组装包的数据部分
			passToLowerData.Data = proxyData
			//递交数据包
			ProxyChan.ProxyChanToLowerNode <- passToLowerData
		}
	}
}

//普通节点启动代码结束
