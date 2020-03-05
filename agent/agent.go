package agent

import (
	"Stowaway/common"
	"Stowaway/node"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/urfave/cli/v2"
)

type SafeMap struct {
	sync.RWMutex
	SocksDataChan map[uint32]chan string
}

var (
	NODEID     uint32 = uint32(1)
	NotLastOne bool   = false
	Waiting    bool   = false

	SocksUsername string
	SocksPass     string

	CmdResult          = make(chan []byte, 1)
	ReConnCome         = make(chan bool, 1)
	Proxy_Command_Chan = make(chan []byte, 1)
	Proxy_Data_Chan    = make(chan []byte, 1)
	LowerNodeCommChan  = make(chan []byte, 1)
	Eof                = make(chan string, 1)

	FileDataMap      *common.SafeFileDataMap
	SocksDataChanMap *SafeMap

	ControlConnToAdmin net.Conn
	DataConnToAdmin    net.Conn
	SocksServer        net.Listener

	AESKey []byte
)

func NewSafeMap() *SafeMap {
	sm := new(SafeMap)
	sm.SocksDataChan = make(map[uint32]chan string, 10)
	return sm
}

func NewSafeFileDataMap() *common.SafeFileDataMap {
	sm := new(common.SafeFileDataMap)
	sm.FileDataChan = make(map[int]string)
	return sm
}

func NewAgent(c *cli.Context) {
	SocksDataChanMap = NewSafeMap()
	FileDataMap = NewSafeFileDataMap()
	AESKey = []byte(c.String("secret"))
	listenPort := c.String("listen")
	single := c.Bool("single")
	reconn := c.String("reconnect")
	passive := c.Bool("reverse")
	monitor := c.String("monitor")
	isStartNode := c.Bool("startnode")

	if isStartNode && passive == false {
		go StartNodeInit(monitor, listenPort, reconn, passive)
		WaitForExit(NODEID)
	} else if passive == false {
		go SimpleNodeInit(monitor, listenPort)
		WaitForExit(NODEID)
	} else if isStartNode && passive {
		go StartNodeReversemodeInit(monitor, listenPort, reconn, single, passive)
		WaitForExit(NODEID)
	} else if passive {
		go SimpleNodeReversemodeInit(monitor, listenPort)
		WaitForExit(NODEID)
	}
}

// 初始化代码开始

// 后续想让startnode与simplenode实现不一样的功能，故将两种node实现代码分开写
func StartNodeInit(monitor, listenPort, reConn string, passive bool) {
	var err error
	NODEID = uint32(1)
	ControlConnToAdmin, DataConnToAdmin, NODEID, err = node.StartNodeConn(monitor, listenPort, NODEID, AESKey)
	go common.SendHeartBeatControl(ControlConnToAdmin, NODEID, AESKey)
	if err != nil {
		os.Exit(1)
	}
	go HandleStartNodeConn(&ControlConnToAdmin, &DataConnToAdmin, monitor, listenPort, reConn, passive, NODEID)
	go node.StartNodeListen(listenPort, NODEID, AESKey, false, false)
	for {
		controlConnForLowerNode := <-node.ControlConnForLowerNodeChan
		dataConnForLowerNode := <-node.DataConnForLowerNodeChan
		NewNodeMessage := <-node.NewNodeMessageChan
		Proxy_Command_Chan = make(chan []byte)
		LowerNodeCommChan <- NewNodeMessage
		NotLastOne = true
		go common.SendHeartBeatData(dataConnForLowerNode, NODEID, AESKey)
		go HandleLowerNodeConn(controlConnForLowerNode, dataConnForLowerNode, NODEID)
	}

}

//普通的node节点
func SimpleNodeInit(monitor, listenPort string) {
	var err error
	NODEID = uint32(0)
	ControlConnToAdmin, DataConnToAdmin, NODEID, err = node.StartNodeConn(monitor, listenPort, NODEID, AESKey)
	go common.SendHeartBeatControl(ControlConnToAdmin, NODEID, AESKey)
	if err != nil {
		os.Exit(1)
	}
	go HandleSimpleNodeConn(&ControlConnToAdmin, &DataConnToAdmin, NODEID)
	go node.StartNodeListen(listenPort, NODEID, AESKey, false, false)
	for {
		controlConnForLowerNode := <-node.ControlConnForLowerNodeChan
		dataConnForLowerNode := <-node.DataConnForLowerNodeChan
		NewNodeMessage := <-node.NewNodeMessageChan
		Proxy_Command_Chan = make(chan []byte)
		LowerNodeCommChan <- NewNodeMessage
		NotLastOne = true
		go common.SendHeartBeatData(dataConnForLowerNode, NODEID, AESKey)
		go HandleLowerNodeConn(controlConnForLowerNode, dataConnForLowerNode, NODEID)
	}
}

//reverse mode下的startnode节点
func StartNodeReversemodeInit(monitor, listenPort, reConn string, single, passive bool) {
	NODEID = uint32(1)
	ControlConnToAdmin, DataConnToAdmin, NODEID = node.AcceptConnFromUpperNode(listenPort, NODEID, AESKey)
	go common.SendHeartBeatControl(ControlConnToAdmin, NODEID, AESKey)
	go HandleStartNodeConn(&ControlConnToAdmin, &DataConnToAdmin, monitor, listenPort, reConn, passive, NODEID)
	if reConn == "0" {
		go node.StartNodeListen(listenPort, NODEID, AESKey, true, single)
	} else {
		go node.StartNodeListen(listenPort, NODEID, AESKey, false, single)
	}
	for {
		controlConnForLowerNode := <-node.ControlConnForLowerNodeChan
		dataConnForLowerNode := <-node.DataConnForLowerNodeChan
		if !Waiting {
			NewNodeMessage := <-node.NewNodeMessageChan
			Proxy_Command_Chan = make(chan []byte)
			LowerNodeCommChan <- NewNodeMessage
			NotLastOne = true
			go common.SendHeartBeatData(dataConnForLowerNode, NODEID, AESKey)
			go HandleLowerNodeConn(controlConnForLowerNode, dataConnForLowerNode, NODEID)
		} else { // 需要重连操作的话
			ReConnCome <- true
			ControlConnToAdmin = controlConnForLowerNode
			DataConnToAdmin = dataConnForLowerNode
			go common.SendHeartBeatControl(ControlConnToAdmin, NODEID, AESKey)
		}
	}
}

//reverse mode下的普通节点
func SimpleNodeReversemodeInit(monitor, listenPort string) {
	NODEID = uint32(0)
	ControlConnToAdmin, DataConnToAdmin, NODEID = node.AcceptConnFromUpperNode(listenPort, NODEID, AESKey)
	go common.SendHeartBeatControl(ControlConnToAdmin, NODEID, AESKey)
	go HandleSimpleNodeConn(&ControlConnToAdmin, &DataConnToAdmin, NODEID)
	go node.StartNodeListen(listenPort, NODEID, AESKey, false, false)
	for {
		controlConnForLowerNode := <-node.ControlConnForLowerNodeChan
		dataConnForLowerNode := <-node.DataConnForLowerNodeChan
		NewNodeMessage := <-node.NewNodeMessageChan
		Proxy_Command_Chan = make(chan []byte)
		LowerNodeCommChan <- NewNodeMessage
		NotLastOne = true
		go common.SendHeartBeatData(dataConnForLowerNode, NODEID, AESKey)
		go HandleLowerNodeConn(controlConnForLowerNode, dataConnForLowerNode, NODEID)
	}
}

//初始化代码结束

//startnode启动代码开始

//启动startnode
func HandleStartNodeConn(controlConnToAdmin *net.Conn, dataConnToAdmin *net.Conn, monitor, listenPort, reConn string, passive bool, NODEID uint32) {
	go HandleControlConnFromAdmin(controlConnToAdmin, monitor, listenPort, reConn, passive, NODEID)
	go HandleControlConnToAdmin(controlConnToAdmin)
	go HandleDataConnFromAdmin(dataConnToAdmin, NODEID)
	go HandleDataConnToAdmin(dataConnToAdmin)
}

//管理startnode发往admin的数据
func HandleDataConnToAdmin(dataConnToAdmin *net.Conn) {
	for {
		proxyCmdResult := <-CmdResult
		_, err := (*dataConnToAdmin).Write(proxyCmdResult)
		if err != nil {
			//logrus.Errorf("ERROR OCCURED!: %s", err)
			continue
		}
	}
}

//看函数名猜功能.jpg XD
func HandleDataConnFromAdmin(dataConnToAdmin *net.Conn, NODEID uint32) {
	for {
		AdminData, err := common.ExtractDataResult(*dataConnToAdmin, AESKey, NODEID)
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}
		if AdminData.NodeId == NODEID {
			switch AdminData.Datatype {
			case "SOCKSDATA":
				SocksDataChanMap.RLock()
				if _, ok := SocksDataChanMap.SocksDataChan[AdminData.Clientsocks]; ok {
					SocksDataChanMap.SocksDataChan[AdminData.Clientsocks] <- AdminData.Result
					SocksDataChanMap.RUnlock()
				} else {
					//fmt.Println("create new chan", AdminData.Clientsocks)
					SocksDataChanMap.RUnlock()
					tempchan := make(chan string, 10)
					SocksDataChanMap.Lock()
					SocksDataChanMap.SocksDataChan[AdminData.Clientsocks] = tempchan
					go HanleClientSocksConn(SocksDataChanMap.SocksDataChan[AdminData.Clientsocks], SocksUsername, SocksPass, AdminData.Clientsocks, NODEID)
					SocksDataChanMap.SocksDataChan[AdminData.Clientsocks] <- AdminData.Result
					SocksDataChanMap.Unlock()
				}
			case "FILEDATA": //接收文件内容
				slicenum, _ := strconv.Atoi(AdminData.FileSliceNum)
				FileDataMap.Lock()
				FileDataMap.FileDataChan[slicenum] = AdminData.Result
				FileDataMap.Unlock()
			case "EOF": //文件读取结束
				Eof <- AdminData.FileSliceNum
			case "FINOK":
				SocksDataChanMap.Lock() //性能损失？
				if _, ok := SocksDataChanMap.SocksDataChan[AdminData.Clientsocks]; ok {
					if !IsClosed(SocksDataChanMap.SocksDataChan[AdminData.Clientsocks]) {
						close(SocksDataChanMap.SocksDataChan[AdminData.Clientsocks])
					}
					delete(SocksDataChanMap.SocksDataChan, AdminData.Clientsocks)
					//fmt.Println("close one, still left", len(SocksDataChanMap.SocksDataChan))
				}
				SocksDataChanMap.Unlock()
			case "HEARTBEAT":
				hbdatapack, _ := common.ConstructDataResult(0, 0, " ", "KEEPALIVE", " ", AESKey, NODEID)
				(*dataConnToAdmin).Write(hbdatapack)
			}
		} else {
			ProxyData, _ := common.ConstructDataResult(AdminData.NodeId, AdminData.Clientsocks, AdminData.FileSliceNum, AdminData.Datatype, AdminData.Result, AESKey, NODEID)
			go ProxyDataToNextNode(ProxyData)
		}
	}
}

// //处理发往上级的
func HandleControlConnToAdmin(controlConnToAdmin *net.Conn) {
	for {
		LowerNodeComm := <-LowerNodeCommChan
		_, err := (*controlConnToAdmin).Write(LowerNodeComm)
		if err != nil {
			log.Println("[*]Command cannot be proxy")
		}
	}
}

//同上
func HandleControlConnFromAdmin(controlConnToAdmin *net.Conn, monitor, listenPort, reConn string, passive bool, NODEID uint32) {
	var neverexit bool = true
	var CannotRead = make(chan bool, 1)
	var GetName = make(chan bool, 1)

	stdout, stdin := CreatInteractiveShell()

	for {
		command, err := common.ExtractCommand(*controlConnToAdmin, AESKey)
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}
		if command.NodeId == NODEID {
			switch command.Command {
			case "SHELL":
				switch command.Info {
				case "":
					//logrus.Info("Get command to start shell")
					if neverexit { //判断之前是否进入过shell模式
						go func() {
							StartShell("", stdin, stdout, NODEID)
						}()
					} else {
						go func() {
							StartShell("\n", stdin, stdout, NODEID)
						}()
					}
				case "exit\n":
					neverexit = false
					continue
				default:
					go func() {
						StartShell(command.Info, stdin, stdout, NODEID)
					}()
				}
			case "SOCKS":
				//logrus.Info("Get command to start SOCKS")
				socksinfo := strings.Split(command.Info, ":::")
				SocksUsername = socksinfo[1]
				SocksPass = socksinfo[2]
				go StartSocks(controlConnToAdmin)
			case "SOCKSOFF":
				//logrus.Info("Get command to stop SOCKS")
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
			case "CONNECT":
				go node.ConnectNextNode(command.Info, NODEID, AESKey)
			case "FILENAME":
				var err error
				UploadFile, err := os.Create(command.Info)
				if err != nil {
					respComm, _ := common.ConstructCommand("CREATEFAIL", "", 0, AESKey) //从控制信道上返回文件是否能成功创建的响应
					ControlConnToAdmin.Write(respComm)
				} else {
					respComm, _ := common.ConstructCommand("NAMECONFIRM", "", 0, AESKey)
					ControlConnToAdmin.Write(respComm)
					go common.ReceiveFile(controlConnToAdmin, Eof, FileDataMap, CannotRead, UploadFile, AESKey, false)
				}
			case "DOWNLOADFILE":
				go common.UploadFile(command.Info, ControlConnToAdmin, DataConnToAdmin, 0, GetName, AESKey, NODEID, false)
			case "NAMECONFIRM":
				GetName <- true
			case "CREATEFAIL":
				GetName <- false
			case "CANNOTREAD":
				CannotRead <- true
			case "ADMINOFFLINE":
				log.Println("[*]Admin seems offline!")
				if reConn != "0" && reConn != "" && !passive {
					SocksDataChanMap = NewSafeMap()
					if NotLastOne {
						messCommand, _ := common.ConstructCommand("CLEAR", "", 2, AESKey)
						Proxy_Command_Chan <- messCommand
					}
					TryReconnect(reConn, monitor, listenPort)
					if NotLastOne {
						messCommand, _ := common.ConstructCommand("RECONN", "", 2, AESKey)
						Proxy_Command_Chan <- messCommand
					}
				} else if reConn == "0" && passive {
					SocksDataChanMap = NewSafeMap()
					if NotLastOne {
						messCommand, _ := common.ConstructCommand("CLEAR", "", 2, AESKey)
						Proxy_Command_Chan <- messCommand
					}
					Waiting = true
					<-ReConnCome
					if NotLastOne {
						messCommand, _ := common.ConstructCommand("RECONN", "", 2, AESKey)
						Proxy_Command_Chan <- messCommand
					}
				} else {
					if NotLastOne {
						offlineCommand, _ := common.ConstructCommand("ADMINOFFLINE", "", 2, AESKey)
						Proxy_Command_Chan <- offlineCommand
					}
					time.Sleep(2 * time.Second)
					os.Exit(1)
				}
			case "KEEPALIVE":
			default:
				//logrus.Error("Unknown command")
				continue
			}
		} else {
			passthroughCommand, _ := common.ConstructCommand(command.Command, command.Info, command.NodeId, AESKey)
			go ProxyCommToNextNode(passthroughCommand)
			//go StartSocksProxy(command.Info)
		}
	}
}

//startnode启动代码结束

//管理下行节点代码开始

//管理下级节点
func HandleLowerNodeConn(controlConnForLowerNode net.Conn, dataConnForLowerNode net.Conn, NODEID uint32) {
	go HandleControlConnToLowerNode(controlConnForLowerNode)
	go HandleControlConnFromLowerNode(controlConnForLowerNode, NODEID)
	go HandleDataConnFromLowerNode(dataConnForLowerNode, NODEID)
	go HandleDataConnToLowerNode(dataConnForLowerNode, NODEID)
}

//管理发往下级节点的控制信道
func HandleControlConnToLowerNode(controlConnForLowerNode net.Conn) {
	for {
		proxy_command := <-Proxy_Command_Chan
		_, err := controlConnForLowerNode.Write(proxy_command)
		if err != nil {
			//logrus.Error(err)
			return
		}
	}
}

//看到那个from了么
func HandleControlConnFromLowerNode(controlConnForLowerNode net.Conn, NODEID uint32) {
	for {
		command, err := common.ExtractCommand(controlConnForLowerNode, AESKey)
		if err != nil {
			offlineMess, _ := common.ConstructCommand("OFFLINE", "", NODEID+1, AESKey)
			Proxy_Command_Chan <- offlineMess
			return
		}
		if command.Command == "RECONNID" && command.Info == "" {
			proxyCommand, _ := common.ConstructCommand(command.Command, controlConnForLowerNode.RemoteAddr().String(), command.NodeId, AESKey)
			LowerNodeCommChan <- proxyCommand
			continue
		}
		if command.Command == "HEARTBEAT" {
			hbcommpack, _ := common.ConstructCommand("KEEPALIVE", "", NODEID+1, AESKey)
			controlConnForLowerNode.Write(hbcommpack)
			continue
		}
		if command.NodeId == NODEID { //暂时只有admin需要处理
		} else {
			proxyCommand, _ := common.ConstructCommand(command.Command, command.Info, command.NodeId, AESKey)
			LowerNodeCommChan <- proxyCommand
		}
	}
}

// 处理来自于下一级节点的数据信道
func HandleDataConnFromLowerNode(dataConnForLowerNode net.Conn, NODEID uint32) {
	for {
		buffer := make([]byte, 409600)
		len, err := dataConnForLowerNode.Read(buffer)
		if err != nil {
			log.Println("[*]Node ", NODEID+1, " seems offline")
			offlineMess, _ := common.ConstructCommand("AGENTOFFLINE", "", NODEID+1, AESKey) //下一级节点掉线，向上级节点传递下级节点掉线的消息
			LowerNodeCommChan <- offlineMess
			break
		}
		CmdResult <- buffer[:len]
	}
}

//处理发往下一级节点的数据信道
func HandleDataConnToLowerNode(dataConnForLowerNode net.Conn, NODEID uint32) {
	for {
		proxy_data := <-Proxy_Data_Chan
		_, err := dataConnForLowerNode.Write(proxy_data)
		if err != nil {
			log.Println("[*]", err)
			break
		}
	}
}

//管理下行节点代码结束

//普通节点启动代码开始

//启动普通节点
func HandleSimpleNodeConn(controlConnToUpperNode *net.Conn, dataConnToUpperNode *net.Conn, NODEID uint32) {
	go HandleControlConnFromUpperNode(controlConnToUpperNode, NODEID)
	go HandleControlConnToUpperNode(controlConnToUpperNode)
	go HandleDataConnFromUpperNode(dataConnToUpperNode)
	go HandleDataConnToUpperNode(dataConnToUpperNode)
}

// 处理发往上一级节点的控制信道
func HandleControlConnToUpperNode(controlConnToUpperNode *net.Conn) {
	for {
		LowerNodeComm := <-LowerNodeCommChan
		_, err := (*controlConnToUpperNode).Write(LowerNodeComm)
		if err != nil {
			log.Println("[*]Command cannot be proxy")
		}
	}
}

//处理来自上一级节点的控制信道
func HandleControlConnFromUpperNode(controlConnToUpperNode *net.Conn, NODEID uint32) {
	var neverexit bool = true
	var CannotRead = make(chan bool, 1)
	var GetName = make(chan bool, 1)

	stdout, stdin := CreatInteractiveShell()

	for {
		command, err := common.ExtractCommand(*controlConnToUpperNode, AESKey)
		if err != nil {
			log.Fatal("upper node offline")
		}
		if command.NodeId == NODEID {
			switch command.Command {
			case "SHELL":
				switch command.Info {
				case "":
					//logrus.Info("Get command to start shell")
					if neverexit {
						go func() {
							StartShell("", stdin, stdout, NODEID)
						}()
					} else {
						go func() {
							StartShell("\n", stdin, stdout, NODEID)
						}()
					}
				case "exit\n":
					neverexit = false
					continue
				default:
					go func() {
						StartShell(command.Info, stdin, stdout, NODEID)
					}()
				}
			case "OFFLINE": //上一级节点下线
				fmt.Println("[*]Node ", NODEID-1, " seems down")
				if NotLastOne {
					offlineMess, _ := common.ConstructCommand("OFFLINE", "", NODEID+1, AESKey)
					Proxy_Command_Chan <- offlineMess
				}
				time.Sleep(2 * time.Second)
				os.Exit(1)
			case "SOCKS":
				//logrus.Info("Get command to start SOCKS")
				socksinfo := strings.Split(command.Info, ":::")
				SocksUsername = socksinfo[1]
				SocksPass = socksinfo[2]
				go StartSocks(controlConnToUpperNode)
			case "SOCKSOFF":
				//logrus.Info("Get command to stop SOCKS")
			case "SSH":
				//fmt.Println("Get command to start SSH")
				err := StartSSH(controlConnToUpperNode, command.Info, NODEID)
				if err == nil {
					go ReadCommand()
				} else {
					break
				}
			case "SSHCOMMAND":
				go WriteCommand(command.Info)
			case "CONNECT":
				go node.ConnectNextNode(command.Info, NODEID, AESKey)
			case "FILENAME":
				var err error
				UploadFile, err := os.Create(command.Info)
				if err != nil {
					respComm, _ := common.ConstructCommand("CREATEFAIL", "", 0, AESKey) //从控制信道上返回文件是否能成功创建的响应
					ControlConnToAdmin.Write(respComm)
				} else {
					respComm, _ := common.ConstructCommand("NAMECONFIRM", "", 0, AESKey)
					ControlConnToAdmin.Write(respComm)
					go common.ReceiveFile(controlConnToUpperNode, Eof, FileDataMap, CannotRead, UploadFile, AESKey, false)
				}
			case "DOWNLOADFILE":
				go common.UploadFile(command.Info, ControlConnToAdmin, DataConnToAdmin, 0, GetName, AESKey, NODEID, false)
			case "NAMECONFIRM":
				GetName <- true
			case "CREATEFAIL":
				GetName <- false
			case "CANNOTREAD":
				CannotRead <- true
			case "ADMINOFFLINE": //startnode不执行重连模式时admin下线后传递的数据
				fmt.Println("Admin seems offline")
				if NotLastOne {
					offlineCommand, _ := common.ConstructCommand("ADMINOFFLINE", "", NODEID+1, AESKey)
					Proxy_Command_Chan <- offlineCommand
				}
				time.Sleep(2 * time.Second)
				os.Exit(1)
			case "RECONN": //startnode执行重连模式时admin下线后传递的数据
				respCommand, _ := common.ConstructCommand("RECONNID", "", NODEID, AESKey)
				(*controlConnToUpperNode).Write(respCommand)
				if NotLastOne {
					passCommand, _ := common.ConstructCommand("RECONN", "", NODEID+1, AESKey)
					Proxy_Command_Chan <- passCommand
				}
			case "CLEAR":
				SocksDataChanMap = NewSafeMap()
				if NotLastOne {
					messCommand, _ := common.ConstructCommand("CLEAR", "", NODEID+1, AESKey)
					Proxy_Command_Chan <- messCommand
				}
			case "KEEPALIVE":
			default:
				//logrus.Error("Unknown command")
				continue
			}
		} else {
			passthroughCommand, _ := common.ConstructCommand(command.Command, command.Info, command.NodeId, AESKey)
			go ProxyCommToNextNode(passthroughCommand)
			//go StartSocksProxy(command.Info)
		}
	}
}

//处理传递给上一个节点的数据信道
func HandleDataConnToUpperNode(dataConnToUpperNode *net.Conn) {
	for {
		proxyCmdResult := <-CmdResult
		_, err := (*dataConnToUpperNode).Write(proxyCmdResult)
		if err != nil {
			//logrus.Errorf("ERROR OCCURED!: %s", err)
			continue
		}
	}
}

//处理由上一个节点传递过来的数据信道
func HandleDataConnFromUpperNode(dataConnToUpperNode *net.Conn) {
	for {
		AdminData, err := common.ExtractDataResult(*dataConnToUpperNode, AESKey, NODEID)
		if err != nil {
			return
		}
		if AdminData.NodeId == NODEID { //判断是否是传递给自己的
			switch AdminData.Datatype {
			case "SOCKSDATA":
				SocksDataChanMap.RLock()
				if _, ok := SocksDataChanMap.SocksDataChan[AdminData.Clientsocks]; ok {
					SocksDataChanMap.SocksDataChan[AdminData.Clientsocks] <- AdminData.Result
					SocksDataChanMap.RUnlock()
				} else {
					SocksDataChanMap.RUnlock()
					tempchan := make(chan string, 10)
					SocksDataChanMap.Lock()
					SocksDataChanMap.SocksDataChan[AdminData.Clientsocks] = tempchan
					go HanleClientSocksConn(SocksDataChanMap.SocksDataChan[AdminData.Clientsocks], SocksUsername, SocksPass, AdminData.Clientsocks, NODEID)
					SocksDataChanMap.SocksDataChan[AdminData.Clientsocks] <- AdminData.Result
					SocksDataChanMap.Unlock()
				}
			case "FINOK":
				SocksDataChanMap.Lock()
				if _, ok := SocksDataChanMap.SocksDataChan[AdminData.Clientsocks]; ok {
					if !IsClosed(SocksDataChanMap.SocksDataChan[AdminData.Clientsocks]) {
						close(SocksDataChanMap.SocksDataChan[AdminData.Clientsocks])
					}
					delete(SocksDataChanMap.SocksDataChan, AdminData.Clientsocks)
				}
				SocksDataChanMap.Unlock()
				//fmt.Println("close one, still left", len(SocksDataChanMap.SocksDataChan))
			case "FILEDATA": //接收文件内容
				slicenum, _ := strconv.Atoi(AdminData.FileSliceNum)
				FileDataMap.FileDataChan[slicenum] = AdminData.Result
			case "EOF": //文件读取结束
				Eof <- AdminData.FileSliceNum
			case "HEARTBEAT":
				hbdatapack, _ := common.ConstructDataResult(0, 0, " ", "KEEPALIVE", " ", AESKey, NODEID)
				(*dataConnToUpperNode).Write(hbdatapack)
			default:
				//logrus.Error("Unknown data")
				continue
			}
		} else {
			ProxyData, _ := common.ConstructDataResult(AdminData.NodeId, AdminData.Clientsocks, AdminData.FileSliceNum, AdminData.Datatype, AdminData.Result, AESKey, NODEID)
			go ProxyDataToNextNode(ProxyData)
		}
	}
}

//普通节点启动代码结束

//agent主体代码结束
