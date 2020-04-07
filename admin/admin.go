package admin

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"

	"Stowaway/common"

	"github.com/urfave/cli/v2"
)

var (
	CliStatus      *string
	NodeIdAllocate uint32
	AdminStatus    *common.AdminStatus
	FileDataMap    *common.IntStrMap
	ClientSockets  *common.Uint32ConnMap
	PortForWardMap *common.Uint32ConnMap
)

//启动admin
func NewAdmin(c *cli.Context) {
	var InitStatus string = "admin"
	NodeIdAllocate = 1
	AdminStatus = common.NewAdminStatus()
	ClientSockets = common.NewUint32ConnMap()
	FileDataMap = common.NewIntStrMap()
	PortForWardMap = common.NewUint32ConnMap()
	AdminStatus.AESKey = []byte(c.String("secret"))
	listenPort := c.String("listen")
	startnodeaddr := c.String("connect")

	Banner()

	if len(AdminStatus.AESKey) != 0 {
		log.Println("[*]Now Connection is encrypting with secret ", c.String("secret"))
	} else {
		log.Println("[*]Now Connection is maintianed without any encryption!")
	}
	if startnodeaddr == "" {
		go StartListen(listenPort)
	} else {
		ConnectToStartNode(startnodeaddr)
	}
	go AddToChain()
	CliStatus = &InitStatus
	Controlpanel()
}

func ConnectToStartNode(startnodeaddr string) {
	startNodeConn, err := net.Dial("tcp", startnodeaddr)
	if err != nil {
		log.Println("[*]Connection refused!")
		os.Exit(1)
	}
	helloMess, _ := common.ConstructPayload(1, "", "COMMAND", "STOWAWAYADMIN", " ", " ", 0, 0, AdminStatus.AESKey, false)
	startNodeConn.Write(helloMess)
	for {
		command, _ := common.ExtractPayload(startNodeConn, AdminStatus.AESKey, 0, true)
		switch command.Command {
		case "INIT":
			respCommand, _ := common.ConstructPayload(1, "", "COMMAND", "ID", " ", " ", 0, 0, AdminStatus.AESKey, false)
			startNodeConn.Write(respCommand)
			AdminStuff.StartNode = strings.Split(startNodeConn.RemoteAddr().String(), ":")[0]
			log.Printf("[*]Connect to startnode %s successfully!\n", startNodeConn.RemoteAddr().String())
			NodeStatus.Nodenote[1] = ""
			AddNodeToTopology(1, 0)
			CalRoute()
			go HandleStartConn(startNodeConn)
			go HandleCommandToControlConn(startNodeConn)
			go MonitorCtrlC(startNodeConn)
			return
		}
	}
}

//启动监听
func StartListen(listenPort string) {
	localAddr := fmt.Sprintf("0.0.0.0:%s", listenPort)
	localListener, err := net.Listen("tcp", localAddr)
	if err != nil {
		log.Printf("[*]Cannot listen %s", localAddr)
		os.Exit(1)
	}
	for {
		startNodeConn, _ := localListener.Accept() //一定要有连接进入才可继续操作，故没有连接时，admin端无法操作
		log.Printf("[*]StartNode connected from %s!\n", startNodeConn.RemoteAddr().String())
		AdminStuff.StartNode = strings.Split(startNodeConn.RemoteAddr().String(), ":")[0]
		go HandleInitControlConn(startNodeConn)
		break
	}
}

// 初始化与startnode的连接
func HandleInitControlConn(startNodeConn net.Conn) {
	for {
		command, err := common.ExtractPayload(startNodeConn, AdminStatus.AESKey, 0, true)
		if err != nil {
			log.Println("[*]Startnode seems offline, control channel set up failed.Exiting...")
			return
		}
		switch command.Command {
		case "STOWAWAYAGENT":
			Message, _ := common.ConstructPayload(0, "", "COMMAND", "CONFIRM", " ", " ", 0, 0, AdminStatus.AESKey, false)
			startNodeConn.Write(Message)
		case "INIT":
			respCommand, _ := common.ConstructPayload(1, "", "COMMAND", "ID", " ", " ", 0, 0, AdminStatus.AESKey, false)
			startNodeConn.Write(respCommand)
			NodeStatus.Nodenote[1] = ""
			AddNodeToTopology(1, 0)
			CalRoute()
			go HandleStartConn(startNodeConn)
			go HandleCommandToControlConn(startNodeConn)
			go MonitorCtrlC(startNodeConn)
			return
		}
	}
}

// 处理与startnode的信道
func HandleStartConn(startNodeConn net.Conn) {
	for {
		nodeResp, err := common.ExtractPayload(startNodeConn, AdminStatus.AESKey, 0, true)
		if err != nil {
			log.Println("[*]StartNode seems offline")
			DelNodeFromTopology(1)
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
				clientnum, _ := strconv.ParseInt(nodeResp.Info, 10, 32)
				client := uint32(clientnum)
				Route.Lock()
				respCommand, _ := common.ConstructPayload(client, Route.Route[client], "DATA", "FINOK", " ", " ", nodeResp.Clientid, 0, AdminStatus.AESKey, false)
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
					if !common.IsClosed(PortReflectMap.Payload[nodeResp.Clientid]) {
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
						tempchan := make(chan string, 10)
						PortReflectMap.Payload[nodeResp.Clientid] = tempchan
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
				log.Println("[*]New node join! Node Id is ", NodeIdAllocate+1)
				AdminStatus.NodesReadyToadd <- map[uint32]string{NodeIdAllocate + 1: nodeResp.Info}
				NodeStatus.Nodenote[NodeIdAllocate+1] = ""
				NodeIdAllocate++
				AddNodeToTopology(NodeIdAllocate, nodeResp.CurrentId)
				CalRoute()
				Route.Lock()
				respCommand, _ := common.ConstructPayload(NodeIdAllocate, Route.Route[NodeIdAllocate], "COMMAND", "ID", " ", " ", 0, 0, AdminStatus.AESKey, false)
				Route.Unlock()
				startNodeConn.Write(respCommand)
			case "AGENTOFFLINE":
				offlineNode := common.StrUint32(nodeResp.Info)
				log.Println("[*]Node ", offlineNode, " seems offline") //有节点掉线后，将此节点及其之后的节点删除
				CloseAll(offlineNode)
				<-WaitForFindAll
				DelNodeFromTopology(offlineNode)
				if AdminStatus.HandleNode == offlineNode {
					AdminStuff.AdminCommandChan <- []string{"exit"}
					<-AdminStatus.ReadyChange
					<-AdminStatus.IsShellMode
				}
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
					AdminStatus.SshSuccess <- true
					fmt.Println("[*]Node start ssh successfully!")
				case "FAILED":
					AdminStatus.SshSuccess <- false
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
				AdminStatus.SshSuccess <- false
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
					respComm, _ := common.ConstructPayload(CurrentNode, Route.Route[CurrentNode], "COMMAND", "CREATEFAIL", " ", " ", 0, 0, AdminStatus.AESKey, false)
					startNodeConn.Write(respComm)
				} else {
					var tempchan *net.Conn = &startNodeConn
					respComm, _ := common.ConstructPayload(CurrentNode, Route.Route[CurrentNode], "COMMAND", "NAMECONFIRM", " ", " ", 0, 0, AdminStatus.AESKey, false)
					startNodeConn.Write(respComm)
					go common.ReceiveFile(Route.Route[CurrentNode], tempchan, FileDataMap, AdminStatus.CannotRead, UploadFile, AdminStatus.AESKey, true, 0)
				}
				Route.Unlock()
			case "FILESIZE":
				filesize, _ := strconv.ParseInt(nodeResp.Info, 10, 64)
				common.File.FileSize = filesize
				Route.Lock()
				respComm, _ := common.ConstructPayload(CurrentNode, Route.Route[CurrentNode], "COMMAND", "FILESIZECONFIRM", " ", " ", 0, 0, AdminStatus.AESKey, false)
				Route.Unlock()
				startNodeConn.Write(respComm)
				common.File.ReceiveFileSize <- true
			case "FILESLICENUM":
				common.File.TotalSilceNum, _ = strconv.Atoi(nodeResp.Info)
				Route.Lock()
				respComm, _ := common.ConstructPayload(CurrentNode, Route.Route[CurrentNode], "COMMAND", "FILESLICENUMCONFIRM", " ", " ", 0, 0, AdminStatus.AESKey, false)
				Route.Unlock()
				startNodeConn.Write(respComm)
				common.File.ReceiveFileSliceNum <- true
			case "FILESLICENUMCONFIRM":
				common.File.TotalConfirm <- true
			case "FILESIZECONFIRM":
				common.File.TotalConfirm <- true
			case "FILENOTEXIST":
				fmt.Printf("[*]File %s not exist!\n", nodeResp.Info)
			case "CANNOTREAD":
				fmt.Printf("[*]File %s cannot be read!\n", nodeResp.Info)
				AdminStatus.CannotRead <- true
				common.File.ReceiveFileSliceNum <- false
				os.Remove(nodeResp.Info)
			case "CANNOTUPLOAD":
				fmt.Printf("[*]Agent cannot read file: %s\n", nodeResp.Info)
			case "GETREFLECTNUM":
				Route.Lock()
				AdminStuff.Lock()
				respComm, _ := common.ConstructPayload(CurrentNode, Route.Route[CurrentNode], "COMMAND", "REFLECTNUM", " ", " ", AdminStuff.ReflectNum, 0, AdminStatus.AESKey, false)
				AdminStuff.ReflectNum++
				AdminStuff.Unlock()
				Route.Unlock()
				startNodeConn.Write(respComm)
			case "RECONNID":
				log.Println("[*]Node ", nodeResp.CurrentId, " reconnect successfully!")
				ipaddress, uppernode := AnalysisInfo(nodeResp.Info)
				AdminStatus.NodesReadyToadd <- map[uint32]string{nodeResp.CurrentId: ipaddress}
				AddNodeToTopology(nodeResp.CurrentId, uppernode)
				CalRoute()
				FindMax()
			case "HEARTBEAT":
				hbcommpack, _ := common.ConstructPayload(1, "", "COMMAND", "KEEPALIVE", " ", " ", 0, 0, AdminStatus.AESKey, false)
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
			default:
				log.Println("[*]Unknown Command")
				continue
			}
		}
	}
}
