package admin

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"

	"Stowaway/node"
	"Stowaway/share"
	"Stowaway/utils"
)

var AdminStatus *utils.AdminStatus

func init() {
	AdminStatus = new(utils.AdminStatus)
	AdminStatus.NewAdminStatus()
}

// NewAdmin 启动admin
func NewAdmin(c *utils.AdminOptions) {
	var initStatus string = "admin"
	AdminStatus.CliStatus = &initStatus

	adminCommandChan := make(chan []string)

	AdminStatus.AESKey = []byte(c.Secret)
	listenPort := c.Listen
	startNodeAddr := c.Connect
	rhostReuse := c.Rhostreuse

	Banner()

	if len(AdminStatus.AESKey) != 0 {
		log.Println("[*]Now Connection is encrypting with secret ", c.Secret)
	} else {
		log.Println("[*]Now Connection is maintianed without any encryption!")
	}

	if startNodeAddr == "" {
		go StartListen(listenPort, adminCommandChan)
	} else {
		ConnectToStartNode(startNodeAddr, rhostReuse, adminCommandChan)
	}

	go AddToChain()

	Controlpanel(adminCommandChan)
}

// StartListen 启动监听
func StartListen(listenPort string, adminCommandChan chan []string) {
	localAddr := fmt.Sprintf("0.0.0.0:%s", listenPort)
	localListener, err := net.Listen("tcp", localAddr)
	if err != nil {
		log.Fatalf("[*]Cannot listen %s", localAddr)
	}

	for {
		startNodeConn, _ := localListener.Accept() //一定要有连接进入才可继续操作，故没有连接时，admin端无法操作

		err = node.CheckSecret(startNodeConn, AdminStatus.AESKey)
		if err != nil {
			continue
		}

		HandleInitControlConn(startNodeConn, adminCommandChan)

		log.Printf("[*]StartNode connected from %s!\n", startNodeConn.RemoteAddr().String())

		return
	}
}

// ConnectToStartNode 主动连接startnode端代码
func ConnectToStartNode(startNodeAddr string, rhostReuse bool, adminCommandChan chan []string) {
	for {
		startNodeConn, err := net.Dial("tcp", startNodeAddr)
		if err != nil {
			log.Fatal("[*]Connection refused!")
		}

		if rhostReuse { //如果startnode在reuse状态下
			err = node.IfValid(startNodeConn)
			if err != nil {
				startNodeConn.Close()
				continue
			}
		} else {
			err := node.SendSecret(startNodeConn, AdminStatus.AESKey)
			if err != nil {
				log.Fatal("[*]Connection refused!")
			}
		}

		utils.ConstructPayloadAndSend(startNodeConn, utils.StartNodeId, "", "COMMAND", "STOWAWAYADMIN", " ", " ", 0, utils.AdminId, AdminStatus.AESKey, false)

		HandleInitControlConn(startNodeConn, adminCommandChan)

		log.Printf("[*]Connect to startnode %s successfully!\n", startNodeConn.RemoteAddr().String())

		return
	}
}

// HandleInitControlConn 初始化与startnode的连接
func HandleInitControlConn(startNodeConn net.Conn, adminCommandChan chan []string) error {
	for {
		command, err := utils.ExtractPayload(startNodeConn, AdminStatus.AESKey, utils.AdminId, true)
		if err != nil {
			log.Fatal("[*]Startnode seems offline, control channel set up failed.Exiting...")
		}
		switch command.Command {
		case "STOWAWAYAGENT":
			utils.ConstructPayloadAndSend(startNodeConn, utils.StartNodeId, "", "COMMAND", "CONFIRM", " ", " ", 0, utils.AdminId, AdminStatus.AESKey, false)
		case "INIT":
			utils.ConstructPayloadAndSend(startNodeConn, utils.StartNodeId, "", "COMMAND", "ID", " ", " ", 0, utils.AdminId, AdminStatus.AESKey, false)
			AdminStatus.StartNode = strings.Split(startNodeConn.RemoteAddr().String(), ":")[0]
			AdminStuff.NodeStatus.Nodenote[utils.StartNodeId] = ""
			AdminStatus.CurrentClient = append(AdminStatus.CurrentClient, utils.StartNodeId) //记录startnode加入网络
			AddNodeToTopology(utils.StartNodeId, utils.AdminId)
			CalRoute()
			go HandleStartConn(startNodeConn, adminCommandChan)
			go HandleCommandToControlConn(startNodeConn, adminCommandChan)
			return nil
		}
	}
}

// HandleStartConn 处理与startnode的信道
func HandleStartConn(startNodeConn net.Conn, adminCommandChan chan []string) {
	fileDataMap := utils.NewIntStrMap()
	cannotRead := make(chan bool, 1)

	for {
		nodeResp, err := utils.ExtractPayload(startNodeConn, AdminStatus.AESKey, utils.AdminId, true)
		if err != nil {
			log.Println("[*]StartNode seems offline")
			DelNodeFromTopology(utils.StartNodeId)
			AdminStatus.StartNode = "offline"
			startNodeConn.Close()
			break
		}
		switch nodeResp.Type {
		case "COMMAND":
			switch nodeResp.Command {
			case "NEW":
				nodeid := GenerateNodeID() //生成一个新的nodeid号进行分配
				log.Println("[*]New node join! Node Id is ", len(AdminStatus.CurrentClient))
				AdminStatus.NodesReadyToadd <- map[string]string{nodeid: nodeResp.Info} //将此节点加入detail命令所使用的NodeStatus.Nodes结构体
				AdminStuff.NodeStatus.Nodenote[nodeid] = ""                             //初始的note置空
				AddNodeToTopology(nodeid, nodeResp.CurrentId)                           //加入拓扑
				CalRoute()                                                              //计算路由
				SendPayloadViaRoute(startNodeConn, nodeid, "COMMAND", "ID", " ", " ", 0, utils.AdminId, AdminStatus.AESKey, false)
			case "AGENTOFFLINE":
				log.Println("[*]Node ", FindIntByNodeid(nodeResp.Info)+1, " seems offline") //有节点掉线后，将此节点及其之后的节点删除
				CloseAll(nodeResp.Info)                                                     //清除一切与此节点及其子节点有关的连接及功能
				DelNodeFromTopology(nodeResp.Info)                                          //从拓扑中删除
				//这里不用重新计算路由，因为控制端已经不会允许已掉线的节点及其子节点的流量流通
				if AdminStatus.HandleNode == nodeResp.Info && *AdminStatus.CliStatus != "admin" { //如果admin端正好操控此节点
					adminCommandChan <- []string{"exit"}
					<-AdminStatus.ReadyChange
					<-AdminStatus.IsShellMode
				}
			case "MYINFO": //拆分节点发送上来的节点自身信息
				info := strings.Split(nodeResp.Info, ":::stowaway:::")
				AdminStuff.NodeStatus.NodeHostname[nodeResp.CurrentId] = info[0]
				AdminStuff.NodeStatus.NodeUser[nodeResp.CurrentId] = info[1]
			case "MYNOTE":
				AdminStuff.NodeStatus.Nodenote[nodeResp.CurrentId] = nodeResp.Info
			case "SHELLSUCCESS":
				AdminStatus.ShellSuccess <- true
			case "SHELLFAIL":
				AdminStatus.ShellSuccess <- false
			case "SOCKSRESP":
				switch nodeResp.Info {
				case "SUCCESS":
					fmt.Println("[*]Socks5 service started successfully! Configure your browser‘s socks5 setting as [your admin serverip]:[port you specify]")
					AdminStatus.NodeSocksStarted <- true
				case "FAILED":
					fmt.Println("[*]Socks5 service started failed!")
					AdminStatus.NodeSocksStarted <- false
				}
			case "SSHRESP":
				switch nodeResp.Info {
				case "SUCCESS":
					AdminStatus.SSHSuccess <- true
					fmt.Println("[*]Node start ssh successfully!")
				case "FAILED":
					AdminStatus.SSHSuccess <- false
					fmt.Println("[*]Node start ssh failed!Check if target's ssh service is on or username and pass given are right")
					CommandContinue()
				}
			case "SSHTUNNELRESP":
				switch nodeResp.Info {
				case "SUCCESS":
					fmt.Println("[*]Successfully connect to node by ssh tunnel!")
				case "FAILED":
					fmt.Println("[*]Fail to connect to node by ssh tunnel! Something wrong is happened!")
				}
				CommandContinue()
			case "SSHCERTERROR":
				fmt.Println("[*]Ssh certificate seems wrong")
				AdminStatus.SSHSuccess <- false
				CommandContinue()
			case "NAMECONFIRM":
				AdminStatus.GetName <- true
			case "CREATEFAIL":
				AdminStatus.GetName <- false
			case "FILENAME":
				uploadFile, err := os.Create(nodeResp.Info)
				if err != nil {
					SendPayloadViaRoute(startNodeConn, AdminStatus.HandleNode, "COMMAND", "CREATEFAIL", " ", " ", 0, utils.AdminId, AdminStatus.AESKey, false)
				} else {
					SendPayloadViaRoute(startNodeConn, AdminStatus.HandleNode, "COMMAND", "NAMECONFIRM", " ", " ", 0, utils.AdminId, AdminStatus.AESKey, false)
					go share.ReceiveFile(Route.Route[AdminStatus.HandleNode], &startNodeConn, fileDataMap, cannotRead, uploadFile, AdminStatus.AESKey, true, utils.AdminId)
				}
			case "FILESIZE":
				share.File.FileSize, _ = strconv.ParseInt(nodeResp.Info, 10, 64)
				SendPayloadViaRoute(startNodeConn, AdminStatus.HandleNode, "COMMAND", "FILESIZECONFIRM", " ", " ", 0, utils.AdminId, AdminStatus.AESKey, false)
				share.File.ReceiveFileSize <- true
			case "FILESLICENUM":
				share.File.TotalSilceNum, _ = strconv.Atoi(nodeResp.Info)
				SendPayloadViaRoute(startNodeConn, AdminStatus.HandleNode, "COMMAND", "FILESLICENUMCONFIRM", " ", " ", 0, utils.AdminId, AdminStatus.AESKey, false)
				share.File.ReceiveFileSliceNum <- true
			case "FILESLICENUMCONFIRM":
				share.File.TotalConfirm <- true
			case "FILESIZECONFIRM":
				share.File.TotalConfirm <- true
			case "FILENOTEXIST":
				fmt.Printf("[*]File %s not exist!\n", nodeResp.Info)
			case "CANNOTREAD":
				fmt.Printf("[*]File %s cannot be read!\n", nodeResp.Info)
				cannotRead <- true
				share.File.ReceiveFileSliceNum <- false
				os.Remove(nodeResp.Info)
			case "CANNOTUPLOAD":
				fmt.Printf("[*]Agent cannot read file: %s\n", nodeResp.Info)
			case "GETREFLECTNUM":
				AdminStuff.ReflectNum.Lock()
				SendPayloadViaRoute(startNodeConn, AdminStatus.HandleNode, "COMMAND", "REFLECTNUM", " ", " ", AdminStuff.ReflectNum.Num, utils.AdminId, AdminStatus.AESKey, false)
				AdminStuff.ReflectNum.Num++
				AdminStuff.ReflectNum.Unlock()
			case "FIN":
				AdminStuff.ClientSockets.Lock()
				if _, ok := AdminStuff.ClientSockets.Payload[nodeResp.Clientid]; ok {
					AdminStuff.ClientSockets.Payload[nodeResp.Clientid].Close()
				}
				if _, ok := AdminStuff.ClientSockets.Payload[nodeResp.Clientid]; ok {
					delete(AdminStuff.ClientSockets.Payload, nodeResp.Clientid)
				}
				AdminStuff.ClientSockets.Unlock()
				SendPayloadViaRoute(startNodeConn, nodeResp.CurrentId, "COMMAND", "FINOK", " ", " ", nodeResp.Clientid, utils.AdminId, AdminStatus.AESKey, false)
			case "RECONNID":
				log.Println("[*]Node reconnect successfully!")
				ipAddress, upperNode := AnalysisInfo(nodeResp.Info)
				AdminStatus.NodesReadyToadd <- map[string]string{nodeResp.CurrentId: ipAddress}
				AdminStuff.NodeStatus.Nodenote[nodeResp.CurrentId] = ""
				ReconnAddCurrentClient(nodeResp.CurrentId) //在节点reconn回来的时候要考虑多种情况，若admin是掉线过，可以直接append，若admin没有掉线过，那么就需要判断重连回来的节点序号是否在CurrentClient中，如果已经存在就不需要append
				AddNodeToTopology(nodeResp.CurrentId, upperNode)
				CalRoute()
			case "HEARTBEAT":
				utils.ConstructPayloadAndSend(startNodeConn, utils.StartNodeId, "", "COMMAND", "KEEPALIVE", " ", " ", 0, utils.AdminId, AdminStatus.AESKey, false)
			case "TRANSSUCCESS":
				fmt.Println("[*]File transmission complete!")
			case "FORWARDFAIL":
				fmt.Println("[*]Remote port seems down,port forward failed!")
				AdminStuff.ForwardStatus.ForwardIsValid <- false
			case "FORWARDOK":
				fmt.Println("[*]Port forward successfully started!")
				AdminStuff.ForwardStatus.ForwardIsValid <- true
			case "FORWARDTIMEOUT":
				fallthrough
			case "FORWARDOFFLINE":
				AdminStuff.PortForWardMap.Lock()
				if _, ok := AdminStuff.PortForWardMap.Payload[nodeResp.Clientid]; ok {
					AdminStuff.PortForWardMap.Payload[nodeResp.Clientid].Close()
					delete(AdminStuff.PortForWardMap.Payload, nodeResp.Clientid)
				}
				AdminStuff.PortForWardMap.Unlock()
			case "REFLECTFAIL":
				fmt.Println("[*]Agent seems cannot listen this port,port reflect failed!")
			case "REFLECTOK":
				fmt.Println("[*]Port reflect successfully started!")
			case "REFLECT":
				TryReflect(startNodeConn, nodeResp.CurrentId, nodeResp.Clientid, nodeResp.Info)
			case "REFLECTFIN":
				AdminStuff.ReflectConnMap.Lock()
				if _, ok := AdminStuff.ReflectConnMap.Payload[nodeResp.Clientid]; ok {
					AdminStuff.ReflectConnMap.Payload[nodeResp.Clientid].Close()
					delete(AdminStuff.ReflectConnMap.Payload, nodeResp.Clientid)
				}
				AdminStuff.ReflectConnMap.Unlock()
				AdminStuff.PortReflectMap.Lock()
				if _, ok := AdminStuff.PortReflectMap.Payload[nodeResp.Clientid]; ok {
					if !utils.IsClosed(AdminStuff.PortReflectMap.Payload[nodeResp.Clientid]) {
						close(AdminStuff.PortReflectMap.Payload[nodeResp.Clientid])
						delete(AdminStuff.PortReflectMap.Payload, nodeResp.Clientid)
					}
				}
				AdminStuff.PortReflectMap.Unlock()
			case "NODECONNECTFAIL":
				fmt.Println("[*]Target seems down! Fail to connect")
			case "LISTENRESP":
				switch nodeResp.Info {
				case "FAILED":
					fmt.Println("[*]Cannot listen this port!")
				case "SUCCESS":
					fmt.Println("[*]Listen successfully!")
				}
			default:
				continue
			}
		case "DATA":
			switch nodeResp.Command {
			case "SHELLRESP":
				fmt.Print(nodeResp.Info)
			case "SSHMESS":
				fmt.Print(nodeResp.Info)
				fmt.Print("(ssh mode)>>>")
			case "SOCKSDATARESP":
				AdminStuff.ClientSockets.RLock()
				if _, ok := AdminStuff.ClientSockets.Payload[nodeResp.Clientid]; ok {
					_, err := AdminStuff.ClientSockets.Payload[nodeResp.Clientid].Write([]byte(nodeResp.Info))
					if err != nil {
						AdminStuff.ClientSockets.RUnlock()
						continue
					}
				}
				AdminStuff.ClientSockets.RUnlock()
			case "FILEDATA": //接收文件内容
				sliceNum, _ := strconv.Atoi(nodeResp.FileSliceNum)
				fileDataMap.Lock()
				fileDataMap.Payload[sliceNum] = nodeResp.Info
				fileDataMap.Unlock()
			case "FORWARDDATARESP":
				AdminStuff.PortForWardMap.Lock()
				if _, ok := AdminStuff.PortForWardMap.Payload[nodeResp.Clientid]; ok {
					AdminStuff.PortForWardMap.Payload[nodeResp.Clientid].Write([]byte(nodeResp.Info))
				}
				AdminStuff.PortForWardMap.Unlock()
			case "REFLECTDATA":
				AdminStuff.ReflectConnMap.RLock()
				if _, ok := AdminStuff.ReflectConnMap.Payload[nodeResp.Clientid]; ok {
					AdminStuff.PortReflectMap.Lock()
					if _, ok := AdminStuff.PortReflectMap.Payload[nodeResp.Clientid]; ok {
						AdminStuff.PortReflectMap.Payload[nodeResp.Clientid] <- nodeResp.Info
					} else {
						AdminStuff.PortReflectMap.Payload[nodeResp.Clientid] = make(chan string, 10)
						go HandleReflect(startNodeConn, AdminStuff.PortReflectMap.Payload[nodeResp.Clientid], nodeResp.Clientid, nodeResp.CurrentId)
						AdminStuff.PortReflectMap.Payload[nodeResp.Clientid] <- nodeResp.Info
					}
					AdminStuff.PortReflectMap.Unlock()
				}
				AdminStuff.ReflectConnMap.RUnlock()
			default:
				continue
			}
		}
	}
}
