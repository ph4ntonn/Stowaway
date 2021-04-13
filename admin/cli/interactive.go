/*
 * @Author: ph4ntom
 * @Date: 2021-03-10 18:11:41
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-04-04 15:45:17
 */
package cli

import (
	"Stowaway/admin/handler"
	"Stowaway/admin/manager"
	"Stowaway/admin/topology"
	"Stowaway/protocol"
	"Stowaway/share"
	"Stowaway/utils"
	"bytes"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/eiannone/keyboard"
)

const (
	MAIN = iota
	NODE
)

type Console struct {
	// Admin status
	conn     net.Conn
	topology *topology.Topology
	secret   string
	// console internal elements
	status     string
	ready      chan bool
	getCommand chan string
	shellMode  bool
	sshMode    bool
	nodeMode   bool
	// manager that needs to be shared with main thread
	mgr *manager.Manager
}

func NewConsole() *Console {
	console := new(Console)
	console.status = "(admin) >> "
	console.ready = make(chan bool)
	console.getCommand = make(chan string)
	return console
}

func (console *Console) Init(tTopology *topology.Topology, myManager *manager.Manager, conn net.Conn, secret string) {
	console.conn = conn
	console.secret = secret
	console.topology = tTopology
	console.mgr = myManager
}

func (console *Console) Run() {
	go console.handleMainPanelCommand()
	console.mainPanel() // block admin
}

func (console *Console) mainPanel() {
	var (
		isGoingOn bool
		// serve for arrow left/right
		leftCommand  string
		rightCommand string
	)
	// start history
	history := NewHistory()
	go history.Run()
	// start helper
	helper := NewHelper()
	go helper.Run()
	// init keyEvents
	keysEvents, _ := keyboard.GetKeys(10)
	// BEGIN TO WORK!!!!!!!!
	fmt.Print(console.status)
	for {
		event := <-keysEvents
		if event.Err != nil {
			panic(event.Err)
		}
		// under shell&&ssh mode,we cannot just erase the whole line and reprint,so there are two different ways to handle input
		// BTW,all arrow stuff under shell&&ssh mode will be abandoned
		if (event.Key != keyboard.KeyEnter && event.Rune >= 0x20 && event.Rune <= 0x7F) || event.Key == keyboard.KeySpace {
			if !console.shellMode && !console.sshMode {
				fmt.Print("\r\033[K")
				fmt.Print(console.status)
				// save every single input
				if event.Key == keyboard.KeySpace {
					leftCommand = leftCommand + " "
				} else {
					leftCommand = leftCommand + string(event.Rune)
				}
				// print command && keep cursor at right position
				fmt.Print(leftCommand + rightCommand)
				fmt.Print(string(bytes.Repeat([]byte("\b"), len(rightCommand))))
			} else {
				if event.Key == keyboard.KeySpace {
					leftCommand = leftCommand + " "
				} else {
					leftCommand = leftCommand + string(event.Rune)
				}
				fmt.Print(string(event.Rune))
			}
		} else if event.Key == keyboard.KeyBackspace2 || event.Key == keyboard.KeyBackspace {
			if !console.shellMode && !console.sshMode {
				fmt.Print("\r\033[K")
				fmt.Print(console.status)
				// let leftcommand--
				if len(leftCommand) >= 1 {
					leftCommand = leftCommand[:len(leftCommand)-1]
				}
				fmt.Print(leftCommand + rightCommand)
				fmt.Print(string(bytes.Repeat([]byte("\b"), len(rightCommand))))
			} else {
				if len(leftCommand) >= 1 {
					leftCommand = leftCommand[:len(leftCommand)-1]
					fmt.Print("\b \b")
				}
			}
		} else if event.Key == keyboard.KeyEnter {
			if !console.shellMode && !console.sshMode {
				// when hit enter,then concat left&&right command,create task to record it
				command := leftCommand + rightCommand
				// if command is not "",send it to history
				if command != "" {
					task := &HistoryTask{
						Mode:    RECORD,
						Command: command,
					}
					history.TaskChan <- task
				}
				// no matter what command is,send it to console to parse
				console.getCommand <- command
				// set searching->false
				isGoingOn = false
				// set both left/right command -> "",new start!
				leftCommand = ""
				rightCommand = ""
				// avoid scenario that console.status is printed before it's changed
				<-console.ready
				fmt.Print("\r\n")
				fmt.Print(console.status)
			} else {
				fmt.Print("\r\n")
				console.getCommand <- leftCommand
				leftCommand = ""
			}
		} else if event.Key == keyboard.KeyArrowUp {
			if !console.shellMode && !console.sshMode {
				fmt.Print("\r\033[K")
				fmt.Print(console.status)
				// new task
				task := &HistoryTask{
					Mode:  SEARCH,
					Order: BEGIN,
				}
				// check if search has already begun
				if !isGoingOn {
					history.TaskChan <- task
					isGoingOn = true
				} else {
					task.Order = NEXT
					history.TaskChan <- task
				}
				// get the history command && set rightcommand -> ""
				leftCommand = <-history.ResultChan
				rightCommand = ""
			}
		} else if event.Key == keyboard.KeyArrowDown {
			if !console.shellMode && !console.sshMode {
				fmt.Print("\r\033[K")
				fmt.Print(console.status)
				// check if searching has already begun
				if isGoingOn {
					task := &HistoryTask{
						Mode:  SEARCH,
						Order: PREV,
					}
					history.TaskChan <- task
					leftCommand = <-history.ResultChan
				} else {
					// not started,then just erase user's input and output nothing
					leftCommand = ""
				}
				rightCommand = ""
			}
		} else if event.Key == keyboard.KeyArrowLeft {
			if !console.shellMode && !console.sshMode {
				fmt.Print("\r\033[K")
				fmt.Print(console.status)
				// concat left command's last character with right command
				if len(leftCommand) >= 1 {
					rightCommand = leftCommand[len(leftCommand)-1:] + rightCommand
					leftCommand = leftCommand[:len(leftCommand)-1]
				}

				fmt.Print(leftCommand + rightCommand)
				fmt.Print(string(bytes.Repeat([]byte("\b"), len(rightCommand))))
			}
		} else if event.Key == keyboard.KeyArrowRight {
			if !console.shellMode && !console.sshMode {
				fmt.Print("\r\033[K")
				fmt.Print(console.status)
				// concat right command's first character with left command
				if len(rightCommand) > 1 {
					leftCommand = leftCommand + rightCommand[:1]
					rightCommand = rightCommand[1:]
				} else if len(rightCommand) == 1 {
					leftCommand = leftCommand + rightCommand[:1]
					rightCommand = ""
				}

				fmt.Print(leftCommand + rightCommand)
				fmt.Print(string(bytes.Repeat([]byte("\b"), len(rightCommand))))
			}
		} else if event.Key == keyboard.KeyTab {
			// if user move the cursor or under shellMode(sshMode),tab is abandoned
			if rightCommand != "" || console.shellMode || console.sshMode {
				continue
			}
			// Tell helper the scenario
			task := &HelperTask{
				IsNodeMode: console.nodeMode,
				Uncomplete: leftCommand,
			}
			helper.TaskChan <- task
			compelete := <-helper.ResultChan
			// if only one match,then just print it
			if len(compelete) == 1 {
				fmt.Print("\r\033[K")
				fmt.Print(console.status)
				fmt.Print(compelete[0])
				leftCommand = compelete[0]
			} else if len(compelete) > 1 {
				// if multiple matches,then mimic linux's style
				fmt.Print("\r\n")
				for _, command := range compelete {
					fmt.Print(command + "    ")
				}
				fmt.Print("\r\n")
				fmt.Print(console.status)
				fmt.Print(leftCommand)
			}
		} else if event.Key == keyboard.KeyCtrlC {
			// Ctrl+C? Then BYE!
			fmt.Print("\r\n[*]BYE!")
			keyboard.Close()
			os.Exit(0)
		} else {
			fmt.Print("\n[*]Unsupported input! Press <ctrl+c> to exit,<enter> to continue")
		}
	}
}

func (console *Console) handleMainPanelCommand() {
	for {
		tCommand := console.pretreatInput()
		fCommand := strings.Split(tCommand, " ")
		switch fCommand[0] {
		case "use":
			if console.expectParams(fCommand, 2, MAIN, 1) {
				break
			}

			uuidNum, _ := utils.Str2Int(fCommand[1])
			task := &topology.TopoTask{
				Mode:    topology.CHECKNODE,
				UUIDNum: uuidNum,
			}
			console.topology.TaskChan <- task

			result := <-console.topology.ResultChan
			if result.IsExist {
				console.nodeMode = true
				console.status = fmt.Sprintf("(node %s) >> ", fCommand[1])
				console.handleNodePanelCommand(uuidNum)
				console.status = "(admin) >> "
				console.nodeMode = false
			} else {
				fmt.Printf("\n[*]Node %s doesn't exist!", fCommand[1])
			}

			console.ready <- true
		case "detail":
			if console.expectParams(fCommand, 1, MAIN, 0) {
				break
			}

			task := &topology.TopoTask{
				Mode: topology.SHOWDETAIL,
			}

			console.topology.TaskChan <- task
			<-console.topology.ResultChan

			console.ready <- true
		case "tree":
			if console.expectParams(fCommand, 1, MAIN, 0) {
				break
			}

			task := &topology.TopoTask{
				Mode: topology.SHOWTREE,
			}
			console.topology.TaskChan <- task
			<-console.topology.ResultChan

			console.ready <- true
		case "":
			if console.expectParams(fCommand, 1, MAIN, 0) {
				break
			}
			console.ready <- true
		case "help":
			if console.expectParams(fCommand, 1, MAIN, 0) {
				break
			}

			ShowMainHelp()

			console.ready <- true
		case "exit":
			if console.expectParams(fCommand, 1, MAIN, 0) {
				break
			}
			fmt.Print("\n[*]BYE!")
			os.Exit(0)
		default:
			fmt.Print("\n[*]Unknown Command!\n")
			ShowMainHelp()
			console.ready <- true
		}
	}
}

func (console *Console) handleNodePanelCommand(uuidNum int) {
	topoTask := &topology.TopoTask{
		Mode:    topology.GETUUID,
		UUIDNum: uuidNum,
	}
	console.topology.TaskChan <- topoTask
	topoResult := <-console.topology.ResultChan
	uuid := topoResult.UUID

	topoTask = &topology.TopoTask{
		Mode: topology.GETROUTE,
		UUID: uuid,
	}
	console.topology.TaskChan <- topoTask
	topoResult = <-console.topology.ResultChan
	route := topoResult.Route

	component := &protocol.MessageComponent{
		Secret: console.secret,
		Conn:   console.conn,
		UUID:   protocol.ADMIN_UUID,
	}

	console.ready <- true

	for {
		tCommand := console.pretreatInput()
		fCommand := strings.Split(tCommand, " ")

		switch fCommand[0] {
		case "addmemo":
			handler.AddMemo(component, console.topology.TaskChan, fCommand[1:], uuid, route)
			console.ready <- true
		case "delmemo":
			if console.expectParams(fCommand, 1, NODE, 0) {
				break
			}

			handler.DelMemo(component, console.topology.TaskChan, uuid, route)
			console.ready <- true
		case "shell":
			if console.expectParams(fCommand, 1, NODE, 0) {
				break
			}

			handler.LetShellStart(component, route, uuid)

			fmt.Print("\r\n[*]Waiting for response.....")
			fmt.Print("\r\n[*]MENTION!UNDER SHELL MODE ARROW UP/DOWN/LEFT/RIGHT ARE ALL ABANDONED!")

			if <-console.mgr.ConsoleManager.OK {
				fmt.Print("\r\n[*]Shell is started successfully!\r\n")
				console.status = ""
				console.shellMode = true
				console.handleShellPanelCommand(component, route, uuid)
				console.shellMode = false
				console.status = fmt.Sprintf("(node %s) >> ", utils.Int2Str(uuidNum))
			} else {
				fmt.Print("\r\n[*]Shell cannot be started!")
				console.ready <- true
			}
		case "listen":
			if console.expectParams(fCommand, 2, NODE, 0) {
				break
			}

			handler.LetListen(component, route, uuid, fCommand[1])
			console.ready <- true
		case "ssh":
			if console.expectParams(fCommand, 2, NODE, 0) {
				break
			}

			ssh := handler.NewSSH(fCommand[1])

			console.status = "[*]Please choose the auth method(1.username/password 2.certificate): "
			console.ready <- true

			firstChoice := console.pretreatInput()
			if firstChoice == "1" {
				ssh.Method = handler.UPMETHOD
			} else if firstChoice == "2" {
				ssh.Method = handler.CERMETHOD
			} else {
				fmt.Print("\r\n[*]Please input 1 or 2!")
				console.status = fmt.Sprintf("(node %s) >> ", utils.Int2Str(uuidNum))
				console.ready <- true
				break
			}

			switch ssh.Method {
			case handler.UPMETHOD:
				console.status = "[*]Please enter the username: "
				console.ready <- true
				ssh.Username = console.pretreatInput()
				console.status = "[*]Please enter the password: "
				console.ready <- true
				ssh.Password = console.pretreatInput()
			case handler.CERMETHOD:
				console.status = "[*]Please enter the username: "
				console.ready <- true
				ssh.Username = console.pretreatInput()
				console.status = "[*]Please enter the filepath of the privkey: "
				console.ready <- true
				ssh.CertificatePath = console.pretreatInput()
			}

			err := ssh.LetSSH(component, route, uuid)
			if err != nil {
				fmt.Printf("\r\n[*]Error: %s", err.Error())
				console.status = fmt.Sprintf("(node %s) >> ", utils.Int2Str(uuidNum))
				console.ready <- true
				break
			}

			fmt.Print("\r\n[*]Waiting for response.....")

			if <-console.mgr.ConsoleManager.OK {
				fmt.Print("\r\n[*]Connect to target host via ssh successfully!")
				console.status = ""
				console.sshMode = true
				console.handleSSHPanelCommand(component, route, uuid)
				console.status = fmt.Sprintf("(node %s) >> ", utils.Int2Str(uuidNum))
				console.sshMode = false
			} else {
				fmt.Print("\r\n[*]Fail to connect to target host via ssh!")
				console.status = fmt.Sprintf("(node %s) >> ", utils.Int2Str(uuidNum))
				console.ready <- true
			}
		case "socks":
			if console.expectParams(fCommand, []int{2, 4}, NODE, 0) {
				break
			}

			socks := handler.NewSocks(fCommand[1])
			if len(fCommand) > 2 {
				socks.Username = fCommand[2]
				socks.Password = fCommand[3]
			}

			fmt.Printf("\r\n[*]Trying to listen on 0.0.0.0:%s......", fCommand[1])
			fmt.Printf("\r\n[*]Waiting for agent's response......")

			err := socks.LetSocks(component, console.mgr, route, uuid)

			if err != nil {
				fmt.Printf("\r\n[*]Error: %s", err.Error())
			} else {
				fmt.Print("\r\n[*]Socks start successfully!")
			}
			console.ready <- true
		case "stopsocks":
			if console.expectParams(fCommand, 1, NODE, 0) {
				break
			}

			IsRunning := handler.GetSocksInfo(console.mgr, uuid)

			if IsRunning {
				console.status = "[*]Do you really want to shutdown socks?(yes/no): "
				console.ready <- true
				option := console.pretreatInput()
				if option == "yes" {
					fmt.Printf("\r\n[*]Closing......")
					handler.StopSocks(console.mgr, uuid)
					fmt.Printf("\r\n[*]Socks service has been closed successfully!")
				} else if option == "no" {
				} else {
					fmt.Printf("\r\n[*]Please input yes/no!")
				}
				console.status = fmt.Sprintf("(node %s) >> ", utils.Int2Str(uuidNum))
			}
			console.ready <- true
		case "forward":
			if console.expectParams(fCommand, 3, NODE, 1) {
				break
			}

			fmt.Printf("\r\n[*]Trying to listen on 0.0.0.0:%s......", fCommand[1])
			fmt.Printf("\r\n[*]Waiting for agent's response......")

			forward := handler.NewForward(fCommand[1], fCommand[2])

			err := forward.LetForward(component, console.mgr, route, uuid)
			if err != nil {
				fmt.Printf("\r\n[*]Error: %s", err.Error())
			} else {
				fmt.Print("\r\n[*]Forward start successfully!")
			}
			console.ready <- true
		case "stopforward":
			if console.expectParams(fCommand, 1, NODE, 0) {
				break
			}

			seq, isRunning := handler.GetForwardInfo(console.mgr, uuid)

			if isRunning {
				console.status = "[*]Do you really want to shutdown forward?(yes/no): "
				console.ready <- true
				option := console.pretreatInput()
				if option == "yes" {
					console.status = "[*]Please choose one to close: "
					console.ready <- true
					option := console.pretreatInput()
					target, err := utils.Str2Int(option)
					if err != nil {
						fmt.Printf("\r\n[*]Please input integer!")
					} else if target > seq || target < 0 {
						fmt.Printf("\r\n[*]Please input integer between 0~%d", seq)
					} else {
						fmt.Printf("\r\n[*]Closing......")
						handler.StopForward(console.mgr, uuid, target)
						fmt.Printf("\r\n[*]Forward service has been closed successfully!")
					}
				} else if option == "no" {
				} else {
					fmt.Printf("\r\n[*]Please input yes/no!")
				}
				console.status = fmt.Sprintf("(node %s) >> ", utils.Int2Str(uuidNum))
			}
			console.ready <- true
		case "backward":
			if console.expectParams(fCommand, 3, NODE, []int{1, 2}) {
				break
			}

			fmt.Printf("\r\n[*]Trying to ask node to listen on 0.0.0.0:%s......", fCommand[1])
			fmt.Printf("\r\n[*]Waiting for agent's response......")

			backward := handler.NewBackward(fCommand[2], fCommand[1])
			// node is okay
			err := backward.LetBackward(component, console.mgr, route, uuid)
			if err != nil {
				fmt.Printf("\r\n[*]Error: %s", err.Error())
			} else {
				fmt.Print("\r\n[*]Forward start successfully!")
			}
			console.ready <- true

		case "upload":
			if console.expectParams(fCommand, 3, NODE, 0) {
				break
			}

			console.mgr.FileManager.File.FilePath = fCommand[1]
			console.mgr.FileManager.File.FileName = fCommand[2]

			err := console.mgr.FileManager.File.SendFileStat(component, route, uuid, share.ADMIN)

			if err == nil && <-console.mgr.ConsoleManager.OK {
				go handler.StartBar(console.mgr.FileManager.File.StatusChan, console.mgr.FileManager.File.FileSize)
				console.mgr.FileManager.File.Upload(component, route, uuid, share.ADMIN)
			} else if err != nil {
				fmt.Printf("\r\n[*]Error: %s", err.Error())
			} else {
				fmt.Print("\r\n[*]Fail to upload file!")
			}
			console.ready <- true
		case "download":
			if console.expectParams(fCommand, 3, NODE, 0) {
				break
			}

			console.mgr.FileManager.File.FilePath = fCommand[1]
			console.mgr.FileManager.File.FileName = fCommand[2]

			console.mgr.FileManager.File.Ask4Download(component, route, uuid)

			if <-console.mgr.ConsoleManager.OK {
				err := console.mgr.FileManager.File.CheckFileStat(component, route, uuid, share.ADMIN)
				if err == nil {
					go handler.StartBar(console.mgr.FileManager.File.StatusChan, console.mgr.FileManager.File.FileSize)
					console.mgr.FileManager.File.Receive(component, route, uuid, share.ADMIN)
				}
			} else {
				fmt.Print("\r\n[*]Unable to download file!")
			}
			console.ready <- true
		case "offline":
			if console.expectParams(fCommand, 1, NODE, 0) {
				break
			}

			handler.LetOffline(component, route, uuid)
			console.ready <- true
		case "":
			if console.expectParams(fCommand, 1, NODE, 0) {
				break
			}
			console.ready <- true
		case "help":
			if console.expectParams(fCommand, 1, NODE, 0) {
				break
			}

			ShowNodeHelp()
			console.ready <- true
		case "exit":
			if console.expectParams(fCommand, 1, NODE, 0) {
				break
			}
			return
		default:
			fmt.Print("\n[*]Unknown Command!\n")
			ShowNodeHelp()
			console.ready <- true
		}
	}
}

func (console *Console) handleShellPanelCommand(component *protocol.MessageComponent, route string, uuid string) {
	sMessage := protocol.PrepareAndDecideWhichSProtoToLower(component.Conn, component.Secret, component.UUID)

	header := &protocol.Header{
		Sender:      protocol.ADMIN_UUID,
		Accepter:    uuid,
		MessageType: protocol.SHELLCOMMAND,
		RouteLen:    uint32(len([]byte(route))),
		Route:       route,
	}

	console.ready <- true

	var done bool
	for {

		if done { // check if user has asked to exit
			return
		}

		tCommand := <-console.getCommand

		if tCommand == "exit" {
			done = true
		}

		fCommand := tCommand + "\n"

		shellCommandMess := &protocol.ShellCommand{
			CommandLen: uint64(len(fCommand)),
			Command:    fCommand,
		}

		protocol.ConstructMessage(sMessage, header, shellCommandMess)
		sMessage.SendMessage()
	}
}

func (console *Console) handleSSHPanelCommand(component *protocol.MessageComponent, route string, uuid string) {
	sMessage := protocol.PrepareAndDecideWhichSProtoToLower(component.Conn, component.Secret, component.UUID)

	header := &protocol.Header{
		Sender:      protocol.ADMIN_UUID,
		Accepter:    uuid,
		MessageType: protocol.SSHCOMMAND,
		RouteLen:    uint32(len([]byte(route))),
		Route:       route,
	}

	console.ready <- true

	var done bool
	for {
		if done { // check if user has asked to exit
			return
		}

		tCommand := <-console.getCommand

		if tCommand == "exit" {
			done = true
		}

		fCommand := tCommand + "\n"

		sshCommandMess := &protocol.SSHCommand{
			CommandLen: uint64(len(fCommand)),
			Command:    fCommand,
		}

		protocol.ConstructMessage(sMessage, header, sshCommandMess)
		sMessage.SendMessage()
	}
}

func (console *Console) expectParams(params []string, numbers interface{}, mode int, needToBeInt interface{}) bool {
	switch numbers.(type) {
	case int:
		num := numbers.(int)
		if len(params) != num {
			fmt.Print("\n[*]Format error!\n")
			if mode == MAIN {
				ShowMainHelp()
			} else {
				ShowNodeHelp()
			}
			console.ready <- true
			return true
		}
	case []int:
		nums := numbers.([]int)
		var flag bool
		for _, num := range nums {
			if len(params) == num {
				flag = true
			}
		}

		if !flag {
			fmt.Print("\n[*]Format error!\n")
			if mode == MAIN {
				ShowMainHelp()
			} else {
				ShowNodeHelp()
			}
			console.ready <- true
			return true
		}
	}

	switch needToBeInt.(type) {
	case int:
		seq := needToBeInt.(int)
		if needToBeInt != 0 {
			_, err := utils.Str2Int(params[seq])
			if err != nil {
				fmt.Print("\n[*]Format error!\n")
				if mode == MAIN {
					ShowMainHelp()
				} else {
					ShowNodeHelp()
				}
				console.ready <- true
				return true
			}
		}
	case []int:
		seqs := needToBeInt.([]int)
		var err error
		for _, seq := range seqs {
			if seq != 0 {
				_, err = utils.Str2Int(params[seq])
				if err != nil {
					break
				}
			}
		}

		if err != nil {
			fmt.Print("\n[*]Format error!\n")
			if mode == MAIN {
				ShowMainHelp()
			} else {
				ShowNodeHelp()
			}
			console.ready <- true
			return true
		}
	}

	return false
}

func (console *Console) pretreatInput() string {
	tCommand := <-console.getCommand
	tCommand = strings.TrimRight(tCommand, " \t\r\n")
	return tCommand
}
