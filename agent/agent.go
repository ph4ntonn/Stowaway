package agent

import (
	"Stowaway/common"
	"Stowaway/node"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

type SafeMap struct {
	sync.RWMutex
	SocksDataChan map[uint32]chan string
}

var (
	NODEID uint32 = uint32(1)

	Monitor       string
	ListenPort    string
	SocksUsername string
	SocksPass     string

	CommandToUpperNodeChan = make(chan []byte)
	cmdResult              = make(chan []byte)
	PROXY_COMMAND_CHAN     = make(chan []byte, 1)
	PROXY_DATA_CHAN        = make(chan []byte, 1)
	LowerNodeCommChan      = make(chan []byte, 1)
	SocksDataChanMap       *SafeMap

	ControlConnToAdmin net.Conn
	DataConnToAdmin    net.Conn
	SocksServer        net.Listener

	AESKey []byte
)

func newSafeMap() *SafeMap {
	sm := new(SafeMap)
	sm.SocksDataChan = make(map[uint32]chan string, 10)
	return sm
}

func NewAgent(c *cli.Context) {
	SocksDataChanMap = newSafeMap()
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
	var finalid uint32
	ControlConnToAdmin, DataConnToAdmin, finalid, err = node.StartNodeConn(monitor, listenPort, NODEID, AESKey)
	NODEID = uint32(finalid)
	if err != nil {
		os.Exit(1)
	}
	go HandleStartNodeConn(ControlConnToAdmin, DataConnToAdmin, monitor, NODEID)
	go node.StartNodeListen(listenPort, NODEID, AESKey)
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

//普通的node节点
func SimpleNodeInit(monitor string, listenPort string) {
	NODEID = uint32(0)
	var finalid uint32
	ControlConnToAdmin, DataConnToAdmin, finalid, _ = node.StartNodeConn(monitor, listenPort, NODEID, AESKey)
	NODEID = uint32(finalid)
	go HandleSimpleNodeConn(ControlConnToAdmin, DataConnToAdmin, monitor, NODEID)
	go node.StartNodeListen(listenPort, NODEID, AESKey)
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

//启动startnode
func HandleStartNodeConn(controlConnToAdmin net.Conn, dataConnToAdmin net.Conn, monitor string, NODEID uint32) {
	go HandleControlConnFromAdmin(controlConnToAdmin, NODEID)
	go HandleControlConnToAdmin(controlConnToAdmin, NODEID)
	go HandleDataConnFromAdmin(dataConnToAdmin, NODEID)
	go HandleDataConnToAdmin(dataConnToAdmin, NODEID)
}

//管理startnode发往admin的数据
func HandleDataConnToAdmin(dataConnToAdmin net.Conn, NODEID uint32) {
	for {
		proxyCmdResult := <-cmdResult
		_, err := dataConnToAdmin.Write(proxyCmdResult)
		if err != nil {
			//logrus.Errorf("ERROR OCCURED!: %s", err)
			return
		}
	}
}

//看函数名猜功能.jpg XD
func HandleDataConnFromAdmin(dataConnToAdmin net.Conn, NODEID uint32) {
	for {
		AdminData, err := common.ExtractDataResult(dataConnToAdmin, AESKey, NODEID)
		if err != nil {
			return
		}
		if AdminData.NodeId == NODEID {
			switch AdminData.Datatype {
			case "SOCKSDATA":
				if _, ok := SocksDataChanMap.SocksDataChan[AdminData.Clientsocks]; ok {
					SocksDataChanMap.SocksDataChan[AdminData.Clientsocks] <- AdminData.Result

				} else {
					//fmt.Println("create new chan", AdminData.Clientsocks)
					tempchan := make(chan string, 1)
					SocksDataChanMap.SocksDataChan[AdminData.Clientsocks] = tempchan
					go HanleClientSocksConn(SocksDataChanMap.SocksDataChan[AdminData.Clientsocks], SocksUsername, SocksPass, AdminData.Clientsocks, NODEID)
					SocksDataChanMap.SocksDataChan[AdminData.Clientsocks] <- AdminData.Result
				}
			case "FINOK":
				close(SocksDataChanMap.SocksDataChan[AdminData.Clientsocks])
				delete(SocksDataChanMap.SocksDataChan, AdminData.Clientsocks)
				//fmt.Println("close one, still left", len(SocksDataChanMap.SocksDataChan))
			}
		} else {
			ProxyData, _ := common.ConstructDataResult(AdminData.NodeId, AdminData.Clientsocks, AdminData.Success, AdminData.Datatype, AdminData.Result, AESKey, NODEID)
			go ProxyDataToNextNode(ProxyData)
		}
	}
}

//同上
func HandleControlConnToAdmin(controlConnToAdmin net.Conn, NODEID uint32) {
	commandtoadmin := <-CommandToUpperNodeChan
	controlConnToAdmin.Write(commandtoadmin)
}

//同上
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
							StartShell("", cmd, stdin, stdout, NODEID)
						}()
					} else {
						go func() {
							StartShell("\n", cmd, stdin, stdout, NODEID)
						}()
					}
				case "exit\n":
					neverexit = false
					continue
				default:
					go func() {
						StartShell(command.Info, cmd, stdin, stdout, NODEID)
					}()
				}
			case "SOCKS":
				logrus.Info("Get command to start SOCKS")
				socksInit := strings.Split(command.Info, ":")
				socksPort := socksInit[0]
				SocksUsername = socksInit[1]
				SocksPass = socksInit[2]
				go StartSocks(controlConnToAdmin, socksPort, SocksUsername, SocksPass)
			case "SOCKSOFF":
				logrus.Info("Get command to stop SOCKS")
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
			default:
				logrus.Error("Unknown command")
				continue
			}
		} else {
			passthroughCommand, _ := common.ConstructCommand(command.Command, command.Info, command.NodeId, AESKey)
			go ProxyCommToNextNode(passthroughCommand)
			//go StartSocksProxy(command.Info)
		}
	}
}

//管理下级节点
func HandleLowerNodeConn(controlConnForLowerNode net.Conn, dataConnForLowerNode net.Conn, NODEID uint32, LowerNodeCommChan chan []byte) {
	go HandleControlConnToLowerNode(controlConnForLowerNode, NODEID, LowerNodeCommChan)
	go HandleControlConnFromLowerNode(controlConnForLowerNode, NODEID, LowerNodeCommChan)
	go HandleDataConnFromLowerNode(dataConnForLowerNode, NODEID)
	go HandleDataConnToLowerNode(dataConnForLowerNode, NODEID)
}

//管理发往下级节点的控制信道
func HandleControlConnToLowerNode(controlConnForLowerNode net.Conn, NODEID uint32, LowerNodeCommChan chan []byte) {
	for {
		proxy_command := <-PROXY_COMMAND_CHAN
		_, err := controlConnForLowerNode.Write(proxy_command)
		if err != nil {
			//logrus.Error(err)
			return
		}
	}
}

//看到那个from了么
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

func HandleDataConnFromLowerNode(dataConnForLowerNode net.Conn, NODEID uint32) {
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

func HandleDataConnToLowerNode(dataConnForLowerNode net.Conn, NODEID uint32) {
	for {
		proxy_data := <-PROXY_DATA_CHAN
		_, err := dataConnForLowerNode.Write(proxy_data)
		if err != nil {
			break
		}
	}
}

//启动普通节点
func HandleSimpleNodeConn(controlConnToUpperNode net.Conn, dataConnToUpperNode net.Conn, monitor string, NODEID uint32) {
	go HandleControlConnFromUpperNode(controlConnToUpperNode, NODEID)
	go HandleControlConnToUpperNode(controlConnToUpperNode, NODEID)
	go HandleDataConnFromUpperNode(dataConnToUpperNode)
	go HandleDataConnToUpperNode(dataConnToUpperNode)
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
							StartShell("", cmd, stdin, stdout, NODEID)
						}()
					} else {
						go func() {
							StartShell("\n", cmd, stdin, stdout, NODEID)
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
						StartShell(command.Info, cmd, stdin, stdout, NODEID)
					}()
				}
			case "SOCKS":
				logrus.Info("Get command to start SOCKS")
				socksInit := strings.Split(command.Info, ":")
				socksPort := socksInit[0]
				SocksUsername := socksInit[1]
				SocksPass := socksInit[2]
				go StartSocks(controlConnToUpperNode, socksPort, SocksUsername, SocksPass)
			case "SOCKSOFF":
				logrus.Info("Get command to stop SOCKS")
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
			default:
				logrus.Error("Unknown command")
				continue
			}
		} else {
			passthroughCommand, _ := common.ConstructCommand(command.Command, command.Info, command.NodeId, AESKey)
			go ProxyCommToNextNode(passthroughCommand)
			//go StartSocksProxy(command.Info)
		}
	}
}

func HandleDataConnToUpperNode(dataConnToUpperNode net.Conn) {
	for {
		proxyCmdResult := <-cmdResult
		_, err := dataConnToUpperNode.Write(proxyCmdResult)
		if err != nil {
			//logrus.Errorf("ERROR OCCURED!: %s", err)
			return
		}
	}
}

func HandleDataConnFromUpperNode(dataConnToUpperNode net.Conn) {
	for {
		AdminData, err := common.ExtractDataResult(dataConnToUpperNode, AESKey, NODEID)
		if err != nil {
			return
		}
		if AdminData.NodeId == NODEID {
			switch AdminData.Datatype {
			case "SOCKSDATA":
				if _, ok := SocksDataChanMap.SocksDataChan[AdminData.Clientsocks]; ok {
					SocksDataChanMap.SocksDataChan[AdminData.Clientsocks] <- AdminData.Result
				} else {
					//fmt.Println("create new chan", AdminData.Clientsocks)
					tempchan := make(chan string, 1)
					SocksDataChanMap.SocksDataChan[AdminData.Clientsocks] = tempchan
					go HanleClientSocksConn(SocksDataChanMap.SocksDataChan[AdminData.Clientsocks], SocksUsername, SocksPass, AdminData.Clientsocks, NODEID)
					SocksDataChanMap.SocksDataChan[AdminData.Clientsocks] <- AdminData.Result
				}
			case "FINOK":
				close(SocksDataChanMap.SocksDataChan[AdminData.Clientsocks])
				delete(SocksDataChanMap.SocksDataChan, AdminData.Clientsocks)
				//fmt.Println("close one, still left", len(SocksDataChanMap.SocksDataChan))
			}
		} else {
			ProxyData, _ := common.ConstructDataResult(AdminData.NodeId, AdminData.Clientsocks, AdminData.Success, AdminData.Datatype, AdminData.Result, AESKey, NODEID)
			go ProxyDataToNextNode(ProxyData)
		}
	}
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

//将命令传递到下一节点
func ProxyCommToNextNode(proxyCommand []byte) {
	PROXY_COMMAND_CHAN <- proxyCommand
}

//将数据传递到下一节点
func ProxyDataToNextNode(proxyData []byte) {
	PROXY_DATA_CHAN <- proxyData
}

//捕捉程序退出信号
func WaitForExit(NODEID uint32) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, os.Kill, syscall.SIGHUP, syscall.SIGUSR2)
	<-signalChan
	offlineMess, _ := common.ConstructCommand("OFFLINE", "", NODEID+1, AESKey)
	PROXY_COMMAND_CHAN <- offlineMess
	time.Sleep(5 * time.Second)
	os.Exit(1)
}
