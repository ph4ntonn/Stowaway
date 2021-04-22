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
	"Stowaway/global"
	"Stowaway/protocol"
	"Stowaway/share"
	"Stowaway/utils"
	"bytes"
	"fmt"
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
	topology *topology.Topology
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

func (console *Console) Init(tTopology *topology.Topology, myManager *manager.Manager) {
	console.topology = tTopology
	console.mgr = myManager
}

func (console *Console) Run() {
	go console.handleMainPanelCommand()
	console.mainPanel()
}

// At first,i think "interactive console? That's too fxxking easy"
// But after i actually sit down and code this part,i changed my mind Orz
// iTerm2 yyds(FYI,yyds means sth is the best)
func (console *Console) mainPanel() {
	var (
		isGoingOn    bool
		leftCommand  []rune
		rightCommand []rune
	)
	// start history
	history := NewHistory()
	go history.Run()
	// start helper
	helper := NewHelper()
	go helper.Run()

	keysEvents, _ := keyboard.GetKeys(10)

	defer keyboard.Close()

	// Tested on:
	// Macos Catalina iterm/original terminal
	// Ubuntu desktop 16.04/18.04
	// Ubuntu server 16.04
	// Centos 7
	// May have problems when the console working on some terminal since I'm using escape sequence,so if ur checking code after face this situation,let me know if possible
	fmt.Print(console.status)
	for {
		event := <-keysEvents
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
					leftCommand = leftCommand[:len(leftCommand)-1]
				}

				fmt.Print(string(leftCommand) + string(rightCommand))

				notSingleNum := (len(string(rightCommand)) - len(rightCommand)) / 2 // count non-english characters‘ num
				singleNum := len(rightCommand) - notSingleNum                       // count English characters
				// every non-english character need two '\b'(Actually,i don't know why,i just tested a lot and find this stupid solution(on Mac,linux). So if u know,plz tell me,thx :) )
				fmt.Print(string(bytes.Repeat([]byte("\b"), notSingleNum*2+singleNum)))
			} else {
				if len(leftCommand) >= 1 {
					notSingleNum := (len(string(leftCommand)) - len(leftCommand)) / 2
					singleNum := len(leftCommand) - notSingleNum

					fmt.Print(string(bytes.Repeat([]byte("\b"), notSingleNum*2+singleNum)))
					fmt.Print("\033[K")

					leftCommand = leftCommand[:len(leftCommand)-1]

					fmt.Print(string(leftCommand))
				}
			}
		} else if event.Key == keyboard.KeyEnter {
			if !console.shellMode && !console.sshMode {
				command := string(leftCommand) + string(rightCommand)
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
				// set both left/right command -> []rune{},new start!
				leftCommand = []rune{}
				rightCommand = []rune{}
				// avoid scenario that console.status is printed before it's changed
				<-console.ready
				fmt.Print("\r\n")
				fmt.Print(console.status)
			} else {
				fmt.Print("\r\n")
				console.getCommand <- string(leftCommand)
				leftCommand = []rune{}
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
				// get the history command
				result := <-history.ResultChan
				fmt.Print(result)
				// set rightcommand -> []rune{}
				leftCommand = []rune(result)
				rightCommand = []rune{}
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
					result := <-history.ResultChan

					fmt.Print(result)
					leftCommand = []rune(result)
				} else {
					// not started,then just erase user's input and output nothing
					leftCommand = []rune{}
				}
				rightCommand = []rune{}
			}
		} else if event.Key == keyboard.KeyArrowLeft {
			if !console.shellMode && !console.sshMode {
				fmt.Print("\r\033[K")
				fmt.Print(console.status)
				// concat left command's last character with right command
				if len(leftCommand) >= 1 {
					rightCommand = []rune(string(leftCommand[len(leftCommand)-1:]) + string(rightCommand))
					leftCommand = leftCommand[:len(leftCommand)-1]
				}
				// print command
				fmt.Print(string(leftCommand) + string(rightCommand))
				// print \b
				notSingleNum := (len(string(rightCommand)) - len(rightCommand)) / 2 // count non-english characters‘ num
				singleNum := len(rightCommand) - notSingleNum
				fmt.Print(string(bytes.Repeat([]byte("\b"), notSingleNum*2+singleNum)))
			}
		} else if event.Key == keyboard.KeyArrowRight {
			if !console.shellMode && !console.sshMode {
				fmt.Print("\r\033[K")
				fmt.Print(console.status)
				// concat right command's first character with left command
				if len(rightCommand) > 1 {
					leftCommand = []rune(string(leftCommand) + string(rightCommand[:1]))
					rightCommand = rightCommand[1:]
				} else if len(rightCommand) == 1 {
					leftCommand = []rune(string(leftCommand) + string(rightCommand[:1]))
					rightCommand = []rune{}
				}

				fmt.Print(string(leftCommand) + string(rightCommand))

				notSingleNum := (len(string(rightCommand)) - len(rightCommand)) / 2
				singleNum := len(rightCommand) - notSingleNum
				fmt.Print(string(bytes.Repeat([]byte("\b"), notSingleNum*2+singleNum)))
			}
		} else if event.Key == keyboard.KeyTab {
			// if user move the cursor or under shellMode(sshMode),tab is abandoned
			if len(rightCommand) != 0 || console.shellMode || console.sshMode {
				continue
			}
			// Tell helper the scenario
			task := &HelperTask{
				IsNodeMode: console.nodeMode,
				Uncomplete: string(leftCommand),
			}
			helper.TaskChan <- task
			compelete := <-helper.ResultChan
			// if only one match,then just print it
			if len(compelete) == 1 {
				fmt.Print("\r\033[K")
				fmt.Print(console.status)
				fmt.Print(compelete[0])
				leftCommand = []rune(compelete[0])
			} else if len(compelete) > 1 {
				// if multiple matches,then mimic linux's style
				fmt.Print("\r\n")
				for _, command := range compelete {
					fmt.Print(command + "    ")
				}
				fmt.Print("\r\n")
				fmt.Print(console.status)
				fmt.Print(string(leftCommand))
			}
		} else if event.Key == keyboard.KeyCtrlC {
			// Ctrl+C? Then BYE!
			fmt.Print("\r\n[*]BYE!\r\n")
			break
		} else {
			if !console.shellMode && !console.sshMode {
				fmt.Print("\r\033[K")
				fmt.Print(console.status)
				// save every single input
				if event.Key == keyboard.KeySpace {
					leftCommand = []rune(string(leftCommand) + " ")
				} else {
					leftCommand = []rune(string(leftCommand) + string(event.Rune))
				}
				// print command && keep cursor at right position
				fmt.Print(string(leftCommand) + string(rightCommand))

				notSingleNum := (len(string(rightCommand)) - len(rightCommand)) / 2
				singleNum := len(rightCommand) - notSingleNum
				fmt.Print(string(bytes.Repeat([]byte("\b"), notSingleNum*2+singleNum)))
			} else {
				if event.Key == keyboard.KeySpace {
					leftCommand = []rune(string(leftCommand) + " ")
					fmt.Print(" ")
				} else {
					leftCommand = []rune(string(leftCommand) + string(event.Rune))
					fmt.Print(string(event.Rune))
				}
			}
		}
	}
}

// handle ur command
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

	console.ready <- true

	for {
		tCommand := console.pretreatInput()
		fCommand := strings.Split(tCommand, " ")

		switch fCommand[0] {
		case "addmemo":
			handler.AddMemo(console.topology.TaskChan, fCommand[1:], uuid, route)
			console.ready <- true
		case "delmemo":
			if console.expectParams(fCommand, 1, NODE, 0) {
				break
			}

			handler.DelMemo(console.topology.TaskChan, uuid, route)
			console.ready <- true
		case "shell":
			if console.expectParams(fCommand, 1, NODE, 0) {
				break
			}

			handler.LetShellStart(route, uuid)

			fmt.Print("\r\n[*]Waiting for response.....")
			fmt.Print("\r\n[*]MENTION!UNDER SHELL MODE ARROW UP/DOWN/LEFT/RIGHT ARE ALL ABANDONED!")

			if <-console.mgr.ConsoleManager.OK {
				fmt.Print("\r\n[*]Shell is started successfully!\r\n")
				console.status = ""
				console.shellMode = true
				console.handleShellPanelCommand(route, uuid)
				console.shellMode = false
				console.status = fmt.Sprintf("(node %s) >> ", utils.Int2Str(uuidNum))
			} else {
				fmt.Print("\r\n[*]Shell cannot be started!")
				console.ready <- true
			}
		case "listen":
			if console.expectParams(fCommand, 1, NODE, 0) {
				break
			}

			listen := handler.NewListen()

			fmt.Print("\r\n[*]BE AWARE! If you choose IPTables Reuse or SOReuse,you MUST confirm that the node you're controlling was started in the corresponding way!")
			fmt.Print("\r\n[*]When you choose IPTables Reuse or SOReuse, the node will use the initial config(when node started) to reuse port!")
			console.status = "[*]Please choose the mode(1.Normal passive/2.IPTables Reuse/3.SOReuse): "
			console.ready <- true

			option := console.pretreatInput()
			if option == "1" {
				listen.Method = handler.NORMAL
				console.status = "[*]Please input the [ip:]<port> : "
				console.ready <- true
				option = console.pretreatInput()
				listen.Addr = option
			} else if option == "2" {
				listen.Method = handler.IPTABLES
			} else if option == "3" {
				listen.Method = handler.SOREUSE
			} else {
				fmt.Printf("\r\n[*]Please input 1/2/3!")
				console.status = fmt.Sprintf("(node %s) >> ", utils.Int2Str(uuidNum))
				console.ready <- true
				continue
			}

			fmt.Print("\r\n[*]Waiting for response......")

			err := listen.LetListen(console.mgr, route, uuid)
			if err != nil {
				fmt.Printf("[*]Error: %s\n", err.Error())
			}

			console.status = fmt.Sprintf("(node %s) >> ", utils.Int2Str(uuidNum))

			console.ready <- true
		case "connect":
			if console.expectParams(fCommand, 2, NODE, 0) {
				break
			}

			fmt.Print("\r\n[*]Waiting for response......")

			err := handler.LetConnect(console.mgr, route, uuid, fCommand[1])
			if err != nil {
				fmt.Printf("[*]Error: %s\n", err.Error())
			}

			console.status = fmt.Sprintf("(node %s) >> ", utils.Int2Str(uuidNum))

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

			err := ssh.LetSSH(route, uuid)
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
				console.handleSSHPanelCommand(route, uuid)
				console.status = fmt.Sprintf("(node %s) >> ", utils.Int2Str(uuidNum))
				console.sshMode = false
			} else {
				fmt.Print("\r\n[*]Fail to connect to target host via ssh!")
				console.status = fmt.Sprintf("(node %s) >> ", utils.Int2Str(uuidNum))
				console.ready <- true
			}
		case "sshtunnel":
			if console.expectParams(fCommand, 3, NODE, 2) {
				break
			}

			sshTunnel := handler.NewSSHTunnel(fCommand[2], fCommand[1])

			console.status = "[*]Please choose the auth method(1.username/password 2.certificate): "
			console.ready <- true

			firstChoice := console.pretreatInput()
			if firstChoice == "1" {
				sshTunnel.Method = handler.UPMETHOD
			} else if firstChoice == "2" {
				sshTunnel.Method = handler.CERMETHOD
			} else {
				fmt.Print("\r\n[*]Please input 1 or 2!")
				console.status = fmt.Sprintf("(node %s) >> ", utils.Int2Str(uuidNum))
				console.ready <- true
				break
			}

			switch sshTunnel.Method {
			case handler.UPMETHOD:
				console.status = "[*]Please enter the username: "
				console.ready <- true
				sshTunnel.Username = console.pretreatInput()
				console.status = "[*]Please enter the password: "
				console.ready <- true
				sshTunnel.Password = console.pretreatInput()
			case handler.CERMETHOD:
				console.status = "[*]Please enter the username: "
				console.ready <- true
				sshTunnel.Username = console.pretreatInput()
				console.status = "[*]Please enter the filepath of the privkey: "
				console.ready <- true
				sshTunnel.CertificatePath = console.pretreatInput()
			}

			fmt.Print("\r\n[*]Waiting for response.....")

			err := sshTunnel.LetSSHTunnel(route, uuid)
			if err != nil {
				fmt.Printf("\r\n[*]Error: %s", err.Error())
				console.status = fmt.Sprintf("(node %s) >> ", utils.Int2Str(uuidNum))
				console.ready <- true
				break
			}

			if ok := <-console.mgr.ConsoleManager.OK; !ok {
				fmt.Print("\r\n[*]Fail to add target node via SSHTunnel!")
			}

			console.status = fmt.Sprintf("(node %s) >> ", utils.Int2Str(uuidNum))
			console.ready <- true
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

			err := socks.LetSocks(console.mgr, route, uuid)

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

			err := forward.LetForward(console.mgr, route, uuid)
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
					choice, err := utils.Str2Int(option)
					if err != nil {
						fmt.Printf("\r\n[*]Please input integer!")
					} else if choice > seq || choice < 0 {
						fmt.Printf("\r\n[*]Please input integer between 0~%d", seq)
					} else {
						fmt.Printf("\r\n[*]Closing......")
						handler.StopForward(console.mgr, uuid, choice)
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
			err := backward.LetBackward(console.mgr, route, uuid)
			if err != nil {
				fmt.Printf("\r\n[*]Error: %s", err.Error())
			} else {
				fmt.Print("\r\n[*]Backward start successfully!")
			}
			console.ready <- true
		case "stopbackward":
			if console.expectParams(fCommand, 1, NODE, 0) {
				break
			}

			seq, isRunning := handler.GetBackwardInfo(console.mgr, uuid)

			if isRunning {
				console.status = "[*]Do you really want to shutdown backward?(yes/no): "
				console.ready <- true
				option := console.pretreatInput()
				if option == "yes" {
					console.status = "[*]Please choose one to close: "
					console.ready <- true
					option := console.pretreatInput()
					choice, err := utils.Str2Int(option)
					if err != nil {
						fmt.Printf("\r\n[*]Please input integer!")
					} else if choice > seq || choice < 0 {
						fmt.Printf("\r\n[*]Please input integer between 0~%d", seq)
					} else {
						fmt.Printf("\r\n[*]Closing......")
						handler.StopBackward(console.mgr, uuid, route, choice)
						fmt.Printf("\r\n[*]Backward service has been closed successfully!")
					}
				} else if option == "no" {
				} else {
					fmt.Printf("\r\n[*]Please input yes/no!")
				}
				console.status = fmt.Sprintf("(node %s) >> ", utils.Int2Str(uuidNum))
			}
			console.ready <- true
		case "upload":
			if console.expectParams(fCommand, 3, NODE, 0) {
				break
			}

			console.mgr.FileManager.File.FilePath = fCommand[1]
			console.mgr.FileManager.File.FileName = fCommand[2]

			err := console.mgr.FileManager.File.SendFileStat(route, uuid, share.ADMIN)

			if err == nil && <-console.mgr.ConsoleManager.OK {
				go handler.StartBar(console.mgr.FileManager.File.StatusChan, console.mgr.FileManager.File.FileSize)
				console.mgr.FileManager.File.Upload(route, uuid, share.ADMIN)
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

			console.mgr.FileManager.File.Ask4Download(route, uuid)

			if <-console.mgr.ConsoleManager.OK {
				err := console.mgr.FileManager.File.CheckFileStat(route, uuid, share.ADMIN)
				if err == nil {
					go handler.StartBar(console.mgr.FileManager.File.StatusChan, console.mgr.FileManager.File.FileSize)
					console.mgr.FileManager.File.Receive(route, uuid, share.ADMIN)
				}
			} else {
				fmt.Print("\r\n[*]Unable to download file!")
			}
			console.ready <- true
		case "offline":
			if console.expectParams(fCommand, 1, NODE, 0) {
				break
			}

			handler.LetOffline(route, uuid)
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

func (console *Console) handleShellPanelCommand(route string, uuid string) {
	sMessage := protocol.PrepareAndDecideWhichSProtoToLower(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)

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

		protocol.ConstructMessage(sMessage, header, shellCommandMess, false)
		sMessage.SendMessage()
	}
}

func (console *Console) handleSSHPanelCommand(route string, uuid string) {
	sMessage := protocol.PrepareAndDecideWhichSProtoToLower(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)

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

		protocol.ConstructMessage(sMessage, header, sshCommandMess, false)
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
