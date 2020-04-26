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
// 启动控制台
func Controlpanel() {
	inputReader := bufio.NewReader(os.Stdin)
	var command string
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
// 处理admin模式下用户的输入及由admin发往startnode的控制信号
func HandleCommandToControlConn(startNodeControlConn net.Conn) {
	for {
		AdminCommand := <-AdminStuff.AdminCommandChan
		switch AdminCommand[0] {
		case "use":
			if len(AdminCommand) == 2 {
				if AdminStuff.StartNode == "0.0.0.0" {
					fmt.Println("[*]There are no nodes connected!")
					AdminStatus.ReadyChange <- true
					AdminStatus.IsShellMode <- true
				} else if AdminCommand[1] == "1" {
					*CliStatus = "startnode"
					AdminStatus.ReadyChange <- true
					AdminStatus.IsShellMode <- true
					currentid, _ := FindNumByNodeid(AdminCommand[1])
					AdminStatus.HandleNode = currentid
					HandleNodeCommand(startNodeControlConn, currentid)
				} else {
					if len(NodeStatus.NodeIP) == 0 {
						fmt.Println("[*]There is no node", AdminCommand[1])
						AdminStatus.ReadyChange <- true
						AdminStatus.IsShellMode <- true
					} else {
						currentid, err := FindNumByNodeid(AdminCommand[1])
						if err != nil {
							fmt.Println("[*]There is no node", AdminCommand[1])
							AdminStatus.ReadyChange <- true
							AdminStatus.IsShellMode <- true
							continue
						}
						if _, ok := NodeStatus.NodeIP[currentid]; ok {
							*CliStatus = "node " + AdminCommand[1]
							AdminStatus.ReadyChange <- true
							AdminStatus.IsShellMode <- true
							AdminStatus.HandleNode = currentid
							HandleNodeCommand(startNodeControlConn, currentid)
						} else {
							fmt.Println("[*]There is no node", AdminCommand[1])
							AdminStatus.ReadyChange <- true
							AdminStatus.IsShellMode <- true
						}
					}
				}
			} else {
				fmt.Println("[*]Bad format!")
				AdminStatus.ReadyChange <- true
				AdminStatus.IsShellMode <- true
			}
		case "detail":
			ShowDetail()
			AdminStatus.ReadyChange <- true
			AdminStatus.IsShellMode <- true
		case "tree":
			ShowTree()
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
			os.Exit(0)
			return
		default:
			fmt.Println("[*]Illegal command, enter help to get available commands")
			AdminStatus.ReadyChange <- true
			AdminStatus.IsShellMode <- true
		}
	}
}

/*-------------------------Node模式下相关代码--------------------------*/
//处理node模式下用户的输入
func HandleNodeCommand(startNodeConn net.Conn, nodeID string) {
	CurrentNode = nodeID //把nodeid提取出来，以供上传/下载文件功能使用

	Route.Lock()
	route := Route.Route[nodeID]
	Route.Unlock()

	for {
		AdminCommand := <-AdminStuff.AdminCommandChan
		switch AdminCommand[0] {
		case "shell":
			respCommand, err := utils.ConstructPayload(nodeID, route, "COMMAND", "SHELL", " ", "", 0, utils.AdminId, AdminStatus.AESKey, false)
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
				continue
			} else if len(AdminCommand) == 4 {
				socksStartData = fmt.Sprintf("%s:::%s:::%s", AdminCommand[1], AdminCommand[2], AdminCommand[3])
			} else {
				fmt.Println("[*]Illegal format! Should be socks [lport] (username) (password) ps:username and password are optional ")
				AdminStatus.ReadyChange <- true
				AdminStatus.IsShellMode <- true
				continue
			}
			respCommand, err := utils.ConstructPayload(nodeID, route, "COMMAND", "SOCKS", " ", socksStartData, 0, utils.AdminId, AdminStatus.AESKey, false)
			_, err = startNodeConn.Write(respCommand)
			if err != nil {
				log.Println("[*]StartNode seems offline")
				*CliStatus = "admin"
				AdminStatus.ReadyChange <- true
				AdminStatus.IsShellMode <- true
				return
			}
			if <-AdminStatus.NodeSocksStarted {
				go StartSocksServiceForClient(AdminCommand, startNodeConn, nodeID)
			}
			AdminStatus.ReadyChange <- true
			AdminStatus.IsShellMode <- true
		case "stopsocks":
			StopSocks()
			AdminStatus.ReadyChange <- true
			AdminStatus.IsShellMode <- true
		case "ssh":
			if len(AdminCommand) == 2 {
				fmt.Print("[*]Please choose the auth method(1.username/password 2.certificate):")
				inputReader := bufio.NewReader(os.Stdin)
				input, _ := inputReader.ReadString('\n')
				input = CheckInput(input)
				if input != "1" && input != "2" {
					fmt.Println("[*]Wrong answer! Should be 1 or 2")
					AdminStatus.ReadyChange <- true
					AdminStatus.IsShellMode <- true
					continue
				} else if input == "1" {
					var command []string
					method := input
					command = append(command, AdminCommand[1])
					fmt.Print("[*]Please enter the username:")
					input, _ = inputReader.ReadString('\n')
					input = CheckInput(input)
					command = append(command, input)
					fmt.Print("[*]Please enter the password:")
					input, _ = inputReader.ReadString('\n')
					input = CheckInput(input)
					command = append(command, input)
					go StartSSHService(startNodeConn, command, nodeID, method)
					HandleSSHToNode(startNodeConn, nodeID)
				} else if input == "2" {
					var command []string
					method := input
					command = append(command, AdminCommand[1])
					fmt.Print("[*]Please enter the username:")
					input, _ = inputReader.ReadString('\n')
					input = CheckInput(input)
					command = append(command, input)
					fmt.Print("[*]Please enter the file path of the key:")
					input, _ = inputReader.ReadString('\n')
					input = CheckInput(input)
					result := CheckKeyFile(input)
					if result == nil {
						fmt.Println("[*]Cannot find the key file!")
						AdminStatus.ReadyChange <- true
						AdminStatus.IsShellMode <- true
						continue
					} else {
						command = append(command, string(result))
						go StartSSHService(startNodeConn, command, nodeID, method)
						HandleSSHToNode(startNodeConn, nodeID)
					}
				}
			} else {
				fmt.Println("Bad format! Should be ssh [ip:port]")
				AdminStatus.ReadyChange <- true
				AdminStatus.IsShellMode <- true
			}
		case "sshtunnel":
			if len(AdminCommand) == 3 {
				var command []string
				command = append(command, AdminCommand[1])
				inputReader := bufio.NewReader(os.Stdin)
				fmt.Print("[*]Please choose the auth method(1.username/password 2.certificate):")
				input, _ := inputReader.ReadString('\n')
				input = CheckInput(input)
				if input != "1" && input != "2" {
					fmt.Println("[*]Wrong answer! Should be 1 or 2")
					AdminStatus.ReadyChange <- true
					AdminStatus.IsShellMode <- true
					continue
				} else if input == "1" {
					var command []string
					method := input
					command = append(command, AdminCommand[1])
					fmt.Print("[*]Please enter the username:")
					input, _ = inputReader.ReadString('\n')
					input = CheckInput(input)
					command = append(command, input)
					fmt.Print("[*]Please enter the password:")
					input, _ = inputReader.ReadString('\n')
					input = CheckInput(input)
					command = append(command, input)
					command = append(command, AdminCommand[2])
					go SendSSHTunnel(startNodeConn, command, nodeID, method)
				} else if input == "2" {
					var command []string
					method := input
					command = append(command, AdminCommand[1])
					fmt.Print("[*]Please enter the username:")
					input, _ = inputReader.ReadString('\n')
					input = CheckInput(input)
					command = append(command, input)
					fmt.Print("[*]Please enter the file path of the key:")
					input, _ = inputReader.ReadString('\n')
					input = CheckInput(input)
					result := CheckKeyFile(input)
					if result == nil {
						fmt.Println("[*]Cannot find the key file!")
						AdminStatus.ReadyChange <- true
						AdminStatus.IsShellMode <- true
						continue
					} else {
						command = append(command, string(result))
						command = append(command, AdminCommand[2])
						go SendSSHTunnel(startNodeConn, command, nodeID, method)
					}
				}

			} else {
				fmt.Println("Bad format! Should be sshtunnel [ip:port] [agent-listening port]")
				AdminStatus.ReadyChange <- true
				AdminStatus.IsShellMode <- true
			}
		case "connect":
			if len(AdminCommand) == 2 {
				inputReader := bufio.NewReader(os.Stdin)
				for {
					fmt.Print("[*]Is the node you want to connect reusing the port?(1.Yes/2.No):") //判断reuse或者不是，调用不同的函数
					input, _ := inputReader.ReadString('\n')
					choice := CheckInput(input)
					if choice != "1" && choice != "2" {
						fmt.Println("[*]You should type in 1 or 2!")
						continue
					}
					data := AdminCommand[1] + ":::" + choice
					respCommand, _ := utils.ConstructPayload(nodeID, route, "COMMAND", "CONNECT", " ", data, 0, utils.AdminId, AdminStatus.AESKey, false)
					startNodeConn.Write(respCommand)
					break
				}
			} else {
				fmt.Println("Bad format! Should be connect [ip:port]")
			}
			AdminStatus.ReadyChange <- true
			AdminStatus.IsShellMode <- true
		case "listen":
			if len(AdminCommand) == 2 {
				port, err := strconv.Atoi(AdminCommand[1])
				if err != nil || port < 0 || port > 65535 {
					fmt.Println("[*]Bad format! Should be listen [port],and port must between 1~65535")
					AdminStatus.ReadyChange <- true
					AdminStatus.IsShellMode <- true
					continue
				}
				respCommand, _ := utils.ConstructPayload(nodeID, route, "COMMAND", "LISTEN", " ", AdminCommand[1], 0, utils.AdminId, AdminStatus.AESKey, false)
				startNodeConn.Write(respCommand)
			} else {
				fmt.Println("[*]Bad format! Should be listen [port]")
			}
			AdminStatus.ReadyChange <- true
			AdminStatus.IsShellMode <- true
		case "upload":
			if len(AdminCommand) == 2 {
				go share.UploadFile(route, AdminCommand[1], &startNodeConn, nodeID, AdminStatus.GetName, AdminStatus.AESKey, utils.AdminId, true)
			} else {
				fmt.Println("[*]Bad format! Should be upload [filename]")
			}
			AdminStatus.ReadyChange <- true
			AdminStatus.IsShellMode <- true
		case "download":
			if len(AdminCommand) == 2 {
				go share.DownloadFile(route, AdminCommand[1], startNodeConn, nodeID, utils.AdminId, AdminStatus.AESKey)
			} else {
				fmt.Println("[*]Bad format! Should be download [filename]")
			}
			AdminStatus.ReadyChange <- true
			AdminStatus.IsShellMode <- true
		case "forward":
			if len(AdminCommand) == 3 {
				go StartPortForwardForClient(AdminCommand, startNodeConn, nodeID, AdminStatus.AESKey)
			} else {
				fmt.Println("[*]Bad format! Should be forward [localport] [rhostip]:[rhostport]")
			}
			AdminStatus.ReadyChange <- true
			AdminStatus.IsShellMode <- true
		case "stopforward":
			StopForward()
			AdminStatus.ReadyChange <- true
			AdminStatus.IsShellMode <- true
		case "reflect":
			if len(AdminCommand) == 3 {
				go StartReflectForClient(AdminCommand, startNodeConn, nodeID, AdminStatus.AESKey)
			} else {
				fmt.Println("[*]Bad format! Should be reflect [rhostport] [localport]")
			}
			AdminStatus.ReadyChange <- true
			AdminStatus.IsShellMode <- true
		case "stopreflect":
			go StopReflect(startNodeConn, nodeID)
			AdminStatus.ReadyChange <- true
			AdminStatus.IsShellMode <- true
		case "addnote":
			ok := AddNote(startNodeConn, AdminCommand, nodeID)
			if ok {
				log.Println("[*]Description added successfully!")
			} else {
				log.Println("[*]Cannot find node ", FindIntByNodeid(nodeID)+1)
			}
			AdminStatus.ReadyChange <- true
			AdminStatus.IsShellMode <- true
		case "delnote":
			ok := DelNote(startNodeConn, nodeID)
			if ok {
				log.Println("[*]Description deleted successfully!")
			} else {
				log.Println("[*]Cannot find node ", FindIntByNodeid(nodeID)+1)
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
			fmt.Println("[*]Illegal command, enter help to get available commands")
			AdminStatus.ReadyChange <- true
			AdminStatus.IsShellMode <- true
		}
	}
}

/*-------------------------Shell模式下相关代码--------------------------*/
//处理shell开启时的输入
func HandleShellToNode(startNodeControlConn net.Conn, nodeID string) {
	Route.Lock()
	route := Route.Route[nodeID]
	Route.Unlock()

	inputReader := bufio.NewReader(os.Stdin)
	for {
		command, err := inputReader.ReadString('\n')
		if runtime.GOOS == "windows" {
			command = strings.Replace(command, "\r", "", -1)
		}
		if err != nil {
			fmt.Println(err)
			os.Exit(0)
		}
		switch command {
		case "exit\n":
			if nodeID == utils.StartNodeId {
				*CliStatus = "startnode"
			} else {
				*CliStatus = "node " + fmt.Sprint(FindIntByNodeid(nodeID)+1)
			}
			respCommand, _ := utils.ConstructPayload(nodeID, route, "COMMAND", "SHELL", " ", command, 0, utils.AdminId, AdminStatus.AESKey, false)
			startNodeControlConn.Write(respCommand)
			AdminStatus.ReadyChange <- true
			AdminStatus.IsShellMode <- true
			return
		default:
			respCommand, _ := utils.ConstructPayload(nodeID, route, "COMMAND", "SHELL", " ", command, 0, utils.AdminId, AdminStatus.AESKey, false)
			startNodeControlConn.Write(respCommand)
		}
	}
}

/*-------------------------Ssh模式下相关代码--------------------------*/
//处理ssh开启时的输入
func HandleSSHToNode(startNodeControlConn net.Conn, nodeID string) {
	Route.Lock()
	route := Route.Route[nodeID]
	Route.Unlock()

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
				respCommand, _ := utils.ConstructPayload(nodeID, route, "COMMAND", "SSHCOMMAND", " ", command, 0, utils.AdminId, AdminStatus.AESKey, false)
				startNodeControlConn.Write(respCommand)
				AdminStatus.ReadyChange <- true
				AdminStatus.IsShellMode <- true
				return
			case "\n":
				fmt.Print("(ssh mode)>>>")
			default:
				respCommand, _ := utils.ConstructPayload(nodeID, route, "COMMAND", "SSHCOMMAND", " ", command, 0, utils.AdminId, AdminStatus.AESKey, false)
				startNodeControlConn.Write(respCommand)
			}
		}
	} else {
		return
	}
}
