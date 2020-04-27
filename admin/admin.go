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

var (
	CliStatus      *string
	CurrentClient  []string //记录当前网络中的节点，主要用来将string型的id对照至int型的序号，方便用户理解
	AdminStatus    *utils.AdminStatus
	FileDataMap    *utils.IntStrMap
	ClientSockets  *utils.Uint32ConnMap
	PortForWardMap *utils.Uint32ConnMap
)

//启动admin
func NewAdmin(c *utils.AdminOptions) {
	var InitStatus string = "admin"
	AdminStatus = utils.NewAdminStatus()
	ClientSockets = utils.NewUint32ConnMap()
	FileDataMap = utils.NewIntStrMap()
	PortForWardMap = utils.NewUint32ConnMap()
	AdminStatus.AESKey = []byte(c.Secret)
	listenPort := c.Listen
	startnodeaddr := c.Connect
	rhostreuse := c.Rhostreuse

	Banner()

	if len(AdminStatus.AESKey) != 0 {
		log.Println("[*]Now Connection is encrypting with secret ", c.Secret)
	} else {
		log.Println("[*]Now Connection is maintianed without any encryption!")
	}
	if startnodeaddr == "" {
		go StartListen(listenPort)
	} else {
		ConnectToStartNode(startnodeaddr, rhostreuse)
	}
	go AddToChain()
	CliStatus = &InitStatus
	Controlpanel()
}

//主动连接startnode端代码
func ConnectToStartNode(startnodeaddr string, rhostreuse bool) {
	for {
		startNodeConn, err := net.Dial("tcp", startnodeaddr)
		if err != nil {
			log.Println("[*]Connection refused!")
			os.Exit(0)
		}

		if rhostreuse { //如果startnode在reuse状态下
			err = node.IfValid(startNodeConn)
			if err != nil {
				startNodeConn.Close()
				continue
			}
		} else {
			err := node.SendSecret(startNodeConn, AdminStatus.AESKey)
			if err != nil {
				log.Println("[*]Connection refused!")
				os.Exit(0)
			}
		}

		helloMess, _ := utils.ConstructPayload(utils.StartNodeId, "", "COMMAND", "STOWAWAYADMIN", " ", " ", 0, utils.AdminId, AdminStatus.AESKey, false)
		startNodeConn.Write(helloMess)
		for {
			command, _ := utils.ExtractPayload(startNodeConn, AdminStatus.AESKey, utils.AdminId, true)
			switch command.Command {
			case "INIT":
				respCommand, _ := utils.ConstructPayload(utils.StartNodeId, "", "COMMAND", "ID", " ", " ", 0, utils.AdminId, AdminStatus.AESKey, false)
				startNodeConn.Write(respCommand)
				AdminStuff.StartNode = strings.Split(startNodeConn.RemoteAddr().String(), ":")[0]
				log.Printf("[*]Connect to startnode %s successfully!\n", startNodeConn.RemoteAddr().String())
				NodeStatus.Nodenote[utils.StartNodeId] = ""
				CurrentClient = append(CurrentClient, utils.StartNodeId) //记录startnode加入网络
				AddNodeToTopology(utils.StartNodeId, utils.AdminId)
				CalRoute()
				go HandleStartConn(startNodeConn)
				go HandleCommandToControlConn(startNodeConn)
				return
			}
		}
	}
}

//启动监听
func StartListen(listenPort string) {
	localAddr := fmt.Sprintf("0.0.0.0:%s", listenPort)
	localListener, err := net.Listen("tcp", localAddr)
	if err != nil {
		log.Printf("[*]Cannot listen %s", localAddr)
		os.Exit(0)
	}
	for {
		startNodeConn, _ := localListener.Accept() //一定要有连接进入才可继续操作，故没有连接时，admin端无法操作

		err = node.CheckSecret(startNodeConn, AdminStatus.AESKey)
		if err != nil {
			continue
		}

		log.Printf("[*]StartNode connected from %s!\n", startNodeConn.RemoteAddr().String())
		AdminStuff.StartNode = strings.Split(startNodeConn.RemoteAddr().String(), ":")[0]
		go HandleInitControlConn(startNodeConn)
		break
	}
}

// 初始化与startnode的连接
func HandleInitControlConn(startNodeConn net.Conn) {
	for {
		command, err := utils.ExtractPayload(startNodeConn, AdminStatus.AESKey, utils.AdminId, true)
		if err != nil {
			log.Println("[*]Startnode seems offline, control channel set up failed.Exiting...")
			return
		}
		switch command.Command {
		case "STOWAWAYAGENT":
			Message, _ := utils.ConstructPayload(utils.StartNodeId, "", "COMMAND", "CONFIRM", " ", " ", 0, utils.AdminId, AdminStatus.AESKey, false)
			startNodeConn.Write(Message)
		case "INIT":
			respCommand, _ := utils.ConstructPayload(utils.StartNodeId, "", "COMMAND", "ID", " ", " ", 0, utils.AdminId, AdminStatus.AESKey, false)
			startNodeConn.Write(respCommand)
			NodeStatus.Nodenote[utils.StartNodeId] = ""
			CurrentClient = append(CurrentClient, utils.StartNodeId) //记录startnode加入网络
			AddNodeToTopology(utils.StartNodeId, utils.AdminId)
			CalRoute()
			go HandleStartConn(startNodeConn)
			go HandleCommandToControlConn(startNodeConn)
			return
		}
	}
}

// 处理与startnode的信道
func HandleStartConn(startNodeConn net.Conn) {
	for {
		nodeResp, err := utils.ExtractPayload(startNodeConn, AdminStatus.AESKey, utils.AdminId, true)
		if err != nil {
			log.Println("[*]StartNode seems offline")
			DelNodeFromTopology(utils.StartNodeId)
			AdminStuff.StartNode = "offline"
			startNodeConn.Close()
			break
		}
		switch nodeResp.Type {
		case "DATA":
			switch nodeResp.Command {
			case "SHELLRESP":
				fmt.Print(nodeResp.Info)
			case "SSHMESS":
				fmt.Print(nodeResp.Info)
				fmt.Print("(ssh mode)>>>")
			case "SOCKSDATARESP":
				ClientSockets.RLock()
				if _, ok := ClientSockets.Payload[nodeResp.Clientid]; ok {
					_, err := ClientSockets.Payload[nodeResp.Clientid].Write([]byte(nodeResp.Info))
					if err != nil {
						ClientSockets.RUnlock()
						continue
					}
				}
				ClientSockets.RUnlock()
			case "FIN":
				ClientSockets.RLock()
				if _, ok := ClientSockets.Payload[nodeResp.Clientid]; ok {
					ClientSockets.Payload[nodeResp.Clientid].Close()
				}
				ClientSockets.RUnlock()
				ClientSockets.Lock()
				if _, ok := ClientSockets.Payload[nodeResp.Clientid]; ok {
					delete(ClientSockets.Payload, nodeResp.Clientid)
				}
				ClientSockets.Unlock()
				Route.Lock()
				respCommand, _ := utils.ConstructPayload(nodeResp.CurrentId, Route.Route[nodeResp.CurrentId], "DATA", "FINOK", " ", " ", nodeResp.Clientid, utils.AdminId, AdminStatus.AESKey, false)
				Route.Unlock()
				startNodeConn.Write(respCommand)
			case "FILEDATA": //接收文件内容
				slicenum, _ := strconv.Atoi(nodeResp.FileSliceNum)
				FileDataMap.Lock()
				FileDataMap.Payload[slicenum] = nodeResp.Info
				FileDataMap.Unlock()
			case "FORWARDDATARESP":
				PortForWardMap.Lock()
				if _, ok := PortForWardMap.Payload[nodeResp.Clientid]; ok {
					PortForWardMap.Payload[nodeResp.Clientid].Write([]byte(nodeResp.Info))
				}
				PortForWardMap.Unlock()
			case "FORWARDTIMEOUT":
				fallthrough
			case "FORWARDOFFLINE":
				PortForWardMap.Lock()
				if _, ok := PortForWardMap.Payload[nodeResp.Clientid]; ok {
					PortForWardMap.Payload[nodeResp.Clientid].Close()
					delete(PortForWardMap.Payload, nodeResp.Clientid)
				}
				PortForWardMap.Unlock()
			case "REFLECT":
				TryReflect(startNodeConn, nodeResp.CurrentId, nodeResp.Clientid, nodeResp.Info)
			case "REFLECTFIN":
				ReflectConnMap.Lock()
				if _, ok := ReflectConnMap.Payload[nodeResp.Clientid]; ok {
					ReflectConnMap.Payload[nodeResp.Clientid].Close()
					delete(ReflectConnMap.Payload, nodeResp.Clientid)
				}
				ReflectConnMap.Unlock()
				PortReflectMap.Lock()
				if _, ok := PortReflectMap.Payload[nodeResp.Clientid]; ok {
					if !utils.IsClosed(PortReflectMap.Payload[nodeResp.Clientid]) {
						close(PortReflectMap.Payload[nodeResp.Clientid])
						delete(PortReflectMap.Payload, nodeResp.Clientid)
					}
				}
				PortReflectMap.Unlock()
			case "REFLECTDATA":
				ReflectConnMap.RLock()
				if _, ok := ReflectConnMap.Payload[nodeResp.Clientid]; ok {
					PortReflectMap.Lock()
					if _, ok := PortReflectMap.Payload[nodeResp.Clientid]; ok {
						PortReflectMap.Payload[nodeResp.Clientid] <- nodeResp.Info
					} else {
						PortReflectMap.Payload[nodeResp.Clientid] = make(chan string, 10)
						go HandleReflect(startNodeConn, PortReflectMap.Payload[nodeResp.Clientid], nodeResp.Clientid, nodeResp.CurrentId)
						PortReflectMap.Payload[nodeResp.Clientid] <- nodeResp.Info
					}
					PortReflectMap.Unlock()
				}
				ReflectConnMap.RUnlock()
			case "KEEPALIVE":
			}
		case "COMMAND":
			switch nodeResp.Command {
			case "NEW":
				nodeid := GenerateNodeID() //生成一个新的nodeid号进行分配
				log.Println("[*]New node join! Node Id is ", len(CurrentClient))
				AdminStatus.NodesReadyToadd <- map[string]string{nodeid: nodeResp.Info} //将此节点加入detail命令所使用的NodeStatus.Nodes结构体
				NodeStatus.Nodenote[nodeid] = ""                                        //初始的note置空
				AddNodeToTopology(nodeid, nodeResp.CurrentId)                           //加入拓扑
				CalRoute()                                                              //计算路由
				Route.Lock()
				respCommand, _ := utils.ConstructPayload(nodeid, Route.Route[nodeid], "COMMAND", "ID", " ", " ", 0, utils.AdminId, AdminStatus.AESKey, false)
				Route.Unlock()
				startNodeConn.Write(respCommand)
			case "AGENTOFFLINE":
				log.Println("[*]Node ", FindIntByNodeid(nodeResp.Info)+1, " seems offline") //有节点掉线后，将此节点及其之后的节点删除
				CloseAll(nodeResp.Info)                                                     //清除一切与此节点及其子节点有关的连接及功能
				<-WaitForFindAll
				DelNodeFromTopology(nodeResp.Info) //从拓扑中删除
				//这里不用重新计算路由，因为控制端已经不会允许已掉线的节点及其子节点的流量流通
				if AdminStatus.HandleNode == nodeResp.Info && *CliStatus != "admin" { //如果admin端正好操控此节点
					AdminStuff.AdminCommandChan <- []string{"exit"}
					<-AdminStatus.ReadyChange
					<-AdminStatus.IsShellMode
				}
			case "MYINFO": //拆分节点发送上来的节点自身信息
				info := strings.Split(nodeResp.Info, ":::stowaway:::")
				hostname := strings.Replace(info[0], "\r", "", -1)
				username := strings.Replace(info[1], "\r", "", -1)
				NodeStatus.NodeHostname[nodeResp.CurrentId] = hostname
				NodeStatus.NodeUser[nodeResp.CurrentId] = username
			case "MYNOTE":
				NodeStatus.Nodenote[nodeResp.CurrentId] = nodeResp.Info
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
					AdminStatus.ReadyChange <- true
					AdminStatus.IsShellMode <- true
				}
			case "SSHTUNNELRESP":
				switch nodeResp.Info {
				case "SUCCESS":
					fmt.Println("[*]Successfully connect to node by ssh tunnel!")
					AdminStatus.ReadyChange <- true
					AdminStatus.IsShellMode <- true
				case "FAILED":
					fmt.Println("[*]Fail to connect to node by ssh tunnel! Something wrong is happened!")
					AdminStatus.ReadyChange <- true
					AdminStatus.IsShellMode <- true
				}
			case "SSHCERTERROR":
				fmt.Println("[*]Ssh certificate seems wrong")
				AdminStatus.SSHSuccess <- false
				AdminStatus.ReadyChange <- true
				AdminStatus.IsShellMode <- true
			case "NAMECONFIRM":
				AdminStatus.GetName <- true
			case "CREATEFAIL":
				AdminStatus.GetName <- false
			case "FILENAME":
				var err error
				UploadFile, err := os.Create(nodeResp.Info)
				Route.Lock()
				if err != nil {
					respComm, _ := utils.ConstructPayload(CurrentNode, Route.Route[CurrentNode], "COMMAND", "CREATEFAIL", " ", " ", 0, utils.AdminId, AdminStatus.AESKey, false)
					startNodeConn.Write(respComm)
				} else {
					respComm, _ := utils.ConstructPayload(CurrentNode, Route.Route[CurrentNode], "COMMAND", "NAMECONFIRM", " ", " ", 0, utils.AdminId, AdminStatus.AESKey, false)
					startNodeConn.Write(respComm)
					go share.ReceiveFile(Route.Route[CurrentNode], &startNodeConn, FileDataMap, AdminStatus.CannotRead, UploadFile, AdminStatus.AESKey, true, utils.AdminId)
				}
				Route.Unlock()
			case "FILESIZE":
				share.File.FileSize, _ = strconv.ParseInt(nodeResp.Info, 10, 64)
				Route.Lock()
				respComm, _ := utils.ConstructPayload(CurrentNode, Route.Route[CurrentNode], "COMMAND", "FILESIZECONFIRM", " ", " ", 0, utils.AdminId, AdminStatus.AESKey, false)
				Route.Unlock()
				startNodeConn.Write(respComm)
				share.File.ReceiveFileSize <- true
			case "FILESLICENUM":
				share.File.TotalSilceNum, _ = strconv.Atoi(nodeResp.Info)
				Route.Lock()
				respComm, _ := utils.ConstructPayload(CurrentNode, Route.Route[CurrentNode], "COMMAND", "FILESLICENUMCONFIRM", " ", " ", 0, utils.AdminId, AdminStatus.AESKey, false)
				Route.Unlock()
				startNodeConn.Write(respComm)
				share.File.ReceiveFileSliceNum <- true
			case "FILESLICENUMCONFIRM":
				share.File.TotalConfirm <- true
			case "FILESIZECONFIRM":
				share.File.TotalConfirm <- true
			case "FILENOTEXIST":
				fmt.Printf("[*]File %s not exist!\n", nodeResp.Info)
			case "CANNOTREAD":
				fmt.Printf("[*]File %s cannot be read!\n", nodeResp.Info)
				AdminStatus.CannotRead <- true
				share.File.ReceiveFileSliceNum <- false
				os.Remove(nodeResp.Info)
			case "CANNOTUPLOAD":
				fmt.Printf("[*]Agent cannot read file: %s\n", nodeResp.Info)
			case "GETREFLECTNUM":
				Route.Lock()
				AdminStuff.Lock()
				respComm, _ := utils.ConstructPayload(CurrentNode, Route.Route[CurrentNode], "COMMAND", "REFLECTNUM", " ", " ", AdminStuff.ReflectNum, utils.AdminId, AdminStatus.AESKey, false)
				AdminStuff.ReflectNum++
				AdminStuff.Unlock()
				Route.Unlock()
				startNodeConn.Write(respComm)
			case "RECONNID":
				log.Println("[*]Node reconnect successfully!")
				ipaddress, uppernode := AnalysisInfo(nodeResp.Info)
				AdminStatus.NodesReadyToadd <- map[string]string{nodeResp.CurrentId: ipaddress}
				NodeStatus.Nodenote[nodeResp.CurrentId] = ""
				ReconnAddCurrentClient(nodeResp.CurrentId) //在节点reconn回来的时候要考虑多种情况，若admin是掉线过，可以直接append，若admin没有掉线过，那么就需要判断重连回来的节点序号是否在CurrentClient中，如果已经存在就不需要append
				AddNodeToTopology(nodeResp.CurrentId, uppernode)
				CalRoute()
			case "HEARTBEAT":
				hbcommpack, _ := utils.ConstructPayload(utils.StartNodeId, "", "COMMAND", "KEEPALIVE", " ", " ", 0, utils.AdminId, AdminStatus.AESKey, false)
				startNodeConn.Write(hbcommpack)
			case "TRANSSUCCESS":
				fmt.Println("[*]File transmission complete!")
			case "FORWARDFAIL":
				fmt.Println("[*]Remote port seems down,port forward failed!")
				ForwardStatus.ForwardIsValid <- false
			case "FORWARDOK":
				fmt.Println("[*]Port forward successfully started!")
				ForwardStatus.ForwardIsValid <- true
			case "REFLECTFAIL":
				fmt.Println("[*]Agent seems cannot listen this port,port reflect failed!")
			case "REFLECTOK":
				fmt.Println("[*]Port reflect successfully started!")
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
				log.Println("[*]Unknown Command")
				continue
			}
		}
	}
}
