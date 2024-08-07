//go:build windows

package cli

import (
	"bytes"
	"fmt"
	"os"
	"os/signal"
	"strings"

	"Stowaway/admin/handler"
	"Stowaway/admin/manager"
	"Stowaway/admin/printer"
	"Stowaway/admin/topology"
	"Stowaway/global"
	"Stowaway/protocol"
	"Stowaway/share"
	"Stowaway/utils"

	"github.com/eiannone/keyboard"
)

const (
	MAIN = iota
	NODE
)

type Console struct {
	// console internal elements
	status     string
	ready      chan bool
	getCommand chan string
	shellMode  bool
	sshMode    bool
	nodeMode   bool
	// shared
	topology *topology.Topology
	mgr      *manager.Manager
}

func NewConsole() *Console {
	console := new(Console)
	console.status = "(admin) >> "
	console.ready = make(chan bool)
	console.getCommand = make(chan string)
	return console
}

func (console *Console) Init(tTopology *topology.Topology, myManager *manager.Manager) {
	console.topology = tTopology
	console.mgr = myManager
}

func (console *Console) Run() {
	go console.handleMainPanelCommand()
	console.mainPanel()
}

func (console *Console) mainPanel() {
	var (
		isGoingOn    bool
		leftCommand  string
		rightCommand string
	)
	// start history
	history := NewHistory()
	go history.Run()
	// start helper
	helper := NewHelper()
	go helper.Run()
	// monitor CTRLC
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	keysEvents, _ := keyboard.GetKeys(10)

	fmt.Print(console.status)
	// Tested on:
	// Macos Catalina iterm2/original terminal
	// Ubuntu desktop 16.04/18.04
	// Ubuntu server 16.04
	// Centos 7
	// Win10 x64 Professional
	// May have problems when the console working on some terminal since I'm using escape sequence.
	for {
		var event keyboard.KeyEvent

		select {
		case event = <-keysEvents:
		case <-c:
			event.Key = keyboard.KeyCtrlC
		}

		if event.Err != nil {
			continue
		}

		// under shell&&ssh mode,we cannot just erase the whole line and reprint,so there are two different ways to handle input
		// BTW,all arrow stuff under shell&&ssh mode will be abandoned
		if event.Key == keyboard.KeyBackspace2 || event.Key == keyboard.KeyBackspace {
			if !console.shellMode && !console.sshMode {
				fmt.Print("\r\033[K")
				fmt.Print(console.status)

				if len(leftCommand) >= 1 {
					leftCommand = string([]rune(leftCommand)[:len([]rune(leftCommand))-1])
				}

				fmt.Print(leftCommand + rightCommand)

				notSingleNum := (len(rightCommand) - len([]rune(rightCommand))) / 2 // count non-english characters‘ num
				singleNum := len([]rune(rightCommand)) - notSingleNum               // count English characters
				// every non-english character need two '\b'(Actually,i don't know why,i just tested a lot and find this stupid solution(on Mac,linux). So if u know,plz tell me,thx :) )
				fmt.Print(string(bytes.Repeat([]byte("\b"), notSingleNum*2+singleNum)))
			} else {
				notSingleNum := (len(leftCommand) - len([]rune(leftCommand))) / 2
				singleNum := len([]rune(leftCommand)) - notSingleNum
				fmt.Print(string(bytes.Repeat([]byte("\b"), notSingleNum*2+singleNum)))
				fmt.Print("\033[K")

				if len(leftCommand) >= 1 {
					leftCommand = string([]rune(leftCommand)[:len([]rune(leftCommand))-1])
				}

				fmt.Print(leftCommand + rightCommand)

				notSingleNum = (len(rightCommand) - len([]rune(rightCommand))) / 2
				singleNum = len([]rune(rightCommand)) - notSingleNum
				fmt.Print(string(bytes.Repeat([]byte("\b"), notSingleNum*2+singleNum)))
			}
		} else if event.Key == keyboard.KeyEnter {
			if !console.shellMode && !console.sshMode {
				command := leftCommand + rightCommand
				// if command is not "",send it to history
				if command != "" {
					task := &HistoryTask{
						Mode:    RECORD,
						Type:    NORMAL,
						Command: command,
					}
					history.TaskChan <- task
				}
				// no matter what command is,send it to console to parse
				console.getCommand <- command
				// set searching->false
				isGoingOn = false
				// set both left/right command -> []rune{},new start!
				leftCommand = ""
				rightCommand = ""
				// avoid scenario that console.status is printed before it's changed
				<-console.ready
				fmt.Print("\r\n")
				fmt.Print(console.status)
			} else {
				fmt.Print("\r\n")

				command := leftCommand + rightCommand
				console.getCommand <- command + "\n"

				if leftCommand != "" {
					var task = &HistoryTask{
						Mode:    RECORD,
						Command: command,
					}

					if console.shellMode {
						task.Type = SHELL
					} else {
						task.Type = SSH
					}

					history.TaskChan <- task
				}

				isGoingOn = false
				leftCommand = ""
				rightCommand = ""
			}
		} else if event.Key == keyboard.KeyArrowUp {
			if !console.shellMode && !console.sshMode {
				fmt.Print("\r\033[K")
				fmt.Print(console.status)
				// new task
				task := &HistoryTask{
					Mode:  SEARCH,
					Type:  NORMAL,
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
				// get the history command
				result := <-history.ResultChan
				fmt.Print(result)
				// set rightcommand -> []rune{}
				leftCommand = result
				rightCommand = ""
			} else {
				task := &HistoryTask{
					Mode:  SEARCH,
					Order: BEGIN,
				}

				if console.shellMode {
					task.Type = SHELL
				} else {
					task.Type = SSH
				}

				if !isGoingOn {
					history.TaskChan <- task
					isGoingOn = true
				} else {
					task.Order = NEXT
					history.TaskChan <- task
				}

				command := <-history.ResultChan

				notSingleNum := (len(leftCommand) - len([]rune(leftCommand))) / 2
				singleNum := len([]rune(leftCommand)) - notSingleNum
				fmt.Print(string(bytes.Repeat([]byte("\b"), notSingleNum*2+singleNum)))
				fmt.Print("\033[K")
				fmt.Print(command)

				leftCommand = command
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
						Type:  NORMAL,
						Order: PREV,
					}
					history.TaskChan <- task
					result := <-history.ResultChan

					fmt.Print(result)
					leftCommand = result
				} else {
					// not started,then just erase user's input and output nothing
					leftCommand = ""
				}
				rightCommand = ""
			} else {
				var command string

				task := &HistoryTask{
					Mode:  SEARCH,
					Order: PREV,
				}

				if console.shellMode {
					task.Type = SHELL
				} else {
					task.Type = SSH
				}

				if isGoingOn {
					history.TaskChan <- task
					command = <-history.ResultChan
				} else {
					command = ""
				}

				notSingleNum := (len(leftCommand) - len([]rune(leftCommand))) / 2
				singleNum := len([]rune(leftCommand)) - notSingleNum
				fmt.Print(string(bytes.Repeat([]byte("\b"), notSingleNum*2+singleNum)))
				fmt.Print("\033[K")
				fmt.Print(command)

				leftCommand = command
				rightCommand = ""
			}
		} else if event.Key == keyboard.KeyArrowLeft {
			if !console.shellMode && !console.sshMode {
				fmt.Print("\r\033[K")
				fmt.Print(console.status)
				// concat left command's last character with right command
				if len([]rune(leftCommand)) >= 1 {
					rightCommand = string([]rune(leftCommand)[len([]rune(leftCommand))-1]) + rightCommand
					leftCommand = string([]rune(leftCommand)[:len([]rune(leftCommand))-1])
				}
				// print command
				fmt.Print(leftCommand + rightCommand)
				// print \b
				notSingleNum := (len(rightCommand) - len([]rune(rightCommand))) / 2 // count non-english characters‘ num
				singleNum := len([]rune(rightCommand)) - notSingleNum
				fmt.Print(string(bytes.Repeat([]byte("\b"), notSingleNum*2+singleNum)))
			} else {
				notSingleNum := (len(leftCommand) - len([]rune(leftCommand))) / 2
				singleNum := len([]rune(leftCommand)) - notSingleNum
				fmt.Print(string(bytes.Repeat([]byte("\b"), notSingleNum*2+singleNum)))
				fmt.Print("\033[K")

				if len([]rune(leftCommand)) >= 1 {
					rightCommand = string([]rune(leftCommand)[len([]rune(leftCommand))-1]) + rightCommand
					leftCommand = string([]rune(leftCommand)[:len([]rune(leftCommand))-1])
				}

				fmt.Print(leftCommand + rightCommand)

				notSingleNum = (len(rightCommand) - len([]rune(rightCommand))) / 2
				singleNum = len([]rune(rightCommand)) - notSingleNum
				fmt.Print(string(bytes.Repeat([]byte("\b"), notSingleNum*2+singleNum)))
			}
		} else if event.Key == keyboard.KeyArrowRight {
			if !console.shellMode && !console.sshMode {
				fmt.Print("\r\033[K")
				fmt.Print(console.status)
				// concat right command's first character with left command
				if len([]rune(rightCommand)) > 1 {
					leftCommand = leftCommand + string([]rune(rightCommand)[:1])
					rightCommand = string([]rune(rightCommand)[1:])
				} else if len([]rune(rightCommand)) == 1 {
					leftCommand = leftCommand + string([]rune(rightCommand)[:1])
					rightCommand = ""
				}

				fmt.Print(leftCommand + rightCommand)

				notSingleNum := (len(rightCommand) - len([]rune(rightCommand))) / 2
				singleNum := len([]rune(rightCommand)) - notSingleNum
				fmt.Print(string(bytes.Repeat([]byte("\b"), notSingleNum*2+singleNum)))
			} else {
				notSingleNum := (len(leftCommand) - len([]rune(leftCommand))) / 2
				singleNum := len([]rune(leftCommand)) - notSingleNum
				fmt.Print(string(bytes.Repeat([]byte("\b"), notSingleNum*2+singleNum)))
				fmt.Print("\033[K")

				if len([]rune(rightCommand)) > 1 {
					leftCommand = leftCommand + string([]rune(rightCommand)[:1])
					rightCommand = string([]rune(rightCommand)[1:])
				} else if len([]rune(rightCommand)) == 1 {
					leftCommand = leftCommand + string([]rune(rightCommand)[:1])
					rightCommand = ""
				}

				fmt.Print(leftCommand + rightCommand)

				notSingleNum = (len(rightCommand) - len([]rune(rightCommand))) / 2
				singleNum = len([]rune(rightCommand)) - notSingleNum
				fmt.Print(string(bytes.Repeat([]byte("\b"), notSingleNum*2+singleNum)))
			}
		} else if event.Key == keyboard.KeyTab {
			if len(rightCommand) != 0 || console.shellMode || console.sshMode {
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
			// Ctrl+C?
			if !console.shellMode && !console.sshMode {
				printer.Warning("\r\n[*] Please use 'exit' to exit stowaway or use 'back' to return to parent panel")
			} else {
				printer.Warning("\r\n[*] Press 'Enter' to force quit shell/ssh mode, other keys to continue")
				event := <-keysEvents
				if event.Key == keyboard.KeyEnter {
					console.mgr.ConsoleManager.Exit <- true
					printer.Success("\r\n[*] Quit shell/ssh mode successfully, press 'Enter' to continue")
				} else {
					printer.Warning("\r\n[*] Continue shell/ssh mode, press 'Enter' to continue")
				}
			}
		} else {
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

				notSingleNum := (len(rightCommand) - len([]rune(rightCommand))) / 2
				singleNum := len([]rune(rightCommand)) - notSingleNum
				fmt.Print(string(bytes.Repeat([]byte("\b"), notSingleNum*2+singleNum)))
			} else {
				notSingleNum := (len(leftCommand) - len([]rune(leftCommand))) / 2
				singleNum := len([]rune(leftCommand)) - notSingleNum
				fmt.Print(string(bytes.Repeat([]byte("\b"), notSingleNum*2+singleNum)))
				fmt.Print("\033[K")

				if event.Key == keyboard.KeySpace {
					leftCommand = leftCommand + " "
				} else {
					leftCommand = leftCommand + string(event.Rune)
				}

				fmt.Print(leftCommand + rightCommand)

				notSingleNum = (len(rightCommand) - len([]rune(rightCommand))) / 2
				singleNum = len([]rune(rightCommand)) - notSingleNum
				fmt.Print(string(bytes.Repeat([]byte("\b"), notSingleNum*2+singleNum)))
			}
		}
	}
}

// handle ur command
func (console *Console) handleMainPanelCommand() {
	for {
		tCommand := console.pretreatInput()

		var fCommand []string
		for _, command := range strings.Split(tCommand, " ") {
			if command != "" {
				fCommand = append(fCommand, command)
			}
		}

		if len(fCommand) == 0 {
			fCommand = append(fCommand, "")
		}

		switch fCommand[0] {
		case "use":
			if console.expectParams(fCommand, 2, MAIN, 1) {
				break
			}

			uuidNum, _ := utils.Str2Int(fCommand[1])

			if console.isOnline(uuidNum) {
				console.nodeMode = true
				console.status = fmt.Sprintf("(node %s) >> ", fCommand[1])
				console.handleNodePanelCommand(uuidNum)
				console.status = "(admin) >> "
				console.nodeMode = false
			} else {
				printer.Fail("\r\n[*] Node %s doesn't exist!", fCommand[1])
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
		case "topo":
			if console.expectParams(fCommand, 1, MAIN, 0) {
				break
			}

			task := &topology.TopoTask{
				Mode: topology.SHOWTOPO,
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

			console.status = "[*] Do you really want to exit stowaway?(y/n): "
			console.ready <- true
			option := console.pretreatInput()

			if option == "y" {
				keyboard.Close()
				printer.Warning("\r\n[*] BYE!")
				os.Exit(0)
			}

			console.status = "(admin) >> "
			console.ready <- true
		default:
			printer.Fail("\r\n[*] Unknown Command!\r\n")
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

	console.ready <- true

	for {
		tCommand := console.pretreatInput()
		fCommand := strings.Split(tCommand, " ")

		switch fCommand[0] {
		case "addmemo":
			if !console.isOnline(uuidNum) {
				return
			}

			handler.AddMemo(console.topology.TaskChan, fCommand[1:], uuid, route)
			console.ready <- true
		case "delmemo":
			if !console.isOnline(uuidNum) {
				return
			}

			if console.expectParams(fCommand, 1, NODE, 0) {
				break
			}

			handler.DelMemo(console.topology.TaskChan, uuid, route)
			console.ready <- true
		case "shell":
			if !console.isOnline(uuidNum) {
				return
			}

			if console.expectParams(fCommand, 1, NODE, 0) {
				break
			}

			printer.Warning("\r\n[*] Waiting for response.....")

			handler.LetShellStart(route, uuid)

			if <-console.mgr.ConsoleManager.OK {
				console.status = ""
				console.shellMode = true
				console.handleShellPanelCommand(route, uuid)
				console.shellMode = false
				console.status = fmt.Sprintf("(node %d) >> ", uuidNum)
			} else {
				printer.Fail("\r\n[*] Shell cannot be started!")
				console.ready <- true
			}
		case "listen":
			if !console.isOnline(uuidNum) {
				return
			}

			if console.expectParams(fCommand, 1, NODE, 0) {
				break
			}

			listen := handler.NewListen()

			printer.Warning("\r\n[*] BE AWARE! If you choose IPTables Reuse or SOReuse,you MUST CONFIRM that the node you're controlling was started in the corresponding way!")
			printer.Warning("\r\n[*] When you choose IPTables Reuse or SOReuse, the node will use the initial config(when node started) to reuse port!")
			console.status = "[*] Please choose the mode(1.Normal passive/2.IPTables Reuse/3.SOReuse): "
			console.ready <- true

			option := console.pretreatInput()
			if option == "1" {
				listen.Method = handler.NORMAL
				console.status = "[*] Please input the [ip:]<port> : "
				console.ready <- true
				option = console.pretreatInput()
				listen.Addr = option
			} else if option == "2" {
				listen.Method = handler.IPTABLES
			} else if option == "3" {
				listen.Method = handler.SOREUSE
			} else {
				printer.Fail("\r\n[*] Please input 1/2/3!")
				console.status = fmt.Sprintf("(node %d) >> ", uuidNum)
				console.ready <- true
				continue
			}

			printer.Warning("\r\n[*] Waiting for response......")

			err := listen.LetListen(console.mgr, route, uuid)
			if err != nil {
				printer.Fail("[*] Error: %s\n", err.Error())
			}

			console.status = fmt.Sprintf("(node %d) >> ", uuidNum)

			console.ready <- true
		case "connect":
			if !console.isOnline(uuidNum) {
				return
			}

			if console.expectParams(fCommand, 2, NODE, 0) {
				break
			}

			printer.Warning("\r\n[*] Waiting for response......")

			err := handler.LetConnect(console.mgr, route, uuid, fCommand[1])
			if err != nil {
				printer.Fail("[*] Error: %s\n", err.Error())
			}

			console.status = fmt.Sprintf("(node %d) >> ", uuidNum)

			console.ready <- true
		case "ssh":
			if !console.isOnline(uuidNum) {
				return
			}

			if console.expectParams(fCommand, 2, NODE, 0) {
				break
			}

			ssh := handler.NewSSH(fCommand[1])

			console.status = "[*] Please choose the auth method(1.username/password 2.certificate): "
			console.ready <- true

			firstChoice := console.pretreatInput()
			if firstChoice == "1" {
				ssh.Method = handler.UPMETHOD
			} else if firstChoice == "2" {
				ssh.Method = handler.CERMETHOD
			} else {
				printer.Fail("\r\n[*] Please input 1 or 2!")
				console.status = fmt.Sprintf("(node %d) >> ", uuidNum)
				console.ready <- true
				break
			}

			switch ssh.Method {
			case handler.UPMETHOD:
				console.status = "[*] Please enter the username: "
				console.ready <- true
				ssh.Username = console.pretreatInput()
				console.status = "[*] Please enter the password: "
				console.ready <- true
				ssh.Password = console.pretreatInput()
			case handler.CERMETHOD:
				console.status = "[*] Please enter the username: "
				console.ready <- true
				ssh.Username = console.pretreatInput()
				console.status = "[*] Please enter the filepath of the privkey: "
				console.ready <- true
				ssh.CertificatePath = console.pretreatInput()
			}

			printer.Warning("\r\n[*] Waiting for response.....")

			err := ssh.LetSSH(route, uuid)
			if err != nil {
				printer.Fail("\r\n[*] Error: %s", err.Error())
				console.status = fmt.Sprintf("(node %d) >> ", uuidNum)
				console.ready <- true
				break
			}

			if <-console.mgr.ConsoleManager.OK {
				console.status = ""
				console.sshMode = true
				console.handleSSHPanelCommand(route, uuid)
				console.status = fmt.Sprintf("(node %d) >> ", uuidNum)
				console.sshMode = false
			} else {
				printer.Fail("\r\n[*] Fail to connect to target host via ssh!")
				console.status = fmt.Sprintf("(node %d) >> ", uuidNum)
				console.ready <- true
			}
		case "sshtunnel":
			if !console.isOnline(uuidNum) {
				return
			}

			if console.expectParams(fCommand, 3, NODE, 2) {
				break
			}

			sshTunnel := handler.NewSSHTunnel(fCommand[2], fCommand[1])

			console.status = "[*] Please choose the auth method(1.username/password 2.certificate): "
			console.ready <- true

			firstChoice := console.pretreatInput()
			if firstChoice == "1" {
				sshTunnel.Method = handler.UPMETHOD
			} else if firstChoice == "2" {
				sshTunnel.Method = handler.CERMETHOD
			} else {
				printer.Fail("\r\n[*] Please input 1 or 2!")
				console.status = fmt.Sprintf("(node %d) >> ", uuidNum)
				console.ready <- true
				break
			}

			switch sshTunnel.Method {
			case handler.UPMETHOD:
				console.status = "[*] Please enter the username: "
				console.ready <- true
				sshTunnel.Username = console.pretreatInput()
				console.status = "[*] Please enter the password: "
				console.ready <- true
				sshTunnel.Password = console.pretreatInput()
			case handler.CERMETHOD:
				console.status = "[*] Please enter the username: "
				console.ready <- true
				sshTunnel.Username = console.pretreatInput()
				console.status = "[*] Please enter the filepath of the privkey: "
				console.ready <- true
				sshTunnel.CertificatePath = console.pretreatInput()
			}

			printer.Warning("\r\n[*] Waiting for response.....")

			err := sshTunnel.LetSSHTunnel(route, uuid)
			if err != nil {
				printer.Fail("\r\n[*] Error: %s", err.Error())
				console.status = fmt.Sprintf("(node %d) >> ", uuidNum)
				console.ready <- true
				break
			}

			if ok := <-console.mgr.ConsoleManager.OK; !ok {
				printer.Fail("\r\n[*] Fail to add target node via SSHTunnel!")
			}

			console.status = fmt.Sprintf("(node %d) >> ", uuidNum)
			console.ready <- true
		case "socks":
			if !console.isOnline(uuidNum) {
				return
			}

			if console.expectParams(fCommand, []int{2, 4}, NODE, 0) {
				break
			}

			socks := handler.NewSocks(fCommand[1])
			if len(fCommand) > 2 {
				socks.Username = fCommand[2]
				socks.Password = fCommand[3]
			}

			printer.Warning("\r\n[*] Trying to listen on 0.0.0.0:%s......", fCommand[1])
			printer.Warning("\r\n[*] Waiting for agent's response......")

			err := socks.LetSocks(console.mgr, route, uuid)

			if err != nil {
				printer.Fail("\r\n[*] Error: %s", err.Error())
			} else {
				printer.Success("\r\n[*] Socks start successfully!")
			}
			console.ready <- true
		case "stopsocks":
			if !console.isOnline(uuidNum) {
				return
			}

			if console.expectParams(fCommand, 1, NODE, 0) {
				break
			}

			IsRunning := handler.GetSocksInfo(console.mgr, uuid)

			if IsRunning {
				console.status = "[*] Do you really want to shutdown socks?(y/n): "
				console.ready <- true
				option := console.pretreatInput()
				if option == "y" {
					printer.Warning("\r\n[*] Closing......")
					handler.StopSocks(console.mgr, uuid)
					printer.Success("\r\n[*] Socks service has been closed successfully!")
				} else if option == "n" {
				} else {
					printer.Fail("\r\n[*] Please input y/n!")
				}
				console.status = fmt.Sprintf("(node %d) >> ", uuidNum)
			}
			console.ready <- true
		case "forward":
			if !console.isOnline(uuidNum) {
				return
			}

			if console.expectParams(fCommand, 3, NODE, 1) {
				break
			}

			printer.Warning("\r\n[*] Trying to listen on 0.0.0.0:%s......", fCommand[1])
			printer.Warning("\r\n[*] Waiting for agent's response......")

			forward := handler.NewForward(fCommand[1], fCommand[2])

			err := forward.LetForward(console.mgr, route, uuid)
			if err != nil {
				printer.Fail("\r\n[*] Error: %s", err.Error())
			} else {
				printer.Success("\r\n[*] Forward start successfully!")
			}
			console.ready <- true
		case "stopforward":
			if !console.isOnline(uuidNum) {
				return
			}

			if console.expectParams(fCommand, 1, NODE, 0) {
				break
			}

			seq, isRunning := handler.GetForwardInfo(console.mgr, uuid)

			if isRunning {
				console.status = "[*] Do you really want to shutdown forward?(y/n): "
				console.ready <- true
				option := console.pretreatInput()
				if option == "y" {
					console.status = "[*] Please choose one to close: "
					console.ready <- true
					option := console.pretreatInput()
					choice, err := utils.Str2Int(option)
					if err != nil {
						printer.Fail("\r\n[*] Please input integer!")
					} else if choice > seq || choice < 0 {
						printer.Fail("\r\n[*] Please input integer between 0~%d", seq)
					} else {
						printer.Warning("\r\n[*] Closing......")
						handler.StopForward(console.mgr, uuid, choice)
						printer.Success("\r\n[*] Forward service has been closed successfully!")
					}
				} else if option == "n" {
				} else {
					printer.Fail("\r\n[*] Please input y/n!")
				}
				console.status = fmt.Sprintf("(node %d) >> ", uuidNum)
			}
			console.ready <- true
		case "backward":
			if !console.isOnline(uuidNum) {
				return
			}

			if console.expectParams(fCommand, 3, NODE, []int{1, 2}) {
				break
			}

			printer.Warning("\r\n[*] Trying to ask node to listen on 0.0.0.0:%s......", fCommand[1])
			printer.Warning("\r\n[*] Waiting for agent's response......")

			backward := handler.NewBackward(fCommand[2], fCommand[1])
			// node is okay
			err := backward.LetBackward(console.mgr, route, uuid)
			if err != nil {
				printer.Fail("\r\n[*] Error: %s", err.Error())
			} else {
				printer.Success("\r\n[*] Backward start successfully!")
			}
			console.ready <- true
		case "stopbackward":
			if !console.isOnline(uuidNum) {
				return
			}

			if console.expectParams(fCommand, 1, NODE, 0) {
				break
			}

			seq, isRunning := handler.GetBackwardInfo(console.mgr, uuid)

			if isRunning {
				console.status = "[*] Do you really want to shutdown backward?(y/n): "
				console.ready <- true
				option := console.pretreatInput()
				if option == "y" {
					console.status = "[*] Please choose one to close: "
					console.ready <- true
					option := console.pretreatInput()
					choice, err := utils.Str2Int(option)
					if err != nil {
						printer.Fail("\r\n[*] Please input integer!")
					} else if choice > seq || choice < 0 {
						printer.Fail("\r\n[*] Please input integer between 0~%d", seq)
					} else {
						printer.Warning("\r\n[*] Closing......")
						handler.StopBackward(console.mgr, uuid, route, choice)
						printer.Success("\r\n[*] Backward service has been closed successfully!")
					}
				} else if option == "n" {
				} else {
					printer.Fail("\r\n[*] Please input y/n!")
				}
				console.status = fmt.Sprintf("(node %d) >> ", uuidNum)
			}
			console.ready <- true
		case "upload":
			if !console.isOnline(uuidNum) {
				return
			}

			var err error

			console.mgr.FileManager.File.FilePath, console.mgr.FileManager.File.FileName, err = utils.ParseFileCommand(fCommand[1:])
			if err != nil {
				printer.Fail("\r\n[*] Error: %s", err.Error())
				console.ready <- true
				break
			}

			err = console.mgr.FileManager.File.SendFileStat(route, uuid, share.ADMIN)

			if err == nil && <-console.mgr.ConsoleManager.OK {
				go handler.StartBar(console.mgr.FileManager.File.StatusChan, console.mgr.FileManager.File.FileSize)
				console.mgr.FileManager.File.Upload(route, uuid, share.ADMIN)
			} else if err != nil {
				printer.Fail("\r\n[*] Error: %s", err.Error())
			} else {
				printer.Fail("\r\n[*] Fail to upload file!")
			}
			console.ready <- true
		case "download":
			if !console.isOnline(uuidNum) {
				return
			}

			var err error

			console.mgr.FileManager.File.FilePath, console.mgr.FileManager.File.FileName, err = utils.ParseFileCommand(fCommand[1:])
			if err != nil {
				printer.Fail("\r\n[*] Error: %s", err.Error())
				console.ready <- true
				break
			}

			console.mgr.FileManager.File.Ask4Download(route, uuid)

			if <-console.mgr.ConsoleManager.OK {
				err := console.mgr.FileManager.File.CheckFileStat(route, uuid, share.ADMIN)
				if err == nil {
					go handler.StartBar(console.mgr.FileManager.File.StatusChan, console.mgr.FileManager.File.FileSize)
					console.mgr.FileManager.File.Receive(route, uuid, share.ADMIN)
				}
			} else {
				printer.Fail("\r\n[*] Unable to download file!")
			}
			console.ready <- true
		case "shutdown":
			if !console.isOnline(uuidNum) {
				return
			}

			if console.expectParams(fCommand, 1, NODE, 0) {
				break
			}

			handler.LetShutdown(route, uuid)
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
		case "back":
			if console.expectParams(fCommand, 1, NODE, 0) {
				break
			}
			return
		case "exit":
			if console.expectParams(fCommand, 1, NODE, 0) {
				break
			}

			console.status = "[*] Do you really want to exit stowaway?(y/n): "
			console.ready <- true
			option := console.pretreatInput()

			if option == "y" {
				keyboard.Close()
				printer.Warning("\r\n[*] BYE!")
				os.Exit(0)
			}

			console.status = fmt.Sprintf("(node %d) >> ", uuidNum)
			console.ready <- true
		default:
			printer.Fail("\r\n[*] Unknown Command!\r\n")
			ShowNodeHelp()
			console.ready <- true
		}
	}
}

func (console *Console) handleShellPanelCommand(route string, uuid string) {
	sMessage := protocol.NewDownMsg(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)

	header := &protocol.Header{
		Sender:      protocol.ADMIN_UUID,
		Accepter:    uuid,
		MessageType: protocol.SHELLCOMMAND,
		RouteLen:    uint32(len([]byte(route))),
		Route:       route,
	}

	console.ready <- true

	for {
		select {
		case tCommand := <-console.getCommand:
			shellCommandMess := &protocol.ShellCommand{
				CommandLen: uint64(len(tCommand)),
				Command:    tCommand,
			}
			protocol.ConstructMessage(sMessage, header, shellCommandMess, false)
			sMessage.SendMessage()
		case <-console.mgr.ConsoleManager.Exit:
			return
		}
	}
}

func (console *Console) handleSSHPanelCommand(route string, uuid string) {
	sMessage := protocol.NewDownMsg(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)

	header := &protocol.Header{
		Sender:      protocol.ADMIN_UUID,
		Accepter:    uuid,
		MessageType: protocol.SSHCOMMAND,
		RouteLen:    uint32(len([]byte(route))),
		Route:       route,
	}

	console.ready <- true

	for {
		select {
		case tCommand := <-console.getCommand:
			sshCommandMess := &protocol.SSHCommand{
				CommandLen: uint64(len(tCommand)),
				Command:    tCommand,
			}
			protocol.ConstructMessage(sMessage, header, sshCommandMess, false)
			sMessage.SendMessage()
		case <-console.mgr.ConsoleManager.Exit:
			return
		}
	}
}

func (console *Console) expectParams(params []string, numbers interface{}, mode int, needToBeInt interface{}) bool {
	switch nums := numbers.(type) {
	case int:
		if len(params) != nums {
			printer.Fail("\r\n[*] Format error!\r\n")
			if mode == MAIN {
				ShowMainHelp()
			} else {
				ShowNodeHelp()
			}
			console.ready <- true
			return true
		}
	case []int:
		var flag bool
		for _, num := range nums {
			if len(params) == num {
				flag = true
			}
		}

		if !flag {
			printer.Fail("\r\n[*] Format error!\r\n")
			if mode == MAIN {
				ShowMainHelp()
			} else {
				ShowNodeHelp()
			}
			console.ready <- true
			return true
		}
	}

	switch seqs := needToBeInt.(type) {
	case int:
		if needToBeInt != 0 {
			_, err := utils.Str2Int(params[seqs])
			if err != nil {
				printer.Fail("\r\n[*] Format error!\r\n")
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
			printer.Fail("\r\n[*] Format error!\r\n")
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

func (console *Console) isOnline(uuidNum int) bool {
	task := &topology.TopoTask{
		Mode:    topology.CHECKNODE,
		UUIDNum: uuidNum,
	}
	console.topology.TaskChan <- task

	result := <-console.topology.ResultChan
	if result.IsExist {
		return true
	}

	printer.Fail("\r\n[*] Node %d seems offline!", uuidNum)
	return false
}
