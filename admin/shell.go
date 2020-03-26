package admin

import (
	"Stowaway/common"
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
)

var CurrentNode uint32

/*-------------------------Node模式下相关代码--------------------------*/
//处理node模式下用户的输入
func HandleNodeCommand(startNodeConn net.Conn, NodeID string) {
	nodeid64, _ := strconv.ParseInt(NodeID, 10, 32)
	nodeID := uint32(nodeid64)
	CurrentNode = nodeID //把nodeid提取出来，以供上传/下载文件功能使用

	for {
		AdminCommand := <-AdminStuff.AdminCommandChan
		switch AdminCommand[0] {
		case "shell":
			respCommand, err := common.ConstructPayload(nodeID, "COMMAND", "SHELL", " ", "", 0, 0, AdminStatus.AESKey, false)
			_, err = startNodeConn.Write(respCommand)
			if err != nil {
				log.Printf("[*]ERROR OCCURED!: %s", err)
			}
			HandleShellToNode(startNodeConn, nodeID)
		case "socks":
			var socksStartData string
			if len(AdminCommand) == 2 {
				socksStartData = fmt.Sprintf("%s:::%s:::%s", AdminCommand[1], "", "")
			} else if len(AdminCommand) == 3 {
				fmt.Println("Illegal username/password! Try again!")
				AdminStatus.ReadyChange <- true
				AdminStatus.IsShellMode <- true
				break
			} else {
				socksStartData = fmt.Sprintf("%s:::%s:::%s", AdminCommand[1], AdminCommand[2], AdminCommand[3])
			}
			respCommand, err := common.ConstructPayload(nodeID, "COMMAND", "SOCKS", " ", socksStartData, 0, 0, AdminStatus.AESKey, false)
			_, err = startNodeConn.Write(respCommand)
			if err != nil {
				log.Println("[*]StartNode seems offline")
			}
			if <-AdminStatus.NodeSocksStarted {
				go StartSocksServiceForClient(AdminCommand, startNodeConn, nodeID)
			}
			AdminStatus.ReadyChange <- true
			AdminStatus.IsShellMode <- true
		case "stopsocks":
			if len(AdminStuff.SocksListenerForClient) == 0 {
				log.Println("[*]You have never started socks service!")
			} else {
				for _, listener := range AdminStuff.SocksListenerForClient {
					err := listener.Close()
					if err != nil {
						log.Println("[*]One socks listener seems already closed.Won't close it again...")
					}
				}
				log.Println("[*]All socks listeners are closed successfully!")
			}
			AdminStatus.ReadyChange <- true
			AdminStatus.IsShellMode <- true
		case "ssh":
			if len(AdminCommand) == 4 {
				go StartSSHService(startNodeConn, AdminCommand, nodeID)
				HandleSSHToNode(startNodeConn, nodeID)
			} else {
				fmt.Println("Wrong format! Should be ssh [ip:port] [name] [pass]")
				AdminStatus.ReadyChange <- true
				AdminStatus.IsShellMode <- true
			}
		case "connect":
			if len(AdminCommand) == 2 {
				respCommand, _ := common.ConstructPayload(nodeID, "COMMAND", "CONNECT", " ", AdminCommand[1], 0, 0, AdminStatus.AESKey, false)
				startNodeConn.Write(respCommand)
			} else {
				fmt.Println("Wrong format! Should be connect [ip:port]")
			}
			AdminStatus.ReadyChange <- true
			AdminStatus.IsShellMode <- true
		case "upload":
			if len(AdminCommand) == 2 {
				go common.UploadFile(AdminCommand[1], &startNodeConn, nodeID, AdminStatus.GetName, AdminStatus.AESKey, 0, true)
			} else {
				fmt.Println("Bad format! Should be upload [filename]")
			}
			AdminStatus.ReadyChange <- true
			AdminStatus.IsShellMode <- true
		case "download":
			if len(AdminCommand) == 2 {
				go common.DownloadFile(AdminCommand[1], startNodeConn, nodeID, 0, AdminStatus.AESKey)
			} else {
				fmt.Println("Bad format! Should be download [filename]")
			}
			AdminStatus.ReadyChange <- true
			AdminStatus.IsShellMode <- true
		case "forward":
			if len(AdminCommand) == 3 {
				go StartPortForwardForClient(AdminCommand, startNodeConn, nodeID, AdminStatus.AESKey)
			} else {
				fmt.Println("Bad format! Should be forward [localport] [rhostip]:[rhostport]")
			}
			AdminStatus.ReadyChange <- true
			AdminStatus.IsShellMode <- true
		case "stopforward":
			go StopForward()
			AdminStatus.ReadyChange <- true
			AdminStatus.IsShellMode <- true
		case "reflect":
			if len(AdminCommand) == 3 {
				go StartReflectForClient(AdminCommand, startNodeConn, nodeID, AdminStatus.AESKey)
			} else {
				fmt.Println("Bad format! Should be reflect [rhostport] [localport]")
			}
			AdminStatus.ReadyChange <- true
			AdminStatus.IsShellMode <- true
		case "stopreflect":
			go StopReflect(startNodeConn, nodeID)
			AdminStatus.ReadyChange <- true
			AdminStatus.IsShellMode <- true
		case "recover":
			log.Println("[*]Recover message sent! Now you can manipulate node ", nodeID+1)
			respCommand, _ := common.ConstructPayload(nodeID, "COMMAND", "RECOVER", " ", " ", 0, 0, AdminStatus.AESKey, false)
			startNodeConn.Write(respCommand)
			AdminStatus.ReadyChange <- true
			AdminStatus.IsShellMode <- true
		case "addnote":
			ok := AddNote(AdminCommand, nodeID)
			if ok {
				log.Println("[*]Description added successfully!")
			} else {
				log.Println("[*]Cannot find node ", nodeID)
			}
			AdminStatus.ReadyChange <- true
			AdminStatus.IsShellMode <- true
		case "delnote":
			ok := DelNote(nodeID)
			if ok {
				log.Println("[*]Description deleted successfully!")
			} else {
				log.Println("[*]Cannot find node ", nodeID)
			}
			AdminStatus.ReadyChange <- true
			AdminStatus.IsShellMode <- true
		case "help":
			ShowNodeHelp()
			AdminStatus.ReadyChange <- true
			AdminStatus.IsShellMode <- true
		case "":
			AdminStatus.ReadyChange <- true
			AdminStatus.IsShellMode <- true
			continue
		case "exit":
			*CliStatus = "admin"
			AdminStatus.ReadyChange <- true
			AdminStatus.IsShellMode <- true
			return
		default:
			fmt.Println("Illegal command, enter help to get available commands")
			AdminStatus.ReadyChange <- true
			AdminStatus.IsShellMode <- true
		}
	}
}

/*-------------------------Shell模式下相关代码--------------------------*/
//处理shell开启时的输入
func HandleShellToNode(startNodeControlConn net.Conn, nodeID uint32) {
	inputReader := bufio.NewReader(os.Stdin)
	for {
		command, err := inputReader.ReadString('\n')
		if runtime.GOOS == "windows" {
			command = strings.Replace(command, "\r", "", -1)
		}
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		switch command {
		case "exit\n":
			if nodeID == 1 {
				*CliStatus = "startnode"
			} else {
				*CliStatus = "node " + fmt.Sprint(nodeID)
			}
			respCommand, _ := common.ConstructPayload(nodeID, "COMMAND", "SHELL", " ", command, 0, 0, AdminStatus.AESKey, false)
			startNodeControlConn.Write(respCommand)
			AdminStatus.ReadyChange <- true
			AdminStatus.IsShellMode <- true
			return
		default:
			respCommand, _ := common.ConstructPayload(nodeID, "COMMAND", "SHELL", " ", command, 0, 0, AdminStatus.AESKey, false)
			startNodeControlConn.Write(respCommand)
		}
	}
}

/*-------------------------Ssh模式下相关代码--------------------------*/
//处理ssh开启时的输入
func HandleSSHToNode(startNodeControlConn net.Conn, nodeID uint32) {
	inputReader := bufio.NewReader(os.Stdin)
	log.Println("[*]Waiting for response,please be patient")
	if conrinueornot := <-AdminStatus.SshSuccess; conrinueornot {
		fmt.Print("(ssh mode)>>>")
		for {
			command, err := inputReader.ReadString('\n')
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			switch command {
			case "exit\n":
				if nodeID == 1 {
					*CliStatus = "startnode"
				} else {
					*CliStatus = "node " + fmt.Sprint(nodeID)
				}
				respCommand, _ := common.ConstructPayload(nodeID, "COMMAND", "SSHCOMMAND", " ", command, 0, 0, AdminStatus.AESKey, false)
				startNodeControlConn.Write(respCommand)
				AdminStatus.ReadyChange <- true
				AdminStatus.IsShellMode <- true
				return
			case "\n":
				fmt.Print("(ssh mode)>>>")
			default:
				respCommand, _ := common.ConstructPayload(nodeID, "COMMAND", "SSHCOMMAND", " ", command, 0, 0, AdminStatus.AESKey, false)
				startNodeControlConn.Write(respCommand)

			}
		}
	} else {
		return
	}
}

/*------------------------- admin模式下相关代码--------------------------*/
// 处理admin模式下用户的输入及由admin发往startnode的控制信号
func HandleCommandToControlConn(startNodeControlConn net.Conn) {
	for {
		AdminCommand := <-AdminStuff.AdminCommandChan
		switch AdminCommand[0] {
		case "use":
			if len(AdminCommand) == 2 {
				if AdminStuff.StartNode == "0.0.0.0" {
					fmt.Println("There are no nodes connected!")
					AdminStatus.ReadyChange <- true
					AdminStatus.IsShellMode <- true
				} else if AdminCommand[1] == "1" {
					*CliStatus = "startnode"
					AdminStatus.ReadyChange <- true
					AdminStatus.IsShellMode <- true
					HandleNodeCommand(startNodeControlConn, AdminCommand[1])
				} else {
					if len(NodeStatus.Nodes) == 0 {
						fmt.Println("There is no node", AdminCommand[1])
						AdminStatus.ReadyChange <- true
						AdminStatus.IsShellMode <- true
					} else {
						key, _ := strconv.ParseInt(AdminCommand[1], 10, 32)
						if _, ok := NodeStatus.Nodes[uint32(key)]; ok {
							*CliStatus = "node " + AdminCommand[1]
							AdminStatus.ReadyChange <- true
							AdminStatus.IsShellMode <- true
							HandleNodeCommand(startNodeControlConn, AdminCommand[1])
						} else {
							fmt.Println("There is no node", AdminCommand[1])
							AdminStatus.ReadyChange <- true
							AdminStatus.IsShellMode <- true
						}
					}
				}
			} else {
				fmt.Println("Bad format!")
				AdminStatus.ReadyChange <- true
				AdminStatus.IsShellMode <- true
			}
		case "chain":
			ShowChain()
			AdminStatus.ReadyChange <- true
			AdminStatus.IsShellMode <- true
		case "help":
			ShowMainHelp()
			AdminStatus.ReadyChange <- true
			AdminStatus.IsShellMode <- true
		case "":
			AdminStatus.ReadyChange <- true
			AdminStatus.IsShellMode <- true
			continue
		case "exit":
			log.Println("[*]BYE!")
			SendOffLineToStartNode(startNodeControlConn)
			os.Exit(0)
			return
		default:
			fmt.Println("Illegal command, enter help to get available commands")
			AdminStatus.ReadyChange <- true
			AdminStatus.IsShellMode <- true
		}
	}
}
