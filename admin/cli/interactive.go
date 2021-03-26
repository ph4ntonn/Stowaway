/*
 * @Author: ph4ntom
 * @Date: 2021-03-10 18:11:41
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-03-26 16:47:56
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
	"time"

	"github.com/eiannone/keyboard"
)

const (
	MAIN = iota
	NODE
)

type Console struct {
	// Admin status
	UUID         string
	Conn         net.Conn
	Secret       string
	CryptoSecret []byte
	Topology     *topology.Topology
	// console original status
	Status     string
	OK         chan bool
	ready      chan bool
	getCommand chan string
	// manager that needs to be shared with main thread
	mgr *manager.Manager
}

func NewConsole() *Console {
	console := new(Console)
	console.Status = "(admin) >> "
	console.OK = make(chan bool)
	console.ready = make(chan bool)
	console.getCommand = make(chan string)
	return console
}

func (console *Console) Init(tTopology *topology.Topology, myManager *manager.Manager, conn net.Conn, uuid string, secret string, cryptoSecret []byte) {
	console.UUID = uuid
	console.Conn = conn
	console.Secret = secret
	console.CryptoSecret = cryptoSecret
	console.Topology = tTopology
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

	history := NewHistory()
	history.Run()

	keysEvents, _ := keyboard.GetKeys(10)

	fmt.Print(console.Status)
	for {
		event := <-keysEvents
		if event.Err != nil {
			panic(event.Err)
		}

		if (event.Key != keyboard.KeyEnter && event.Rune >= 0x20 && event.Rune <= 0x7F) || event.Key == keyboard.KeySpace {
			fmt.Print("\033[u\033[K\r")
			fmt.Print(console.Status)
			if event.Key == keyboard.KeySpace {
				leftCommand = leftCommand + " "
			} else {
				leftCommand = leftCommand + string(event.Rune)
			}
			fmt.Print(leftCommand + rightCommand)
			fmt.Print(string(bytes.Repeat([]byte("\b"), len(rightCommand))))
		} else if event.Key == keyboard.KeyBackspace2 || event.Key == keyboard.KeyBackspace {
			fmt.Print("\033[u\033[K\r")
			fmt.Print(console.Status)
			if len(leftCommand) >= 1 {
				leftCommand = leftCommand[:len(leftCommand)-1]
			}
			fmt.Print(leftCommand + rightCommand)
			fmt.Print(string(bytes.Repeat([]byte("\b"), len(rightCommand))))
		} else if event.Key == keyboard.KeyEnter {
			command := leftCommand + rightCommand
			if command != "" {
				history.Record <- command
			}
			console.getCommand <- command
			isGoingOn = false
			leftCommand = ""
			rightCommand = ""
			<-console.ready // avoid situation that console.Status is printed before it's changed
			fmt.Print("\r\n")
			fmt.Print(console.Status)
		} else if event.Key == keyboard.KeyArrowUp {
			fmt.Print("\033[u\033[K\r")
			fmt.Print(console.Status)
			if !isGoingOn {
				history.Search <- BEGIN
				isGoingOn = true
			} else {
				history.Search <- NEXT
			}
			leftCommand = <-history.Result
			rightCommand = ""
		} else if event.Key == keyboard.KeyArrowDown {
			fmt.Print("\033[u\033[K\r")
			fmt.Print(console.Status)
			if isGoingOn {
				history.Search <- PREV
				leftCommand = <-history.Result
			} else {
				leftCommand = ""
			}
			rightCommand = ""
		} else if event.Key == keyboard.KeyArrowLeft {
			fmt.Print("\033[u\033[K\r")
			fmt.Print(console.Status)
			if len(leftCommand) >= 1 {
				rightCommand = leftCommand[len(leftCommand)-1:] + rightCommand
				leftCommand = leftCommand[:len(leftCommand)-1]
			}
			fmt.Print(leftCommand + rightCommand)
			fmt.Print(string(bytes.Repeat([]byte("\b"), len(rightCommand))))
		} else if event.Key == keyboard.KeyArrowRight {
			fmt.Print("\033[u\033[K\r")
			fmt.Print(console.Status)
			if len(rightCommand) > 1 {
				leftCommand = leftCommand + rightCommand[:1]
				rightCommand = rightCommand[1:]
			} else if len(rightCommand) == 1 {
				leftCommand = leftCommand + rightCommand[:1]
				rightCommand = ""
			}
			fmt.Print(leftCommand + rightCommand)
			fmt.Print(string(bytes.Repeat([]byte("\b"), len(rightCommand))))
		} else if event.Key == keyboard.KeyCtrlC {
			fmt.Print("\n[*]BYE!")
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
			if console.expectParamsNum(fCommand, 2, MAIN, 1) {
				break
			}
			idNum, _ := utils.Str2Int(fCommand[1])
			task := &topology.TopoTask{
				Mode:  topology.CHECKNODE,
				IDNum: idNum,
			}
			console.Topology.TaskChan <- task
			result := <-console.Topology.ResultChan
			if result.IsExist {
				console.Status = fmt.Sprintf("(node %s) >> ", fCommand[1])
				console.handleNodePanelCommand(idNum)
				console.Status = "(admin) >> "
			} else {
				fmt.Printf("\n[*]Node %s doesn't exist!", fCommand[1])
			}
			console.ready <- true
		case "detail":
			if console.expectParamsNum(fCommand, 1, MAIN, 0) {
				break
			}
			task := &topology.TopoTask{
				Mode: topology.SHOWDETAIL,
			}
			console.Topology.TaskChan <- task
			<-console.Topology.ResultChan
			console.ready <- true
		case "tree":
			if console.expectParamsNum(fCommand, 1, MAIN, 0) {
				break
			}
			task := &topology.TopoTask{
				Mode: topology.SHOWTREE,
			}
			console.Topology.TaskChan <- task
			<-console.Topology.ResultChan
			console.ready <- true
		case "":
			if console.expectParamsNum(fCommand, 1, MAIN, 0) {
				break
			}
			console.ready <- true
		case "help":
			if console.expectParamsNum(fCommand, 1, MAIN, 0) {
				break
			}
			ShowMainHelp()
			console.ready <- true
		case "exit":
			if console.expectParamsNum(fCommand, 1, MAIN, 0) {
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

func (console *Console) handleNodePanelCommand(idNum int) {
	topoTask := &topology.TopoTask{
		Mode: topology.CALCULATE,
	}
	console.Topology.TaskChan <- topoTask
	routeResult := <-console.Topology.ResultChan
	route := routeResult.RouteInfo[idNum]

	topoTask = &topology.TopoTask{
		Mode:  topology.GETNODEID,
		IDNum: idNum,
	}
	console.Topology.TaskChan <- topoTask
	topoResult := <-console.Topology.ResultChan
	nodeID := topoResult.NodeID

	component := &protocol.MessageComponent{
		Secret: console.Secret,
		Conn:   console.Conn,
		UUID:   console.UUID,
	}

	console.ready <- true

	for {
		tCommand := console.pretreatInput()
		fCommand := strings.Split(tCommand, " ")
		switch fCommand[0] {
		case "addmemo":
			handler.AddMemo(component, console.Topology.TaskChan, fCommand[1:], nodeID, route)
			console.ready <- true
		case "delmemo":
			if console.expectParamsNum(fCommand, 1, NODE, 0) {
				break
			}
			handler.DelMemo(component, console.Topology.TaskChan, nodeID, route)
			console.ready <- true
		case "shell":
			if console.expectParamsNum(fCommand, 1, NODE, 0) {
				break
			}

			handler.LetShellStart(component, route, nodeID)
			console.Status = ""

			fmt.Print("\r\n[*]Waiting for response.....")

			if <-console.OK {
				console.handleShellPanelCommand(component, route, nodeID)
			}

			console.Status = fmt.Sprintf("(node %s) >> ", utils.Int2Str(idNum))
			console.ready <- true
		case "listen":
			if console.expectParamsNum(fCommand, 2, NODE, 0) {
				break
			}
			handler.LetListen(component, route, nodeID, fCommand[1])
			console.ready <- true
		case "ssh":
			if console.expectParamsNum(fCommand, 2, NODE, 0) {
				break
			}

			console.Status = "" // temporarily set status to ""
			console.ready <- true

			var err error
			ssh := handler.NewSSH()
			ssh.Addr = fCommand[1]

			time.Sleep(300 * time.Millisecond) // sleep 300 ms to make sure ```fmt.Print("\r\n") fmt.Print(console.Status)``` run first!
			fmt.Print("[*]Please choose the auth method(1.username/password 2.certificate): ")
			firstChoice := console.pretreatInput()

			if firstChoice == "1" {
				ssh.Method = handler.UPMETHOD
			} else if firstChoice == "2" {
				ssh.Method = handler.CERMETHOD
			} else {
				time.Sleep(300 * time.Millisecond)
				fmt.Print("\r\n[*]Please input 1 or 2!")
				console.Status = fmt.Sprintf("(node %s) >> ", utils.Int2Str(idNum))
				console.ready <- true
				break
			}

			console.ready <- true

			switch ssh.Method {
			case handler.UPMETHOD:
				time.Sleep(300 * time.Millisecond)
				fmt.Print("[*]Please enter the username: ")
				ssh.Username = console.pretreatInput()
				console.ready <- true
				time.Sleep(300 * time.Millisecond)
				fmt.Print("[*]Please enter the password: ")
				ssh.Password = console.pretreatInput()
				err = ssh.LetSSH(component, route, nodeID)
			case handler.CERMETHOD:
				time.Sleep(300 * time.Millisecond)
				fmt.Print("[*]Please enter the username: ")
				ssh.Username = console.pretreatInput()
				console.ready <- true
				time.Sleep(300 * time.Millisecond)
				fmt.Print("[*]Please enter the filepath of the privkey: ")
				ssh.CertificatePath = console.pretreatInput()
				err = ssh.LetSSH(component, route, nodeID)
			}

			if err != nil {
				time.Sleep(300 * time.Millisecond)
				fmt.Printf("\r\n[*]Error: %s", err.Error())
				console.Status = fmt.Sprintf("(node %s) >> ", utils.Int2Str(idNum))
				console.ready <- true
				break
			}

			fmt.Print("\r\n[*]Waiting for response.....")

			if <-console.OK {
				console.Status = fmt.Sprintf("(ssh %s) >> ", ssh.Addr)
				console.handleSSHPanelCommand(component, route, nodeID)
			}

			console.Status = fmt.Sprintf("(node %s) >> ", utils.Int2Str(idNum))
			console.ready <- true
		case "socks":
			if console.expectParamsNum(fCommand, 2, NODE, 1) {
				if console.expectParamsNum(fCommand, 4, NODE, 1) {
					break
				}
			}
			socks := handler.NewSocks()
			socks.Port, _ = utils.Str2Int(fCommand[1])
			if len(fCommand) > 2 {
				socks.Username = fCommand[2]
				socks.Password = fCommand[3]
			}

			if err := socks.LetSocks(component, console.mgr, route, nodeID, idNum); err != nil {
				fmt.Printf("\r\n[*]Error: %s", err.Error())
			}
			console.ready <- true
		case "upload":
			if console.expectParamsNum(fCommand, 3, NODE, 0) {
				break
			}

			console.mgr.File.FilePath = fCommand[1]
			console.mgr.File.FileName = fCommand[2]

			err := console.mgr.File.SendFileStat(component, route, nodeID, share.ADMIN)

			if err == nil && <-console.OK {
				go handler.StartBar(console.mgr.File.StatusChan, console.mgr.File.FileSize)
				console.mgr.File.Upload(component, route, nodeID, share.ADMIN)
			} else if err != nil {
				fmt.Printf("\r\n[*]Error: %s", err.Error())
			}

			console.ready <- true
		case "download":
			if console.expectParamsNum(fCommand, 3, NODE, 0) {
				break
			}

			console.mgr.File.FilePath = fCommand[1]
			console.mgr.File.FileName = fCommand[2]

			console.mgr.File.Ask4Download(component, route, nodeID)

			if <-console.OK {
				err := console.mgr.File.CheckFileStat(component, route, nodeID, share.ADMIN)
				if err == nil {
					go handler.StartBar(console.mgr.File.StatusChan, console.mgr.File.FileSize)
					console.mgr.File.Receive(component, route, nodeID, share.ADMIN)
				}
			}

			console.ready <- true
		case "offline":
			if console.expectParamsNum(fCommand, 1, NODE, 0) {
				break
			}
			handler.LetOffline(component, route, nodeID)
			console.ready <- true
		case "":
			if console.expectParamsNum(fCommand, 1, NODE, 0) {
				break
			}
			console.ready <- true
		case "help":
			if console.expectParamsNum(fCommand, 1, NODE, 0) {
				break
			}
			ShowNodeHelp()
			console.ready <- true
		case "exit":
			if console.expectParamsNum(fCommand, 1, NODE, 0) {
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

func (console *Console) handleShellPanelCommand(component *protocol.MessageComponent, route string, nodeID string) {
	sMessage := protocol.PrepareAndDecideWhichSProtoToLower(component.Conn, component.Secret, component.UUID)

	header := &protocol.Header{
		Sender:      protocol.ADMIN_UUID,
		Accepter:    nodeID,
		MessageType: protocol.SHELLCOMMAND,
		RouteLen:    uint32(len([]byte(route))),
		Route:       route,
	}

	console.ready <- true

	for {
		tCommand := <-console.getCommand

		if tCommand == "exit" {
			return
		}

		console.ready <- true

		fCommand := tCommand + "\n"

		shellCommandMess := &protocol.ShellCommand{
			CommandLen: uint64(len(fCommand)),
			Command:    fCommand,
		}

		protocol.ConstructMessage(sMessage, header, shellCommandMess)
		sMessage.SendMessage()
	}
}

func (console *Console) handleSSHPanelCommand(component *protocol.MessageComponent, route string, nodeID string) {
	sMessage := protocol.PrepareAndDecideWhichSProtoToLower(component.Conn, component.Secret, component.UUID)

	header := &protocol.Header{
		Sender:      protocol.ADMIN_UUID,
		Accepter:    nodeID,
		MessageType: protocol.SSHCOMMAND,
		RouteLen:    uint32(len([]byte(route))),
		Route:       route,
	}

	console.ready <- true

	for {
		tCommand := <-console.getCommand

		if tCommand == "exit" {
			return
		}

		console.ready <- true

		if tCommand == "" {
			continue
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

func (console *Console) expectParamsNum(params []string, num int, mode int, needToBeInt int) bool {
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

	if needToBeInt != 0 {
		_, err := utils.Str2Int(params[needToBeInt])
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
