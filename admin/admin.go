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

	AdminStatus    *common.AdminStatus
	FileDataMap    *common.IntStrMap
	ClientSockets  *common.Uint32ConnMap
	PortForWardMap *common.Uint32ConnMap
)

//启动admin
func NewAdmin(c *cli.Context) {
	var InitStatus string = "admin"
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
	for {
		command, _ := common.ExtractPayload(startNodeConn, AdminStatus.AESKey, 0, true)
		switch command.Command {
		case "INIT":
			respCommand, _ := common.ConstructPayload(1, "COMMAND", "ID", " ", " ", 0, 0, AdminStatus.AESKey, false)
			startNodeConn.Write(respCommand)
			AdminStuff.StartNode = strings.Split(startNodeConn.RemoteAddr().String(), ":")[0]
			log.Printf("[*]Connect to startnode %s successfully!\n", startNodeConn.RemoteAddr().String())
			NodeStatus.Nodenote[1] = ""
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
		case "INIT":
			respCommand, _ := common.ConstructPayload(1, "COMMAND", "ID", " ", " ", 0, 0, AdminStatus.AESKey, false)
			startNodeConn.Write(respCommand)
			NodeStatus.Nodenote[1] = ""
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
			for Nodeid, _ := range NodeStatus.Nodes {
				if Nodeid >= 1 {
					delete(NodeStatus.Nodes, Nodeid)
				}
			}
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
				respCommand, _ := common.ConstructPayload(client, "DATA", "FINOK", " ", " ", nodeResp.Clientid, 0, AdminStatus.AESKey, false)
				startNodeConn.Write(respCommand)
			case "FILEDATA": //接收文件内容
				slicenum, _ := strconv.Atoi(nodeResp.FileSliceNum)
				FileDataMap.Payload[slicenum] = nodeResp.Info
			case "EOF": //文件读取结束
				AdminStatus.EOF <- nodeResp.FileSliceNum
			case "FORWARDDATARESP":
				PortForWardMap.Payload[nodeResp.Clientid].Write([]byte(nodeResp.Info))
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
				log.Println("[*]New node join! Node Id is ", nodeResp.CurrentId+1)
				AdminStatus.NodesReadyToadd <- map[uint32]string{nodeResp.CurrentId + 1: nodeResp.Info}
				NodeStatus.Nodenote[nodeResp.CurrentId+1] = ""
			case "AGENTOFFLINE":
				log.Println("[*]Node ", nodeResp.CurrentId+1, " seems offline") //有节点掉线后，将此节点及其之后的节点删除
				for Nodeid, _ := range NodeStatus.Nodes {
					if Nodeid >= nodeResp.CurrentId+1 {
						delete(NodeStatus.Nodes, Nodeid)
					}
				}
				log.Println("[*]All agents' socks,reflect,forward service down! Please restart these services manually")
				CloseAll()
				log.Println("[*]If this node reconnect sometime. DON'T forget to use command(recover) in its upper node first before use the reconnected node!")
			case "SOCKSRESP":
				switch nodeResp.Info {
				case "SUCCESS":
					fmt.Println("[*]Node start socks5 successfully!")
					AdminStatus.NodeSocksStarted <- true
				case "FAILED":
					fmt.Println("[*]Node start socks5 failed!")
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
			case "NAMECONFIRM":
				AdminStatus.GetName <- true
			case "CREATEFAIL":
				AdminStatus.GetName <- false
			case "FILENAME":
				var err error
				UploadFile, err := os.Create(nodeResp.Info)
				if err != nil {
					respComm, _ := common.ConstructPayload(CurrentNode, "COMMAND", "CREATEFAIL", " ", " ", 0, 0, AdminStatus.AESKey, false)
					startNodeConn.Write(respComm)
				} else {
					var tempchan *net.Conn = &startNodeConn
					respComm, _ := common.ConstructPayload(CurrentNode, "COMMAND", "NAMECONFIRM", " ", " ", 0, 0, AdminStatus.AESKey, false)
					startNodeConn.Write(respComm)
					go common.ReceiveFile(tempchan, AdminStatus.EOF, FileDataMap, AdminStatus.CannotRead, UploadFile, AdminStatus.AESKey, true, 0)
				}
			case "FILENOTEXIST":
				fmt.Printf("File %s not exist!\n", nodeResp.Info)
			case "CANNOTREAD":
				fmt.Printf("File %s cannot be read!\n", nodeResp.Info)
				AdminStatus.CannotRead <- true
			case "RECONNID":
				log.Println("[*]New node join! Node Id is ", nodeResp.CurrentId)
				AdminStatus.NodesReadyToadd <- map[uint32]string{nodeResp.CurrentId: nodeResp.Info}
			case "HEARTBEAT":
				hbcommpack, _ := common.ConstructPayload(1, "COMMAND", "KEEPALIVE", " ", " ", 0, 0, AdminStatus.AESKey, false)
				startNodeConn.Write(hbcommpack)
			case "TRANSSUCCESS":
				fmt.Println("File transmission complete!")
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
