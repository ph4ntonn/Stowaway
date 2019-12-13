package admin

import (
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"

	"Stowaway/common"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var CliStatus *string
var InitStatus string = "admin"
var ReadyChange = make(chan bool)
var IsShellMode = make(chan bool)
var NodeSocksStarted = make(chan bool, 1)
var SocksRespChan = make(chan string, 1)
var StartNode [1]string = [1]string{"0.0.0.0"}
var NodesReadyToadd = make(chan map[uint32]string)
var CurrentDir string
var StartNodeStatus string

func NewAdmin(c *cli.Context) error {
	//commSecret := c.String("secret")
	listenPort := c.String("listen")
	//ccPort := c.String("control")
	// go StartListen(listenPort)
	Banner()
	go StartListen(listenPort)
	go AddToChain()
	CliStatus = &InitStatus
	Controlpanel()
	return nil
}

func StartListen(listenPort string) error {
	localAddr := fmt.Sprintf("0.0.0.0:%s", listenPort)
	localListener, err := net.Listen("tcp", localAddr)
	if err != nil {
		return fmt.Errorf("Cannot listen %s", localAddr)
	}
	for {
		conn, err := localListener.Accept()
		startNodeIP := strings.Split(conn.RemoteAddr().String(), ":")[0]
		if err != nil {
			return fmt.Errorf("Cannot read data from socket")
		}
		if startNodeIP == StartNode[0] && StartNode[0] != "0.0.0.0" {
			logrus.Printf("StartNode connected from %s!\n", conn.RemoteAddr().String())
			go HandleDataConn(conn)
		} else if startNodeIP != StartNode[0] && StartNode[0] == "0.0.0.0" {
			//logrus.Printf("StartNode connected from %s!\n", conn.RemoteAddr().String())
			go HandleInitControlConn(conn)
		} else {
		}
	}
}

func HandleInitControlConn(startNodeControlConn net.Conn) {
	for {
		command, err := common.ExtractCommand(startNodeControlConn)
		switch command.Command {
		case "INIT":
			switch command.Info {
			case "FIRSTCONNECT":
				respCommand, err := common.ConstructCommand("ACCEPT", "DATA", 1)
				StartNode[0] = strings.Split(startNodeControlConn.RemoteAddr().String(), ":")[0]
				_, err = startNodeControlConn.Write(respCommand)
				if err != nil {
					logrus.Errorf("ERROR OCCURED!: %s", err)
				}
				go HandleCommandFromControlConn(startNodeControlConn)
				go HandleCommandToControlConn(startNodeControlConn)
				go MonitorCtrlC(startNodeControlConn)
			}
		case "LISTENPORT":
			StartNodeStatus = command.Info
			return
		}
		if err != nil {
			logrus.Error(err)
			continue
		}
	}
}

func HandleDataConn(startNodeDataConn net.Conn) {
	for {
		nodeResp, err := common.ExtractDataResult(startNodeDataConn)
		if err != nil {
			logrus.Error("StartNode seems offline")
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
		}
	}
}

func HandleCommandToControlConn(startNodeControlConn net.Conn) {
	for {
		AdminCommand := <-ADMINCOMMANDCHAN
		switch AdminCommand[0] {
		case "use":
			if len(AdminCommand) == 2 {
				if StartNode[0] == "0.0.0.0" {
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

func HandleCommandFromControlConn(startNodeControlConn net.Conn) { //处理由startnode proxy过来的lower node 回送命令
	for {
		command, _ := common.ExtractCommand(startNodeControlConn)
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
				fmt.Println("[*]Node start ssh successfully!")
			case "FAILED":
				fmt.Println("[*]Node start ssh failed!")
			}
		}

	}
}

func StartSocksService(command []string) {
	socksport := command[1]
	checkport, _ := strconv.Atoi(socksport)
	if checkport <= 0 || checkport > 65535 {
		logrus.Error("Port Illegal!")
		return
	}

	socks5Addr := fmt.Sprintf("0.0.0.0:%s", socksport)
	localListener, err := net.Listen("tcp", socks5Addr)
	if err != nil {
		logrus.Error("Cannot listen this port!")
		return
	}
	for {
		conn, _ := localListener.Accept()
		go ProxySocksToclient(conn, socksport)
	}
}

func ProxySocksToclient(client net.Conn, targetport string) {
	temp, _ := strconv.Atoi(targetport)
	targetport = strconv.Itoa(temp + 1)
	socksAddr := fmt.Sprintf("%s:%s", Nodes[1], targetport)
	socksproxyconn, err := net.Dial("tcp", socksAddr)
	if err != nil {
		logrus.Error("Cannot connect to socks server")
	}
	go io.Copy(client, socksproxyconn)
	io.Copy(socksproxyconn, client)
	defer client.Close()
	defer socksproxyconn.Close()
}

func StartSSHService(startNodeControlConn net.Conn, info []string, nodeid uint32) {
	information := fmt.Sprintf("%s::%s::%s", info[1], info[2], info[3])
	sshCommand, _ := common.ConstructCommand("SSH", information, nodeid)
	startNodeControlConn.Write(sshCommand)

}
