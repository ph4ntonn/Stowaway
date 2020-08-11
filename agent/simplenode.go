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
func HandleSimpleNodeConn(connToUpperNode *net.Conn, nodeid string) {
	payloadBuffChan := make(chan *utils.Payload, 10)
	go HandleNodeConn(connToUpperNode, payloadBuffChan, nodeid)
	go HandleDataFromUpperNode(connToUpperNode, payloadBuffChan, nodeid)
	go HandleDataToUpperNode(connToUpperNode)
}

// HandleNodeConn 处理来自上级节点的信道
func HandleNodeConn(connToUpperNode *net.Conn, payloadBuffChan chan *utils.Payload, nodeid string) {
	for {
		command, err := utils.ExtractPayload(*connToUpperNode, AgentStatus.AESKey, nodeid, false)
		if err != nil {
			node.NodeStuff.Offline = true
			WaitingAdmin(nodeid) //上一级节点间网络连接断开后不掉线，等待上级节点重连回来
			go SendInfo(nodeid)  //重连后发送自身信息
			go SendNote(nodeid)  //重连后发送admin设置的备忘
			continue
		}
		payloadBuffChan <- command
	}
}

// HandleDataToUpperNode 处理发往上级节点的信道
func HandleDataToUpperNode(connToUpperNode *net.Conn) {
	for {
		proxyData := <-AgentStuff.ProxyChan.ProxyChanToUpperNode
		_, err := (*connToUpperNode).Write(proxyData)
		if err != nil {
			continue
		}
	}
}

// HandleDataFromUpperNode 处理来自上一级节点的数据
func HandleDataFromUpperNode(connToUpperNode *net.Conn, payloadBuffChan chan *utils.Payload, nodeid string) {
	var (
		err          error
		cannotRead   = make(chan bool, 1)
		getName      = make(chan bool, 1)
		fileDataChan = make(chan []byte, 1)
		stdin        io.Writer
		stdout       io.Reader
	)
	for {
		command := <-payloadBuffChan
		if command.NodeId == nodeid {
			switch command.Type {
			case "COMMAND":
				switch command.Command {
				case "SHELL":
					switch command.Info {
					case "":
						stdout, stdin, err = CreatInteractiveShell()
						if err != nil {
							message, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "SHELLFAIL", " ", "", 0, nodeid, AgentStatus.AESKey, false)
							AgentStuff.ProxyChan.ProxyChanToUpperNode <- message
							break
						}
						message, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "SHELLSUCCESS", " ", "", 0, nodeid, AgentStatus.AESKey, false)
						AgentStuff.ProxyChan.ProxyChanToUpperNode <- message
						go StartShell("", stdin, stdout, nodeid)
					default:
						stdin.Write([]byte(command.Info))
					}
				case "SOCKS":
					socksInfo := strings.Split(command.Info, ":::")
					AgentStuff.SocksInfo.SocksUsername = socksInfo[1]
					AgentStuff.SocksInfo.SocksPass = socksInfo[2]
					StartSocks()
				case "SOCKSOFF":
				case "UDPSTARTED":
					AgentStuff.Socks5UDPAssociate.Lock()
					AgentStuff.Socks5UDPAssociate.Info[command.Clientid].Ready <- command.Info
					AgentStuff.Socks5UDPAssociate.Unlock()
				case "SSH":
					go func() {
						err := StartSSH(command.Info, nodeid)
						if err == nil {
							go ReadCommand()
						}
					}()
				case "SSHCOMMAND":
					go WriteCommand(command.Info)
				case "SSHTUNNEL":
					go func() {
						err := SSHTunnelNextNode(command.Info, nodeid)
						if err != nil {
							fmt.Println("[*]", err)
						}
					}()
				case "CONNECT":
					go func() {
						var status bool = false
						command := strings.Split(command.Info, ":::")
						addr := command[0]
						choice := command[1]
						if choice == "1" {
							status = node.ConnectNextNodeReuse(addr, nodeid, AgentStatus.AESKey)
						} else {
							status = node.ConnectNextNode(addr, nodeid, AgentStatus.AESKey)
						}
						if !status {
							message, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "NODECONNECTFAIL", " ", "", 0, nodeid, AgentStatus.AESKey, false)
							AgentStuff.ProxyChan.ProxyChanToUpperNode <- message
						}
					}()
				case "FILENAME":
					uploadFile, err := os.Create(command.Info)
					if err != nil {
						respComm, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "CREATEFAIL", " ", " ", 0, nodeid, AgentStatus.AESKey, false)
						AgentStuff.ProxyChan.ProxyChanToUpperNode <- respComm
					} else {
						respComm, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "NAMECONFIRM", " ", " ", 0, nodeid, AgentStatus.AESKey, false)
						AgentStuff.ProxyChan.ProxyChanToUpperNode <- respComm
						go share.ReceiveFile("", connToUpperNode, fileDataChan, cannotRead, uploadFile, AgentStatus.AESKey, false, nodeid)
					}
				case "FILESIZE":
					share.File.FileSize, _ = strconv.ParseInt(command.Info, 10, 64)
					respComm, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "FILESIZECONFIRM", " ", " ", 0, nodeid, AgentStatus.AESKey, false)
					AgentStuff.ProxyChan.ProxyChanToUpperNode <- respComm
					share.File.ReceiveFileSize <- true
				case "FILESLICENUM":
					share.File.TotalSilceNum, _ = strconv.Atoi(command.Info)
					respComm, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "FILESLICENUMCONFIRM", " ", " ", 0, nodeid, AgentStatus.AESKey, false)
					AgentStuff.ProxyChan.ProxyChanToUpperNode <- respComm
					share.File.ReceiveFileSliceNum <- true
				case "FILESLICENUMCONFIRM":
					share.File.TotalConfirm <- true
				case "FILESIZECONFIRM":
					share.File.TotalConfirm <- true
				case "DOWNLOADFILE":
					go share.UploadFile("", command.Info, connToUpperNode, utils.AdminId, getName, AgentStatus.AESKey, nodeid, false)
				case "NAMECONFIRM":
					getName <- true
				case "CREATEFAIL":
					getName <- false
				case "CANNOTREAD":
					cannotRead <- true
					share.File.ReceiveFileSliceNum <- false
					os.Remove(command.Info)
				case "FORWARDTEST":
					go TestForward(command.Info)
				case "FORWARD": //连接指定需要映射的端口
					TryForward(command.Info, command.Clientid)
				case "FORWARDFIN":
					AgentStuff.ForwardConnMap.Lock()
					if _, ok := AgentStuff.ForwardConnMap.Payload[command.Clientid]; ok {
						AgentStuff.ForwardConnMap.Payload[command.Clientid].Close()
						delete(AgentStuff.ForwardConnMap.Payload, command.Clientid)
					}
					AgentStuff.ForwardConnMap.Unlock()
					AgentStuff.PortFowardMap.Lock()
					if _, ok := AgentStuff.PortFowardMap.Payload[command.Clientid]; ok {
						if !utils.IsClosed(AgentStuff.PortFowardMap.Payload[command.Clientid]) {
							close(AgentStuff.PortFowardMap.Payload[command.Clientid])
						}
					}
					AgentStuff.PortFowardMap.Unlock()
				case "REFLECTTEST":
					go TestReflect(command.Info)
				case "REFLECTTIMEOUT":
					fallthrough
				case "REFLECTOFFLINE":
					AgentStuff.ReflectConnMap.Lock()
					if _, ok := AgentStuff.ReflectConnMap.Payload[command.Clientid]; ok {
						AgentStuff.ReflectConnMap.Payload[command.Clientid].Close()
						delete(AgentStuff.ReflectConnMap.Payload, command.Clientid)
					}
					AgentStuff.ReflectConnMap.Unlock()
				case "REFLECTNUM":
					AgentStuff.ReflectStatus.ReflectNum <- command.Clientid
				case "STOPREFLECT":
					AgentStuff.ReflectConnMap.Lock()
					for key, conn := range AgentStuff.ReflectConnMap.Payload {
						conn.Close()
						delete(AgentStuff.ForwardConnMap.Payload, key)
					}
					AgentStuff.ReflectConnMap.Unlock()

					for _, listener := range CurrentPortReflectListener {
						listener.Close()
					}
				case "FIN":
					AgentStuff.CurrentSocks5Conn.Lock()
					if _, ok := AgentStuff.CurrentSocks5Conn.Payload[command.Clientid]; ok {
						AgentStuff.CurrentSocks5Conn.Payload[command.Clientid].Close()
						delete(AgentStuff.CurrentSocks5Conn.Payload, command.Clientid)
					}
					AgentStuff.CurrentSocks5Conn.Unlock()
					AgentStuff.SocksDataChanMap.Lock()
					if _, ok := AgentStuff.SocksDataChanMap.Payload[command.Clientid]; ok {
						if !utils.IsClosed(AgentStuff.SocksDataChanMap.Payload[command.Clientid]) {
							close(AgentStuff.SocksDataChanMap.Payload[command.Clientid])
						}
						delete(AgentStuff.SocksDataChanMap.Payload, command.Clientid)
					}
					AgentStuff.SocksDataChanMap.Unlock()
				case "FINOK":
					AgentStuff.SocksDataChanMap.Lock()
					if _, ok := AgentStuff.SocksDataChanMap.Payload[command.Clientid]; ok {
						if !utils.IsClosed(AgentStuff.SocksDataChanMap.Payload[command.Clientid]) {
							close(AgentStuff.SocksDataChanMap.Payload[command.Clientid])
						}
						delete(AgentStuff.SocksDataChanMap.Payload, command.Clientid)
					}
					AgentStuff.SocksDataChanMap.Unlock()
				case "RECONN":
					respCommand, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "RECONNID", " ", "", 0, nodeid, AgentStatus.AESKey, false)
					AgentStuff.ProxyChan.ProxyChanToUpperNode <- respCommand
					go SendInfo(nodeid) //重连后发送自身信息
					go SendNote(nodeid) //重连后发送admin设置的备忘
					if AgentStatus.NotLastOne {
						BroadCast("RECONN")
					}
				case "CLEAR":
					ClearAllConn()
					AgentStuff.SocksDataChanMap = utils.NewUint32ChanStrMap()
					if AgentStatus.NotLastOne {
						BroadCast("CLEAR")
					}
				case "LISTEN":
					go func() {
						err := TestListen(command.Info)
						if err != nil {
							respComm, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "LISTENRESP", " ", "FAILED", 0, nodeid, AgentStatus.AESKey, false)
							AgentStuff.ProxyChan.ProxyChanToUpperNode <- respComm
						} else {
							respComm, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "LISTENRESP", " ", "SUCCESS", 0, nodeid, AgentStatus.AESKey, false)
							AgentStuff.ProxyChan.ProxyChanToUpperNode <- respComm
							go node.StartNodeListen(command.Info, nodeid, AgentStatus.AESKey)
						}
					}()
				case "YOURINFO":
					AgentStatus.NodeNote = command.Info
				case "UDPFIN":
					fallthrough
				case "UDPFINOK":
					AgentStuff.Socks5UDPAssociate.Lock()
					if _, ok := AgentStuff.Socks5UDPAssociate.Info[command.Clientid]; ok {
						AgentStuff.Socks5UDPAssociate.Info[command.Clientid].Listener.Close()
						if !utils.IsClosed(AgentStuff.Socks5UDPAssociate.Info[command.Clientid].Ready) {
							close(AgentStuff.Socks5UDPAssociate.Info[command.Clientid].Ready)
						}
						if !utils.IsClosed(AgentStuff.Socks5UDPAssociate.Info[command.Clientid].UDPData) {
							close(AgentStuff.Socks5UDPAssociate.Info[command.Clientid].UDPData)
						}
						delete(AgentStuff.Socks5UDPAssociate.Info, command.Clientid)
					}
					AgentStuff.Socks5UDPAssociate.Unlock()
				case "KEEPALIVE":
				default:
					continue
				}
			case "DATA":
				switch command.Command {
				case "TCPSOCKSDATA":
					AgentStuff.SocksDataChanMap.Lock()
					if _, ok := AgentStuff.SocksDataChanMap.Payload[command.Clientid]; ok {
						AgentStuff.SocksDataChanMap.Payload[command.Clientid] <- command.Info
					} else {
						AgentStuff.SocksDataChanMap.Payload[command.Clientid] = make(chan string, 1)
						go HanleClientSocksConn(AgentStuff.SocksDataChanMap.Payload[command.Clientid], AgentStuff.SocksInfo.SocksUsername, AgentStuff.SocksInfo.SocksPass, command.Clientid, nodeid)
						AgentStuff.SocksDataChanMap.Payload[command.Clientid] <- command.Info
					}
					AgentStuff.SocksDataChanMap.Unlock()
				case "UDPSOCKSDATA":
					AgentStuff.Socks5UDPAssociate.Lock()
					if _, ok := AgentStuff.Socks5UDPAssociate.Info[command.Clientid]; ok {
						AgentStuff.Socks5UDPAssociate.Info[command.Clientid].UDPData <- command.Info
					}
					AgentStuff.Socks5UDPAssociate.Unlock()
				case "FILEDATA": //接收文件内容
					fileDataChan <- []byte(command.Info)
				case "FORWARDDATA":
					AgentStuff.ForwardConnMap.RLock()
					if _, ok := AgentStuff.ForwardConnMap.Payload[command.Clientid]; ok {
						AgentStuff.PortFowardMap.Lock()
						if _, ok := AgentStuff.PortFowardMap.Payload[command.Clientid]; ok {
							AgentStuff.PortFowardMap.Payload[command.Clientid] <- command.Info
						} else {
							AgentStuff.PortFowardMap.Payload[command.Clientid] = make(chan string, 1)
							go HandleForward(AgentStuff.PortFowardMap.Payload[command.Clientid], command.Clientid)
							AgentStuff.PortFowardMap.Payload[command.Clientid] <- command.Info
						}
						AgentStuff.PortFowardMap.Unlock()
					}
					AgentStuff.ForwardConnMap.RUnlock()
				case "REFLECTDATARESP":
					AgentStuff.ReflectConnMap.Lock()
					AgentStuff.ReflectConnMap.Payload[command.Clientid].Write([]byte(command.Info))
					AgentStuff.ReflectConnMap.Unlock()
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
			AgentStuff.ProxyChan.ProxyChanToLowerNode <- passToLowerData
		}
	}
}

//普通节点启动代码结束
