package agent

import (
	"Stowaway/common"
	"Stowaway/node"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var ConnectedNode string = "0.0.0.0"
var Monitor string
var ListenPort string

var CommandToUpperNodeChan = make(chan []byte)
var cmdResult = make(chan []byte)
var PROXY_COMMAND_CHAN = make(chan []byte, 1)
var LowerNodeCommChan = make(chan []byte, 1)

var ControlConnToAdmin net.Conn
var DataConnToAdmin net.Conn

var NODEID uint32 = uint32(1)
var AESKey []byte

func NewAgent(c *cli.Context) {
	AESKey = []byte(c.String("secret"))
	listenPort := c.String("listen")
	//ccPort := c.String("control")
	monitor := c.String("monitor")
	isStartNode := c.Bool("startnode")
	Monitor = monitor
	ListenPort = listenPort
	if isStartNode {
		go StartNodeInit(monitor, listenPort)
		WaitForExit(NODEID)
	} else {
		go SimpleNodeInit(monitor, listenPort)
		WaitForExit(NODEID)
	}
}

// 后续想让startnode与simplenode实现不一样的功能，故将两种node实现代码分开写
func StartNodeInit(monitor string, listenPort string) {
	NODEID = uint32(1)
	ControlConnToAdmin, DataConnToAdmin, finalid, err := node.StartNodeConn(monitor, listenPort, NODEID, AESKey)
	NODEID = uint32(finalid)
	if err != nil {
		os.Exit(1)
	}
	go HandleStartNodeConn(ControlConnToAdmin, DataConnToAdmin, monitor, NODEID)
	go node.StartNodeListen(listenPort, NODEID, ConnectedNode, AESKey)
	go WaitForExit(NODEID)
	for {
		controlConnForLowerNode := <-node.ControlConnForLowerNodeChan
		dataConnForLowerNode := <-node.DataConnForLowerNodeChan
		NewNodeMessage := <-node.NewNodeMessageChan
		PROXY_COMMAND_CHAN = make(chan []byte)
		LowerNodeCommChan <- NewNodeMessage
		go ProxyLowerNodeCommToUpperNode(ControlConnToAdmin, LowerNodeCommChan)
		go HandleLowerNodeConn(controlConnForLowerNode, dataConnForLowerNode, NODEID, LowerNodeCommChan)
	}

}

func SimpleNodeInit(monitor string, listenPort string) {
	NODEID = uint32(0)
	controlConnToUpperNode, dataConnToUpperNode, finalid, _ := node.StartNodeConn(monitor, listenPort, NODEID, AESKey)
	NODEID = uint32(finalid)
	go HandleSimpleNodeConn(controlConnToUpperNode, dataConnToUpperNode, monitor, NODEID)
	go node.StartNodeListen(listenPort, NODEID, ConnectedNode, AESKey)
	go WaitForExit(NODEID)
	for {
		controlConnForLowerNode := <-node.ControlConnForLowerNodeChan
		dataConnForLowerNode := <-node.DataConnForLowerNodeChan
		NewNodeMessage := <-node.NewNodeMessageChan
		PROXY_COMMAND_CHAN = make(chan []byte)
		LowerNodeCommChan <- NewNodeMessage
		go ProxyLowerNodeCommToUpperNode(controlConnToUpperNode, LowerNodeCommChan)
		go HandleLowerNodeConn(controlConnForLowerNode, dataConnForLowerNode, NODEID, LowerNodeCommChan)
	}

}

func HandleStartNodeConn(controlConnToAdmin net.Conn, dataConnToAdmin net.Conn, monitor string, NODEID uint32) {
	go HandleControlConnFromAdmin(controlConnToAdmin, NODEID)
	go HandleControlConnToAdmin(controlConnToAdmin, NODEID)
	go HandleDataConnFromAdmin(dataConnToAdmin, NODEID)
	go HandleDataConnToAdmin(dataConnToAdmin, NODEID)
}

func HandleDataConnToAdmin(dataConnToAdmin net.Conn, NODEID uint32) {
	for {
		proxyCmdResult := <-cmdResult
		_, err := dataConnToAdmin.Write(proxyCmdResult)
		if err != nil {
			logrus.Errorf("ERROR OCCURED!: %s", err)
			continue
		}
	}
}

//暂时不需要
func HandleDataConnFromAdmin(dataConnToAdmin net.Conn, NODEID uint32) {
	// for {
	// 	AdminData, err := common.ExtractDataResult(dataConnToAdmin)
	// 	if err != nil {
	// 		fmt.Println(err)
	// 	}
	// 	if AdminData.NodeId == NODEID {
	// 		switch AdminData.Datatype {
	// 		case "SOCKS5":
	// 			fmt.Println(AdminData.Result)
	// 			go socks.CheckBuffer(dataConnToAdmin, []byte(AdminData.Result), int(AdminData.ResultLength))
	// 		}
	// 	}
	// }
}

func HandleControlConnToAdmin(controlConnToAdmin net.Conn, NODEID uint32) {
	commandtoadmin := <-CommandToUpperNodeChan
	controlConnToAdmin.Write(commandtoadmin)
}
func HandleControlConnFromAdmin(controlConnToAdmin net.Conn, NODEID uint32) {
	cmd, stdout, stdin := CreatInteractiveShell()
	var neverexit bool = true
	for {
		command, err := common.ExtractCommand(controlConnToAdmin, AESKey)
		if err != nil {
			return
		}
		if command.NodeId == NODEID {
			switch command.Command {
			case "SHELL":
				switch command.Info {
				case "":
					logrus.Info("Get command to start shell")
					if neverexit {
						go func() {
							StartShell("", cmd, stdin, stdout)
						}()
					} else {
						go func() {
							StartShell("\n", cmd, stdin, stdout)
						}()
					}
				case "exit\n":
					neverexit = false
					continue
				default:
					go func() {
						StartShell(command.Info, cmd, stdin, stdout)
					}()
				}
			case "SOCKS":
				logrus.Info("Get command to start SOCKS")
				socksInit := strings.Split(command.Info, ":")
				socksPort := socksInit[0]
				socksUsername := socksInit[1]
				socksPass := socksInit[2]
				go StartSocks(controlConnToAdmin, socksPort, socksUsername, socksPass)
			case "SSH":
				fmt.Println("Get command to start SSH")
				err := StartSSH(controlConnToAdmin, command.Info, NODEID)
				if err == nil {
					go ReadCommand()
				} else {
					break
				}
			case "SSHCOMMAND":
				go WriteCommand(command.Info)
			case "ADMINOFFLINE":
				logrus.Error("Admin seems offline!")
				offlineCommand, _ := common.ConstructCommand("ADMINOFFLINE", "", 2, AESKey)
				PROXY_COMMAND_CHAN <- offlineCommand
				time.Sleep(2 * time.Second)
				os.Exit(1)
			}
		} else {
			if command.Command != "SOCKS" {
				passthroughCommand, _ := common.ConstructCommand(command.Command, command.Info, command.NodeId, AESKey)
				go ProxyToNextNode(passthroughCommand)
			} else {
				passthroughCommand, _ := common.ConstructCommand(command.Command, command.Info, command.NodeId, AESKey)
				go ProxyToNextNode(passthroughCommand)
				go StartSocksProxy(command.Info)
			}
		}
	}
}

func HandleLowerNodeConn(controlConnForLowerNode net.Conn, dataConnForLowerNode net.Conn, NODEID uint32, LowerNodeCommChan chan []byte) {
	go HandleControlConnToLowerNode(controlConnForLowerNode, NODEID, LowerNodeCommChan)
	go HandleControlConnFromLowerNode(controlConnForLowerNode, NODEID, LowerNodeCommChan)
	for {
		buffer := make([]byte, 409600)
		len, err := dataConnForLowerNode.Read(buffer)
		if err != nil {
			logrus.Error("Node ", NODEID+1, " seems offline")
			offlineMess, _ := common.ConstructCommand("AGENTOFFLINE", "", NODEID+1, AESKey)
			LowerNodeCommChan <- offlineMess
			break
		}
		cmdResult <- buffer[:len]
	}
}

func HandleControlConnToLowerNode(controlConnForLowerNode net.Conn, NODEID uint32, LowerNodeCommChan chan []byte) {
	for {
		proxy_command := <-PROXY_COMMAND_CHAN
		_, err := controlConnForLowerNode.Write(proxy_command)
		if err != nil {
			logrus.Error(err)
			continue
		}
	}
}

func HandleControlConnFromLowerNode(controlConnForLowerNode net.Conn, NODEID uint32, LowerNodeCommChan chan []byte) {
	for {
		command, err := common.ExtractCommand(controlConnForLowerNode, AESKey)
		if err != nil {
			offlineMess, _ := common.ConstructCommand("OFFLINE", "", NODEID+1, AESKey)
			PROXY_COMMAND_CHAN <- offlineMess
			return
		}
		if command.NodeId == NODEID { //暂时只有admin需要处理
		} else {
			proxyCommand, _ := common.ConstructCommand(command.Command, command.Info, command.NodeId, AESKey)
			LowerNodeCommChan <- proxyCommand
		}
	}
}

func HandleSimpleNodeConn(controlConnToUpperNode net.Conn, dataConnToUpperNode net.Conn, monitor string, NODEID uint32) {
	go HandleControlConnFromUpperNode(controlConnToUpperNode, NODEID)
	go HandleControlConnToUpperNode(controlConnToUpperNode, NODEID)
	for {
		proxyCmdResult := <-cmdResult
		_, err := dataConnToUpperNode.Write(proxyCmdResult)
		if err != nil {
			logrus.Errorf("ERROR OCCURED!: %s", err)
		}
	}
}

func HandleControlConnToUpperNode(controlConnToUpperNode net.Conn, NODEID uint32) {
	commandtouppernode := <-CommandToUpperNodeChan
	controlConnToUpperNode.Write(commandtouppernode)
}

func HandleControlConnFromUpperNode(controlConnToUpperNode net.Conn, NODEID uint32) {
	cmd, stdout, stdin := CreatInteractiveShell()
	var neverexit bool = true
	for {
		command, err := common.ExtractCommand(controlConnToUpperNode, AESKey)
		if err != nil {
			logrus.Error("upper node offline")
			os.Exit(1)
		}
		if command.NodeId == NODEID {
			switch command.Command {
			case "SHELL":
				switch command.Info {
				case "":
					logrus.Info("Get command to start shell")
					if neverexit {
						go func() {
							StartShell("", cmd, stdin, stdout)
						}()
					} else {
						go func() {
							StartShell("\n", cmd, stdin, stdout)
						}()
					}
				case "exit\n":
					neverexit = false
					continue
				case "OFFLINE":
					logrus.Error("Node ", NODEID-1, "seems down")
					offlineMess, _ := common.ConstructCommand("OFFLINE", "", NODEID+1, AESKey)
					PROXY_COMMAND_CHAN <- offlineMess
					os.Exit(1)
				default:
					go func() {
						StartShell(command.Info, cmd, stdin, stdout)
					}()
				}
			case "SOCKS":
				logrus.Info("Get command to start SOCKS")
				socksInit := strings.Split(command.Info, ":")
				socksPort := socksInit[0]
				socksUsername := socksInit[1]
				socksPass := socksInit[2]
				go StartSocks(controlConnToUpperNode, socksPort, socksUsername, socksPass)
			case "SSH":
				fmt.Println("Get command to start SSH")
				err := StartSSH(controlConnToUpperNode, command.Info, NODEID)
				if err == nil {
					go ReadCommand()
				} else {
					break
				}
			case "SSHCOMMAND":
				go WriteCommand(command.Info)
			case "ADMINOFFLINE":
				logrus.Error("Admin seems offline")
				offlineCommand, _ := common.ConstructCommand("ADMINOFFLINE", "", NODEID+1, AESKey)
				PROXY_COMMAND_CHAN <- offlineCommand
				time.Sleep(2 * time.Second)
				os.Exit(1) //admin下线后可以不断开，这里选择结束程序
			}
		} else {
			if command.Command != "SOCKS" {
				passthroughCommand, _ := common.ConstructCommand(command.Command, command.Info, command.NodeId, AESKey)
				go ProxyToNextNode(passthroughCommand)
			} else {
				passthroughCommand, _ := common.ConstructCommand(command.Command, command.Info, command.NodeId, AESKey)
				go ProxyToNextNode(passthroughCommand)
				go StartSocksProxy(command.Info)
			}
		}
	}
}

//暂时不需要
func HandleUpperNodeDataConn(dataConnToUpperNode net.Conn) {

}

func ProxyLowerNodeCommToUpperNode(upper net.Conn, LowerNodeCommChan chan []byte) {
	for {
		LowerNodeComm := <-LowerNodeCommChan
		_, err := upper.Write(LowerNodeComm)
		if err != nil {
			logrus.Error("Command cannot be proxy")
			return
		}
	}
}

func ProxyToNextNode(proxyCommand []byte) {
	PROXY_COMMAND_CHAN <- proxyCommand
}

func WaitForExit(NODEID uint32) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, os.Kill, syscall.SIGHUP, syscall.SIGUSR2)
	<-signalChan
	offlineMess, _ := common.ConstructCommand("OFFLINE", "", NODEID+1, AESKey)
	PROXY_COMMAND_CHAN <- offlineMess
	time.Sleep(5 * time.Second)
	os.Exit(1)
}
