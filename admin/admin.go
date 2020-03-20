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
	CliStatus *string
	StartNode string = "0.0.0.0"

	ReadyChange      = make(chan bool, 1)
	IsShellMode      = make(chan bool, 1)
	SshSuccess       = make(chan bool, 1)
	NodeSocksStarted = make(chan bool, 1)
	GetName          = make(chan bool, 1)
	CannotRead       = make(chan bool, 1)
	Eof              = make(chan string, 1)
	NodesReadyToadd  = make(chan map[uint32]string)

	FileDataMap    *common.IntStrMap
	ClientSockets  *common.Uint32ConnMap
	PortForWardMap *common.Uint32ConnMap

	AESKey []byte

	DataConn               net.Conn
	SocksListenerForClient net.Listener
)

//启动admin
func NewAdmin(c *cli.Context) {
	var InitStatus string = "admin"
	ClientSockets = common.NewUint32ConnMap()
	FileDataMap = common.NewIntStrMap()
	PortForWardMap = common.NewUint32ConnMap()
	AESKey = []byte(c.String("secret"))
	listenPort := c.String("listen")
	startnodeaddr := c.String("connect")

	Banner()

	if len(AESKey) != 0 {
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
	controlConnToStartNode, err := net.Dial("tcp", startnodeaddr)
	if err != nil {
		log.Println("[*]Connection refused!")
		os.Exit(1)
	}
	for {
		command, _ := common.ExtractCommand(controlConnToStartNode, AESKey)
		switch command.Command {
		case "INIT":
			respCommand, _ := common.ConstructCommand("ID", "", 1, AESKey)
			controlConnToStartNode.Write(respCommand)
		case "IDOK":
			dataConnToStartNode, err := net.Dial("tcp", startnodeaddr)
			if err != nil {
				log.Println("[*]Connection refused!")
				os.Exit(1)
			}
			DataConn = dataConnToStartNode
			StartNode = strings.Split(controlConnToStartNode.RemoteAddr().String(), ":")[0]
			log.Printf("[*]Connect to startnode %s successfully!\n", controlConnToStartNode.RemoteAddr().String())
			go HandleDataConn(dataConnToStartNode)
			go common.SendHeartBeatData(dataConnToStartNode, 1, AESKey)
			go HandleCommandFromControlConn(controlConnToStartNode)
			go HandleCommandToControlConn(controlConnToStartNode)
			go MonitorCtrlC(controlConnToStartNode)
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
		conn, _ := localListener.Accept()                                //一定要有连接进入才可继续操作，故没有连接时，admin端无法操作
		startNodeIP := strings.Split(conn.RemoteAddr().String(), ":")[0] //记录一下startnode的ip，为数据信道建立作准备
		if startNodeIP == StartNode && StartNode != "0.0.0.0" {          //两次ip是否相同
			log.Printf("[*]StartNode connected from %s!\n", conn.RemoteAddr().String())
			DataConn = conn
			go HandleDataConn(conn)
			go common.SendHeartBeatData(conn, 1, AESKey)
		} else if startNodeIP != StartNode && StartNode == "0.0.0.0" {
			go HandleInitControlConn(conn)
		}
	}
}

// 初始化与startnode的连接
func HandleInitControlConn(startNodeControlConn net.Conn) {
	for {
		command, err := common.ExtractCommand(startNodeControlConn, AESKey)
		switch command.Command {
		case "INIT":
			respCommand, err := common.ConstructCommand("ACCEPT", "DATA", 1, AESKey)
			StartNode = strings.Split(startNodeControlConn.RemoteAddr().String(), ":")[0]
			_, err = startNodeControlConn.Write(respCommand)
			if err != nil {
				log.Println("[*]Startnode seems offline, control channel set up failed.Exiting...")
				return
			}
			go HandleCommandFromControlConn(startNodeControlConn)
			go HandleCommandToControlConn(startNodeControlConn)
			go MonitorCtrlC(startNodeControlConn)
			return
		}
		if err != nil {
			log.Println("[*]", err)
			continue
		}
	}
}

// 处理与startnode的数据信道
func HandleDataConn(startNodeDataConn net.Conn) {
	for {
		nodeResp, err := common.ExtractDataResult(startNodeDataConn, AESKey, 0)
		if err != nil {
			log.Println("[*]StartNode seems offline")
			for Nodeid, _ := range Nodes {
				if Nodeid >= 1 {
					delete(Nodes, Nodeid)
				}
			}
			StartNode = "offline"
			break
		}
		switch nodeResp.Datatype {
		case "SHELLRESP":
			fmt.Print(nodeResp.Result)
		case "SSHMESS":
			fmt.Print(nodeResp.Result)
			fmt.Print("(ssh mode)>>>")
		case "SOCKSDATARESP":
			ClientSockets.RLock()
			// fmt.Println("get response!", string(nodeResp.Result))
			if _, ok := ClientSockets.Payload[nodeResp.Clientid]; ok {
				_, err := ClientSockets.Payload[nodeResp.Clientid].Write([]byte(nodeResp.Result))
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
			clientnum, _ := strconv.ParseInt(nodeResp.Result, 10, 32)
			client := uint32(clientnum)
			respCommand, _ := common.ConstructDataResult(client, nodeResp.Clientid, " ", "FINOK", " ", AESKey, 0)
			startNodeDataConn.Write(respCommand)
		case "FILEDATA": //接收文件内容
			slicenum, _ := strconv.Atoi(nodeResp.FileSliceNum)
			FileDataMap.Payload[slicenum] = nodeResp.Result
		case "EOF": //文件读取结束
			Eof <- nodeResp.FileSliceNum
		case "FORWARDDATARESP":
			PortForWardMap.Payload[nodeResp.Clientid].Write([]byte(nodeResp.Result))
		case "REFLECTTIMEOUT":
			fallthrough
		case "FORWARDOFFLINE":
			PortForWardMap.Lock()
			if _, ok := PortForWardMap.Payload[nodeResp.Clientid]; ok {
				PortForWardMap.Payload[nodeResp.Clientid].Close()
				delete(PortForWardMap.Payload, nodeResp.Clientid)
			}
			PortForWardMap.Unlock()
		case "REFLECT":
			TryReflect(DataConn, nodeResp.CurrentId, nodeResp.Clientid, nodeResp.Result)
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
				}
			}
			PortReflectMap.Unlock()
		case "REFLECTDATA":
			ReflectConnMap.RLock()
			if _, ok := ReflectConnMap.Payload[nodeResp.Clientid]; ok {
				PortReflectMap.Lock()
				if _, ok := PortReflectMap.Payload[nodeResp.Clientid]; ok {
					PortReflectMap.Payload[nodeResp.Clientid] <- nodeResp.Result
				} else {
					tempchan := make(chan string, 10)
					PortReflectMap.Payload[nodeResp.Clientid] = tempchan
					go HandleReflect(DataConn, PortReflectMap.Payload[nodeResp.Clientid], nodeResp.Clientid, nodeResp.CurrentId)
					PortReflectMap.Payload[nodeResp.Clientid] <- nodeResp.Result
				}
				PortReflectMap.Unlock()
			}
			ReflectConnMap.RUnlock()
		case "KEEPALIVE":
		}
	}
}

// 处理admin模式下用户的输入及由admin发往startnode的控制信号
func HandleCommandToControlConn(startNodeControlConn net.Conn) {
	for {
		AdminCommand := <-AdminCommandChan
		switch AdminCommand[0] {
		case "use":
			if len(AdminCommand) == 2 {
				if StartNode == "0.0.0.0" {
					fmt.Println("There are no nodes connected!")
					ReadyChange <- true
					IsShellMode <- true
				} else if AdminCommand[1] == "1" {
					*CliStatus = "startnode"
					ReadyChange <- true
					IsShellMode <- true
					HandleNodeCommand(startNodeControlConn, AdminCommand[1])
				} else {
					if len(Nodes) == 0 {
						fmt.Println("There is no node", AdminCommand[1])
						ReadyChange <- true
						IsShellMode <- true
					} else {
						key, _ := strconv.ParseInt(AdminCommand[1], 10, 32)
						if _, ok := Nodes[uint32(key)]; ok {
							*CliStatus = "node " + AdminCommand[1]
							ReadyChange <- true
							IsShellMode <- true
							HandleNodeCommand(startNodeControlConn, AdminCommand[1])
						} else {
							fmt.Println("There is no node", AdminCommand[1])
							ReadyChange <- true
							IsShellMode <- true
						}
					}
				}
			} else {
				fmt.Println("Bad format!")
				ReadyChange <- true
				IsShellMode <- true
			}
		case "chain":
			ShowChain()
			ReadyChange <- true
			IsShellMode <- true
		case "help":
			ShowMainHelp()
			ReadyChange <- true
			IsShellMode <- true
		case "":
			ReadyChange <- true
			IsShellMode <- true
			continue
		case "exit":
			log.Println("[*]BYE!")
			SendOffLineToStartNode(startNodeControlConn)
			os.Exit(0)
			return
		default:
			fmt.Println("Illegal command, enter help to get available commands")
			ReadyChange <- true
			IsShellMode <- true
		}
	}
}

//处理由startnode proxy过来的lower node 回送命令（包括startnode本身）
func HandleCommandFromControlConn(startNodeControlConn net.Conn) {
	for {
		command, err := common.ExtractCommand(startNodeControlConn, AESKey)
		if err != nil {
			startNodeControlConn.Close() // startnode下线，关闭conn，防止死循环导致cpu占用过高
			break
		}
		switch command.Command {
		case "NEW":
			log.Println("[*]New node join! Node Id is ", command.NodeId)
			NodesReadyToadd <- map[uint32]string{command.NodeId: command.Info}
		case "AGENTOFFLINE":
			log.Println("[*]Node ", command.NodeId, " seems offline") //有节点掉线后，将此节点及其之后的节点删除
			for Nodeid, _ := range Nodes {
				if Nodeid >= command.NodeId {
					delete(Nodes, Nodeid)
				}
			}
		case "SOCKSRESP":
			switch command.Info {
			case "SUCCESS":
				fmt.Println("[*]Node start socks5 successfully!")
				NodeSocksStarted <- true
			case "FAILED":
				fmt.Println("[*]Node start socks5 failed!")
				NodeSocksStarted <- false
			}
		case "SSHRESP":
			switch command.Info {
			case "SUCCESS":
				SshSuccess <- true
				fmt.Println("[*]Node start ssh successfully!")
			case "FAILED":
				SshSuccess <- false
				fmt.Println("[*]Node start ssh failed!Check if target's ssh service is on or username and pass given are right")
				ReadyChange <- true
				IsShellMode <- true
			}
		case "NAMECONFIRM":
			GetName <- true
		case "CREATEFAIL":
			GetName <- false
		case "FILENAME":
			var err error
			UploadFile, err := os.Create(command.Info)
			if err != nil {
				respComm, _ := common.ConstructCommand("CREATEFAIL", "", CurrentNode, AESKey) //从控制信道上返回文件是否能成功创建的响应
				startNodeControlConn.Write(respComm)
			} else {
				var tempchan *net.Conn = &startNodeControlConn
				respComm, _ := common.ConstructCommand("NAMECONFIRM", "", CurrentNode, AESKey)
				startNodeControlConn.Write(respComm)
				go common.ReceiveFile(tempchan, Eof, FileDataMap, CannotRead, UploadFile, AESKey, true)
			}
		case "FILENOTEXIST":
			fmt.Printf("File %s not exist!\n", command.Info)
		case "CANNOTREAD":
			fmt.Printf("File %s cannot be read!\n", command.Info)
			CannotRead <- true
		case "RECONNID":
			log.Println("[*]New node join! Node Id is ", command.NodeId)
			NodesReadyToadd <- map[uint32]string{command.NodeId: command.Info}
		case "HEARTBEAT":
			hbcommpack, _ := common.ConstructCommand("KEEPALIVE", "", 1, AESKey)
			startNodeControlConn.Write(hbcommpack)
		case "TRANSSUCCESS":
			fmt.Println("File transmission complete!")
		case "FORWARDFAIL":
			fmt.Println("[*]Remote port seems down,port forward failed!")
			ForwardIsValid <- false
		case "FORWARDOK":
			fmt.Println("[*]Port forward successfully started!")
			ForwardIsValid <- true
		case "REFLECTFAIL":
			fmt.Println("[*]Agent seems cannot listen this port,port reflect failed!")
		case "REFLECTOK":
			fmt.Println("[*]Port reflect successfully started!")
		default:
			log.Println("[*]Unknown Command")
			continue
		}

	}
}
