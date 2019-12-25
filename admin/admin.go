package admin

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"

	"Stowaway/common"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

type SafeMap struct {
	sync.RWMutex
	ClientSocketsMap map[uint32]net.Conn
}

var (
	CliStatus       *string
	StartNodeStatus string
	InitStatus      string = "admin"
	StartNode       string = "0.0.0.0"

	ReadyChange      = make(chan bool)
	IsShellMode      = make(chan bool)
	SSHSUCCESS       = make(chan bool, 1)
	NodeSocksStarted = make(chan bool, 1)
	SocksRespChan    = make(chan string, 1)
	NodesReadyToadd  = make(chan map[uint32]string)
	ClientSockets    *SafeMap

	ClientNum uint32 = 0
	AESKey    []byte

	DataConn               net.Conn
	SocksListenerForClient net.Listener
)

//ClientSockets.ClientSocketsMap = make(map[uint32]net.Conn)
//启动admin
func NewAdmin(c *cli.Context) error {
	ClientSockets = newSafeMap()
	AESKey = []byte(c.String("secret"))
	listenPort := c.String("listen")
	//ccPort := c.String("control")
	Banner()
	go StartListen(listenPort)
	go AddToChain()
	CliStatus = &InitStatus
	Controlpanel()
	return nil
}

func newSafeMap() *SafeMap {
	sm := new(SafeMap)
	sm.ClientSocketsMap = make(map[uint32]net.Conn)
	return sm
}

//启动监听
func StartListen(listenPort string) {
	localAddr := fmt.Sprintf("0.0.0.0:%s", listenPort)
	localListener, err := net.Listen("tcp", localAddr)
	if err != nil {
		logrus.Errorf("Cannot listen %s", localAddr)
		os.Exit(1)
	}
	for {
		conn, _ := localListener.Accept()                                //一定要有连接进入才可继续操作，故没有连接时，admin端无法操作
		startNodeIP := strings.Split(conn.RemoteAddr().String(), ":")[0] //记录一下startnode的ip，为数据信道建立作准备
		if startNodeIP == StartNode && StartNode != "0.0.0.0" {          //两次ip是否相同
			logrus.Printf("StartNode connected from %s!\n", conn.RemoteAddr().String())
			DataConn = conn
			go HandleDataConn(conn)
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
			StartNodeStatus = command.Info //获取一下agent端监听端口的信息
			respCommand, err := common.ConstructCommand("ACCEPT", "DATA", 1, AESKey)
			StartNode = strings.Split(startNodeControlConn.RemoteAddr().String(), ":")[0]
			_, err = startNodeControlConn.Write(respCommand)
			if err != nil {
				logrus.Errorf("Startnode seems offline, control channel set up failed.Exiting...")
				return
			}
			go HandleCommandFromControlConn(startNodeControlConn)
			go HandleCommandToControlConn(startNodeControlConn)
			go MonitorCtrlC(startNodeControlConn)
			return
		}
		if err != nil {
			logrus.Error(err)
			continue
		}
	}
}

// 处理与startnode的数据信道
func HandleDataConn(startNodeDataConn net.Conn) {
	for {
		nodeResp, err := common.ExtractDataResult(startNodeDataConn, AESKey, 0)
		if err != nil {
			logrus.Error("StartNode seems offline")
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
			if nodeResp.Success == "1" {
				fmt.Print(nodeResp.Result)
			} else {
				fmt.Println("Something wrong occured!Try another one")
			}
		case "SSHMESS":
			if nodeResp.Success == "1" {
				fmt.Print(nodeResp.Result)
				fmt.Print("(ssh mode)>>>")
			} else {
				fmt.Println("Something wrong occured!Try another one")
			}
		case "SOCKSDATARESP":
			ClientSockets.RLock()
			// fmt.Println("get response!", string(nodeResp.Result))
			_, err := ClientSockets.ClientSocketsMap[nodeResp.Clientsocks].Write([]byte(nodeResp.Result))
			if err != nil {
				ClientSockets.RUnlock()
				continue
			}
			ClientSockets.RUnlock()
		case "FIN":
			ClientSockets.RLock()
			ClientSockets.ClientSocketsMap[nodeResp.Clientsocks].Close()
			ClientSockets.RUnlock()
			clientnum, _ := strconv.ParseInt(nodeResp.Result, 10, 32)
			client := uint32(clientnum)
			respCommand, _ := common.ConstructDataResult(client, nodeResp.Clientsocks, " ", "FINOK", " ", AESKey, 0)
			startNodeDataConn.Write(respCommand)
		}
	}
}

// 处理由admin发往startnode的控制信号
func HandleCommandToControlConn(startNodeControlConn net.Conn) {
	for {
		AdminCommand := <-ADMINCOMMANDCHAN
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
			logrus.Info("BYE!")
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
			logrus.Info("New node join! Node Id is ", command.NodeId)
			NodesReadyToadd <- map[uint32]string{command.NodeId: command.Info}
		case "AGENTOFFLINE":
			logrus.Error("Node ", command.NodeId, " seems offline")
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
				SSHSUCCESS <- true
				fmt.Println("[*]Node start ssh successfully!")
			case "FAILED":
				SSHSUCCESS <- false
				fmt.Println("[*]Node start ssh failed!Check if target's ssh service is on or username and pass given are right")
				ReadyChange <- true
				IsShellMode <- true
			}
		default:
			logrus.Error("Unknown Command")
			continue
		}

	}
}

// 启动socks5 for client
func StartSocksServiceForClient(command []string, startNodeControlConn net.Conn, nodeID uint32) {
	var err error
	socksport := command[1]
	checkport, _ := strconv.Atoi(socksport)
	if checkport <= 0 || checkport > 65535 {
		logrus.Error("Port Illegal!")
		return
	}

	socks5Addr := fmt.Sprintf("0.0.0.0:%s", socksport)
	SocksListenerForClient, err = net.Listen("tcp", socks5Addr)
	if err != nil {
		respCommand, _ := common.ConstructCommand("SOCKSOFF", " ", nodeID, AESKey)
		_, err = startNodeControlConn.Write(respCommand)
		if err != nil {
			logrus.Error("Cannot stop agent's socks service,check the connection!")
		}
		logrus.Error("Cannot listen this port!")
		return
	}
	for {
		conn, err := SocksListenerForClient.Accept()
		if err != nil {
			logrus.Info("Socks service stoped")
			return
		}
		ClientSockets.Lock()
		ClientSockets.ClientSocketsMap[ClientNum] = conn
		ClientSockets.Unlock()
		ClientSockets.RLock()
		go HandleNewSocksConn(ClientSockets.ClientSocketsMap[ClientNum], ClientNum, nodeID)
		ClientSockets.RUnlock()
		ClientNum++
	}
}

func HandleNewSocksConn(clientsocks net.Conn, num uint32, nodeID uint32) {
	buffer := make([]byte, 204800)
	for {
		len, err := clientsocks.Read(buffer)
		if err != nil {
			clientsocks.Close()
			return
		} else {
			respData, _ := common.ConstructDataResult(nodeID, num, " ", "SOCKSDATA", string(buffer[:len]), AESKey, 0)
			DataConn.Write(respData)
		}
	}
}

// 发送ssh开启命令
func StartSSHService(startNodeControlConn net.Conn, info []string, nodeid uint32) {
	information := fmt.Sprintf("%s::%s::%s", info[1], info[2], info[3])
	sshCommand, _ := common.ConstructCommand("SSH", information, nodeid, AESKey)
	startNodeControlConn.Write(sshCommand)
}
