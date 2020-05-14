package admin

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"

	"Stowaway/share"
	"Stowaway/utils"
)

var CurrentNode string

/*-------------------------控制台相关代码--------------------------*/

// Controlpanel 启动控制台
func Controlpanel() {
	var command string

	inputReader := bufio.NewReader(os.Stdin)
	platform := utils.CheckSystem()

	for {
		fmt.Printf("(%s) >> ", *CliStatus)
		input, err := inputReader.ReadString('\n')
		if err != nil {
			fmt.Println(err)
			continue
		}
		if platform == 0x01 {
			command = strings.Replace(input, "\r\n", "", -1)
		} else {
			command = strings.Replace(input, "\n", "", -1)
		}

		execCommand := strings.Split(command, " ")
		AdminStuff.AdminCommandChan <- execCommand

		<-AdminStatus.ReadyChange
		<-AdminStatus.IsShellMode
	}
}

/*------------------------- admin模式下相关代码--------------------------*/

// HandleCommandToControlConn 处理admin模式下用户的输入及由admin发往startnode的控制信号
func HandleCommandToControlConn(startNodeControlConn net.Conn) {
	for {
		AdminCommand := <-AdminStuff.AdminCommandChan
		switch AdminCommand[0] {
		case "use":
			if len(AdminCommand) == 2 {
				if AdminStuff.StartNode == "0.0.0.0" {
					fmt.Println("[*]There are no nodes connected!")
					CommandContinue()
				} else if AdminCommand[1] == "1" {
					*CliStatus = "startnode"
					CommandContinue()
					currentid, _ := FindNumByNodeid(AdminCommand[1])
					AdminStatus.HandleNode = currentid
					HandleNodeCommand(startNodeControlConn, currentid)
				} else {
					if len(NodeStatus.NodeIP) == 0 {
						fmt.Println("[*]There is no node", AdminCommand[1])
						CommandContinue()
					} else {
						currentid, err := FindNumByNodeid(AdminCommand[1])
						if err != nil {
							fmt.Println("[*]There is no node", AdminCommand[1])
							CommandContinue()
							continue
						}
						if _, ok := NodeStatus.NodeIP[currentid]; ok {
							*CliStatus = "node " + AdminCommand[1]
							CommandContinue()
							AdminStatus.HandleNode = currentid
							HandleNodeCommand(startNodeControlConn, currentid)
						} else {
							fmt.Println("[*]There is no node", AdminCommand[1])
							CommandContinue()
							AdminStatus.IsShellMode <- true
						}
					}
				}
			} else {
				fmt.Println("[*]Bad format!")
				CommandContinue()
			}
		case "detail":
			ShowDetail()
			CommandContinue()
		case "tree":
			ShowTree()
			CommandContinue()
		case "help":
			ShowMainHelp()
			CommandContinue()
		case "":
			CommandContinue()
			continue
		case "exit":
			log.Println("[*]BYE!")
			os.Exit(0)
			return
		default:
			fmt.Println("[*]Illegal command, enter help to get available commands")
			CommandContinue()
		}
	}
}

/*-------------------------Node模式下相关代码--------------------------*/

// HandleNodeCommand 处理node模式下用户的输入
func HandleNodeCommand(startNodeConn net.Conn, nodeID string) {
	CurrentNode = nodeID //把nodeid提取出来，以供上传/下载文件功能使用

	route := utils.GetInfoViaLockMap(Route, nodeID).(string)

	for {
		AdminCommand := <-AdminStuff.AdminCommandChan
		switch AdminCommand[0] {
		case "shell":
			err := utils.ConstructPayloadAndSend(startNodeConn, nodeID, route, "COMMAND", "SHELL", " ", "", 0, utils.AdminId, AdminStatus.AESKey, false)
			if err != nil {
				log.Printf("[*]ERROR OCCURED!: %s", err)
			}
			HandleShellToNode(startNodeConn, nodeID)
		case "socks":
			var socksStartData string
			switch len(AdminCommand) {
			case 2:
				socksStartData = fmt.Sprintf("%s:::%s:::%s", AdminCommand[1], "", "")
			case 3:
				fmt.Println("Illegal username/password! Try again!")
				CommandContinue()
				continue
			case 4:
				socksStartData = fmt.Sprintf("%s:::%s:::%s", AdminCommand[1], AdminCommand[2], AdminCommand[3])
			default:
				fmt.Println("[*]Illegal format! Should be socks [lport] (username) (password) ps:username and password are optional ")
				CommandContinue()
				continue
			}
			err := utils.ConstructPayloadAndSend(startNodeConn, nodeID, route, "COMMAND", "SOCKS", " ", socksStartData, 0, utils.AdminId, AdminStatus.AESKey, false)
			if err != nil {
				log.Println("[*]StartNode seems offline")
				*CliStatus = "admin"
				CommandContinue()
				return
			}
			if <-AdminStatus.NodeSocksStarted {
				go StartSocksServiceForClient(AdminCommand, startNodeConn, nodeID)
			}
			CommandContinue()
		case "stopsocks":
			StopSocks()
			CommandContinue()
		case "ssh":
			var command []string
			if len(AdminCommand) == 2 {
				fmt.Print("[*]Please choose the auth method(1.username/password 2.certificate):")
				input := ReadChoice()
				switch input {
				case "1":
					method := input
					command = append(command, AdminCommand[1])
					fmt.Print("[*]Please enter the username:")
					input = ReadChoice()
					command = append(command, input)
					fmt.Print("[*]Please enter the password:")
					input = ReadChoice()
					command = append(command, input)
					go StartSSHService(startNodeConn, command, nodeID, method)
					HandleSSHToNode(startNodeConn, nodeID)
				case "2":
					method := input
					command = append(command, AdminCommand[1])
					fmt.Print("[*]Please enter the username:")
					input = ReadChoice()
					command = append(command, input)
					fmt.Print("[*]Please enter the file path of the key:")
					input = ReadChoice()
					result := CheckKeyFile(input)
					if result == nil {
						fmt.Println("[*]Cannot find the key file!")
						CommandContinue()
						continue
					} else {
						command = append(command, string(result))
						go StartSSHService(startNodeConn, command, nodeID, method)
						HandleSSHToNode(startNodeConn, nodeID)
					}
				default:
					fmt.Println("[*]Wrong answer! Should be 1 or 2")
					CommandContinue()
					continue
				}
			} else {
				fmt.Println("Bad format! Should be ssh [ip:port]")
				CommandContinue()
			}
		case "sshtunnel":
			var command []string
			if len(AdminCommand) == 3 {
				command = append(command, AdminCommand[1])
				fmt.Print("[*]Please choose the auth method(1.username/password 2.certificate):")
				input := ReadChoice()
				switch input {
				case "1":
					method := input
					fmt.Print("[*]Please enter the username:")
					input = ReadChoice()
					command = append(command, input)
					fmt.Print("[*]Please enter the password:")
					input = ReadChoice()
					command = append(command, []string{input, AdminCommand[2]}...)
					go SendSSHTunnel(startNodeConn, command, nodeID, method)
				case "2":
					method := input
					fmt.Print("[*]Please enter the username:")
					input = ReadChoice()
					command = append(command, input)
					fmt.Print("[*]Please enter the file path of the key:")
					input = ReadChoice()
					result := CheckKeyFile(input)
					if result == nil {
						fmt.Println("[*]Cannot find the key file!")
						CommandContinue()
						continue
					}
					command = append(command, []string{string(result), AdminCommand[2]}...)
					go SendSSHTunnel(startNodeConn, command, nodeID, method)
				default:
					fmt.Println("[*]Wrong answer! Should be 1 or 2")
					CommandContinue()
					continue
				}
			} else {
				fmt.Println("Bad format! Should be sshtunnel [ip:port] [agent-listening port]")
				CommandContinue()
			}
		case "connect":
			if len(AdminCommand) == 2 {
				for {
					fmt.Print("[*]Is the node you want to connect reusing the port?(1.Yes/2.No):") //判断reuse或者不是，调用不同的函数
					choice := ReadChoice()

					if choice != "1" && choice != "2" {
						fmt.Println("[*]You should type in 1 or 2!")
						continue
					}

					data := AdminCommand[1] + ":::" + choice
					utils.ConstructPayloadAndSend(startNodeConn, nodeID, route, "COMMAND", "CONNECT", " ", data, 0, utils.AdminId, AdminStatus.AESKey, false)
					break
				}
			} else {
				fmt.Println("Bad format! Should be connect [ip:port]")
			}
			CommandContinue()
		case "listen":
			if len(AdminCommand) == 2 {
				port, err := strconv.Atoi(AdminCommand[1])
				if err != nil || port < 0 || port > 65535 {
					fmt.Println("[*]Bad format! Should be listen [port],and port must between 1~65535")
					CommandContinue()
					continue
				}
				utils.ConstructPayloadAndSend(startNodeConn, nodeID, route, "COMMAND", "LISTEN", " ", AdminCommand[1], 0, utils.AdminId, AdminStatus.AESKey, false)
			} else {
				fmt.Println("[*]Bad format! Should be listen [port]")
			}
			CommandContinue()
		case "upload":
			if len(AdminCommand) == 2 {
				go share.UploadFile(route, AdminCommand[1], &startNodeConn, nodeID, AdminStatus.GetName, AdminStatus.AESKey, utils.AdminId, true)
			} else {
				fmt.Println("[*]Bad format! Should be upload [filename]")
			}
			CommandContinue()
		case "download":
			if len(AdminCommand) == 2 {
				go share.DownloadFile(route, AdminCommand[1], startNodeConn, nodeID, utils.AdminId, AdminStatus.AESKey)
			} else {
				fmt.Println("[*]Bad format! Should be download [filename]")
			}
			CommandContinue()
		case "forward":
			if len(AdminCommand) == 3 {
				go StartPortForwardForClient(AdminCommand, startNodeConn, nodeID, AdminStatus.AESKey)
			} else {
				fmt.Println("[*]Bad format! Should be forward [localport] [rhostip]:[rhostport]")
			}
			CommandContinue()
		case "stopforward":
			StopForward()
			CommandContinue()
		case "reflect":
			if len(AdminCommand) == 3 {
				go StartReflectForClient(AdminCommand, startNodeConn, nodeID, AdminStatus.AESKey)
			} else {
				fmt.Println("[*]Bad format! Should be reflect [rhostport] [localport]")
			}
			CommandContinue()
		case "stopreflect":
			go StopReflect(startNodeConn, nodeID)
			CommandContinue()
		case "addnote":
			ok := AddNote(startNodeConn, AdminCommand, nodeID)
			if ok {
				log.Println("[*]Description added successfully!")
			} else {
				log.Println("[*]Cannot find node ", FindIntByNodeid(nodeID)+1)
			}
			CommandContinue()
		case "delnote":
			ok := DelNote(startNodeConn, nodeID)
			if ok {
				log.Println("[*]Description deleted successfully!")
			} else {
				log.Println("[*]Cannot find node ", FindIntByNodeid(nodeID)+1)
			}
			CommandContinue()
		case "help":
			ShowNodeHelp()
			CommandContinue()
		case "":
			CommandContinue()
			continue
		case "exit":
			*CliStatus = "admin"
			CommandContinue()
			return
		default:
			fmt.Println("[*]Illegal command, enter help to get available commands")
			CommandContinue()
		}
	}
}

/*-------------------------Shell模式下相关代码--------------------------*/

// HandleShellToNode 处理shell开启时的输入
func HandleShellToNode(startNodeControlConn net.Conn, nodeID string) {
	route := utils.GetInfoViaLockMap(Route, nodeID).(string)
	inputReader := bufio.NewReader(os.Stdin)

	for {
		command, err := inputReader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}
		if runtime.GOOS == "windows" {
			command = strings.Replace(command, "\r", "", -1)
		}

		switch command {
		case "exit\n":
			if nodeID == utils.StartNodeId {
				*CliStatus = "startnode"
			} else {
				*CliStatus = "node " + fmt.Sprint(FindIntByNodeid(nodeID)+1)
			}
			utils.ConstructPayloadAndSend(startNodeControlConn, nodeID, route, "COMMAND", "SHELL", " ", command, 0, utils.AdminId, AdminStatus.AESKey, false)
			CommandContinue()
			return
		default:
			utils.ConstructPayloadAndSend(startNodeControlConn, nodeID, route, "COMMAND", "SHELL", " ", command, 0, utils.AdminId, AdminStatus.AESKey, false)
		}
	}
}

/*-------------------------Ssh模式下相关代码--------------------------*/

// HandleSSHToNode 处理ssh开启时的输入
func HandleSSHToNode(startNodeControlConn net.Conn, nodeID string) {
	route := utils.GetInfoViaLockMap(Route, nodeID).(string)
	inputReader := bufio.NewReader(os.Stdin)

	log.Println("[*]Waiting for response,please be patient")

	if conrinueornot := <-AdminStatus.SSHSuccess; conrinueornot {
		fmt.Print("(ssh mode)>>>")
		for {
			command, err := inputReader.ReadString('\n')
			if err != nil {
				fmt.Println(err)
				continue
			}
			switch command {
			case "exit\n":
				if nodeID == utils.StartNodeId {
					*CliStatus = "startnode"
				} else {
					*CliStatus = "node " + fmt.Sprint(FindIntByNodeid(nodeID)+1)
				}
				utils.ConstructPayloadAndSend(startNodeControlConn, nodeID, route, "COMMAND", "SSHCOMMAND", " ", command, 0, utils.AdminId, AdminStatus.AESKey, false)
				CommandContinue()
				return
			case "\n":
				fmt.Print("(ssh mode)>>>")
			default:
				utils.ConstructPayloadAndSend(startNodeControlConn, nodeID, route, "COMMAND", "SSHCOMMAND", " ", command, 0, utils.AdminId, AdminStatus.AESKey, false)
			}
		}
	} else {
		return
	}
}
