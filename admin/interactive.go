package admin

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"runtime"
	"strings"

	"Stowaway/share"
	"Stowaway/utils"
)

/*-------------------------控制台相关代码--------------------------*/

// Controlpanel 启动控制台
func Controlpanel(adminCommandChan chan []string) {
	var command string

	inputReader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("(%s) >> ", *AdminStatus.CliStatus)
		input, err := inputReader.ReadString('\n')
		if err != nil {
			fmt.Println(err)
			continue
		}

		command = strings.TrimRight(input," \t\r\n")

		execCommand := strings.Split(command, " ")
		adminCommandChan <- execCommand

		<-AdminStatus.ReadyChange
		<-AdminStatus.IsShellMode
	}
}

/*------------------------- admin模式下相关代码--------------------------*/

// HandleCommandToControlConn 处理admin模式下用户的输入及由admin发往startnode的控制信号
func HandleCommandToControlConn(topology *Topology, startNodeControlConn net.Conn, adminCommandChan chan []string) {
	for {
		AdminCommand := <-adminCommandChan
		switch AdminCommand[0] {
		case "use":
			if len(AdminCommand) == 2 {
				if AdminStatus.StartNode == "offline" {
					fmt.Println("[*]There are no nodes connected!")
					CommandContinue()
				} else if AdminCommand[1] == "1" {
					*AdminStatus.CliStatus = "startnode"
					CommandContinue()
					currentid, _ := FindNumByNodeid(AdminCommand[1])
					AdminStatus.HandleNode = currentid
					HandleNodeCommand(startNodeControlConn, currentid, adminCommandChan)
				} else {
					if len(AdminStuff.NodeStatus.NodeIP) == 0 {
						fmt.Println("[*]There is no node", AdminCommand[1])
						CommandContinue()
					} else {
						currentid, err := FindNumByNodeid(AdminCommand[1])
						if err != nil {
							fmt.Println("[*]There is no node", AdminCommand[1])
							CommandContinue()
							continue
						}
						if _, ok := AdminStuff.NodeStatus.NodeIP[currentid]; ok {
							*AdminStatus.CliStatus = "node " + AdminCommand[1]
							CommandContinue()
							AdminStatus.HandleNode = currentid
							HandleNodeCommand(startNodeControlConn, currentid, adminCommandChan)
						} else {
							fmt.Println("[*]There is no node", AdminCommand[1])
							CommandContinue()
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
			topology.ShowTree()
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
func HandleNodeCommand(startNodeConn net.Conn, nodeid string, adminCommandChan chan []string) {
	route := utils.GetInfoViaLockMap(Route, nodeid).(string)

	for {
		AdminCommand := <-adminCommandChan
		switch AdminCommand[0] {
		case "shell":
			err := utils.ConstructPayloadAndSend(startNodeConn, nodeid, route, "COMMAND", "SHELL", " ", "", 0, utils.AdminId, AdminStatus.AESKey, false)
			if err != nil {
				log.Printf("[*]ERROR OCCURED!: %s", err)
			}
			if suc := <-AdminStatus.ShellSuccess; suc {
				HandleShellToNode(startNodeConn, nodeid)
			} else {
				log.Println("[*]Cannot start the shell!")
			}
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
			err := utils.ConstructPayloadAndSend(startNodeConn, nodeid, route, "COMMAND", "SOCKS", " ", socksStartData, 0, utils.AdminId, AdminStatus.AESKey, false)
			if err != nil {
				log.Println("[*]StartNode seems offline")
				*AdminStatus.CliStatus = "admin"
				CommandContinue()
				return
			}
			if <-AdminStatus.NodeSocksStarted {
				go StartSocksServiceForClient(AdminCommand, startNodeConn, nodeid)
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
					go StartSSHService(startNodeConn, command, nodeid, method)
					HandleSSHToNode(startNodeConn, nodeid)
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
						go StartSSHService(startNodeConn, command, nodeid, method)
						HandleSSHToNode(startNodeConn, nodeid)
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
					go SendSSHTunnel(startNodeConn, command, nodeid, method)
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
					go SendSSHTunnel(startNodeConn, command, nodeid, method)
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
					utils.ConstructPayloadAndSend(startNodeConn, nodeid, route, "COMMAND", "CONNECT", " ", data, 0, utils.AdminId, AdminStatus.AESKey, false)
					break
				}
			} else {
				fmt.Println("Bad format! Should be connect [ip:port]")
			}
			CommandContinue()
		case "listen":
			if len(AdminCommand) == 2 {
				address,_,err := utils.CheckIPPort(AdminCommand[1])	
				if err != nil {
					fmt.Println("[*]Bad format! Should be listen [port],and port must between 1~65535")
					CommandContinue()
					continue
				}
				utils.ConstructPayloadAndSend(startNodeConn, nodeid, route, "COMMAND", "LISTEN", " ", address, 0, utils.AdminId, AdminStatus.AESKey, false)
			} else {
				fmt.Println("[*]Bad format! Should be listen [port]")
			}
			CommandContinue()
		case "upload":
			if len(AdminCommand) == 2 {
				go share.UploadFile(route, AdminCommand[1], &startNodeConn, nodeid, AdminStatus.GetName, AdminStatus.AESKey, utils.AdminId, true)
			} else {
				fmt.Println("[*]Bad format! Should be upload [filename]")
			}
			CommandContinue()
		case "download":
			if len(AdminCommand) == 2 {
				go share.DownloadFile(route, AdminCommand[1], startNodeConn, nodeid, utils.AdminId, AdminStatus.AESKey)
			} else {
				fmt.Println("[*]Bad format! Should be download [filename]")
			}
			CommandContinue()
		case "forward":
			if len(AdminCommand) == 3 {
				go StartPortForwardForClient(AdminCommand, startNodeConn, nodeid, AdminStatus.AESKey)
			} else {
				fmt.Println("[*]Bad format! Should be forward [localport] [rhostip]:[rhostport]")
			}
			CommandContinue()
		case "stopforward":
			StopForward()
			CommandContinue()
		case "reflect":
			if len(AdminCommand) == 3 {
				go StartReflectForClient(AdminCommand, startNodeConn, nodeid, AdminStatus.AESKey)
			} else {
				fmt.Println("[*]Bad format! Should be reflect [rhostport] [localport]")
			}
			CommandContinue()
		case "stopreflect":
			go StopReflect(startNodeConn, nodeid)
			CommandContinue()
		case "addnote":
			ok := AddNote(startNodeConn, AdminCommand, nodeid)
			if ok {
				log.Println("[*]Description added successfully!")
			} else {
				log.Println("[*]Cannot find node ", FindIntByNodeid(nodeid)+1)
			}
			CommandContinue()
		case "delnote":
			ok := DelNote(startNodeConn, nodeid)
			if ok {
				log.Println("[*]Description deleted successfully!")
			} else {
				log.Println("[*]Cannot find node ", FindIntByNodeid(nodeid)+1)
			}
			CommandContinue()
		case "offline":
			go SendOfflineMess(startNodeConn, nodeid)
			CommandContinue()
		case "help":
			ShowNodeHelp()
			CommandContinue()
		case "":
			CommandContinue()
			continue
		case "exit":
			*AdminStatus.CliStatus = "admin"
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
func HandleShellToNode(startNodeControlConn net.Conn, nodeid string) {
	route := utils.GetInfoViaLockMap(Route, nodeid).(string)
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
			if nodeid == utils.StartNodeId {
				*AdminStatus.CliStatus = "startnode"
			} else {
				*AdminStatus.CliStatus = "node " + fmt.Sprint(FindIntByNodeid(nodeid)+1)
			}
			utils.ConstructPayloadAndSend(startNodeControlConn, nodeid, route, "COMMAND", "SHELL", " ", command, 0, utils.AdminId, AdminStatus.AESKey, false)
			CommandContinue()
			return
		default:
			utils.ConstructPayloadAndSend(startNodeControlConn, nodeid, route, "COMMAND", "SHELL", " ", command, 0, utils.AdminId, AdminStatus.AESKey, false)
		}
	}
}

/*-------------------------Ssh模式下相关代码--------------------------*/

// HandleSSHToNode 处理ssh开启时的输入
func HandleSSHToNode(startNodeControlConn net.Conn, nodeid string) {
	route := utils.GetInfoViaLockMap(Route, nodeid).(string)
	inputReader := bufio.NewReader(os.Stdin)

	log.Println("[*]Waiting for response,please be patient")

	if continueOrNot := <-AdminStatus.SSHSuccess; continueOrNot {
		fmt.Print("(ssh mode)>>>")
		for {
			command, err := inputReader.ReadString('\n')
			if err != nil {
				fmt.Println(err)
				continue
			}
			switch command {
			case "exit\n":
				if nodeid == utils.StartNodeId {
					*AdminStatus.CliStatus = "startnode"
				} else {
					*AdminStatus.CliStatus = "node " + fmt.Sprint(FindIntByNodeid(nodeid)+1)
				}
				utils.ConstructPayloadAndSend(startNodeControlConn, nodeid, route, "COMMAND", "SSHCOMMAND", " ", command, 0, utils.AdminId, AdminStatus.AESKey, false)
				CommandContinue()
				return
			case "\n":
				fmt.Print("(ssh mode)>>>")
			default:
				utils.ConstructPayloadAndSend(startNodeControlConn, nodeid, route, "COMMAND", "SSHCOMMAND", " ", command, 0, utils.AdminId, AdminStatus.AESKey, false)
			}
		}
	} else {
		return
	}
}
