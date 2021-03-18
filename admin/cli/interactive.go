/*
 * @Author: ph4ntom
 * @Date: 2021-03-10 18:11:41
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-03-18 18:39:25
 */
package cli

import (
	"Stowaway/admin/handler"
	"Stowaway/admin/topology"
	"Stowaway/protocol"
	"Stowaway/utils"
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
	ID           string
	Conn         net.Conn
	Secret       string
	CryptoSecret []byte
	Topology     *topology.Topology
	Route        *topology.Route
	// console original status
	Status     string
	OK         chan bool
	Ready      chan bool
	getCommand chan string
}

func NewConsole() *Console {
	console := new(Console)
	console.Status = "(admin) >> "
	console.OK = make(chan bool)
	console.Ready = make(chan bool)
	console.getCommand = make(chan string)
	return console
}

func (console *Console) Init(tTopology *topology.Topology, tRoute *topology.Route, conn net.Conn, ID string, secret string, cryptoSecret []byte) {
	console.ID = ID
	console.Conn = conn
	console.Secret = secret
	console.CryptoSecret = cryptoSecret
	console.Topology = tTopology
	console.Route = tRoute
}

func (console *Console) Run() {
	go console.handleMainPanelCommand()
	console.mainPanel() // block admin
}

func (console *Console) mainPanel() {
	var (
		command   string
		isGoingOn bool
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
			if event.Key == keyboard.KeySpace {
				fmt.Print(" ")
				command = command + " "
			} else {
				fmt.Print(string(event.Rune))
				command = command + string(event.Rune)
			}
		} else if event.Key == keyboard.KeyBackspace2 || event.Key == keyboard.KeyBackspace {
			var fLen int
			cLen := len(command) - 1

			if cLen != -1 {
				fmt.Print("\b \b")
			}

			if cLen >= 0 {
				fLen = cLen
			} else {
				fLen = 0
			}
			command = command[:fLen]
		} else if event.Key == keyboard.KeyEnter {
			if command != "" {
				history.Record <- command
			}
			console.getCommand <- command
			isGoingOn = false
			command = ""
			<-console.Ready // avoid situation that console.Status is printed before it's changed
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
			command = <-history.Result
		} else if event.Key == keyboard.KeyArrowDown {
			fmt.Print("\033[u\033[K\r")
			fmt.Print(console.Status)
			if isGoingOn {
				history.Search <- PREV
				command = <-history.Result
			}
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
		tCommand := <-console.getCommand
		tCommand = strings.TrimRight(tCommand, " \t\r\n")
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
			console.Ready <- true
		case "detail":
			if console.expectParamsNum(fCommand, 1, MAIN, 0) {
				break
			}
			task := &topology.TopoTask{
				Mode: topology.SHOWDETAIL,
			}
			console.Topology.TaskChan <- task
			<-console.Topology.ResultChan
			console.Ready <- true
		case "tree":
			if console.expectParamsNum(fCommand, 1, MAIN, 0) {
				break
			}
			task := &topology.TopoTask{
				Mode: topology.SHOWTREE,
			}
			console.Topology.TaskChan <- task
			<-console.Topology.ResultChan
			console.Ready <- true
		case "":
			if console.expectParamsNum(fCommand, 0, MAIN, 0) {
				break
			}
			console.Ready <- true
		case "help":
			if console.expectParamsNum(fCommand, 1, MAIN, 0) {
				break
			}
			ShowMainHelp()
			console.Ready <- true
		case "exit":
			if console.expectParamsNum(fCommand, 1, MAIN, 0) {
				break
			}
			fmt.Print("\n[*]BYE!")
			os.Exit(0)
		default:
			fmt.Print("\n[*]Unknown Command!\n")
			ShowMainHelp()
			console.Ready <- true
		}
	}
}

func (console *Console) handleNodePanelCommand(idNum int) {
	sMessage := protocol.PrepareAndDecideWhichSProto(console.Conn, console.Secret, console.ID)

	routeTask := &topology.RouteTask{
		Mode: topology.CALCULATE,
	}
	console.Route.TaskChan <- routeTask
	routeResult := <-console.Route.ResultChan
	route := routeResult.RouteInfo[idNum]

	topoTask := &topology.TopoTask{
		Mode:  topology.GETNODEID,
		IDNum: idNum,
	}
	console.Topology.TaskChan <- topoTask
	topoResult := <-console.Topology.ResultChan
	nodeID := topoResult.NodeID

	console.Ready <- true

	for {
		tCommand := <-console.getCommand
		tCommand = strings.TrimRight(tCommand, " \t\r\n")
		fCommand := strings.Split(tCommand, " ")
		switch fCommand[0] {
		case "addmemo":
			handler.AddMemo(sMessage, console.Topology.TaskChan, fCommand[1:], nodeID, route)
			console.Ready <- true
		case "delmemo":
			if console.expectParamsNum(fCommand, 1, NODE, 0) {
				break
			}
			handler.DelMemo(sMessage, console.Topology.TaskChan, nodeID, route)
			console.Ready <- true
		case "shell":
			if console.expectParamsNum(fCommand, 1, NODE, 0) {
				break
			}
			handler.LetShellStart(sMessage, route, nodeID)
			if <-console.OK {
				console.Status = ""
				console.handleShellPanelCommand(sMessage, route, nodeID)
				console.Status = fmt.Sprintf("(node %s) >> ", utils.Int2Str(idNum))
			}
			console.Ready <- true
		case "listen":
			if console.expectParamsNum(fCommand, 2, NODE, 0) {
				break
			}
			handler.LetListen(sMessage, route, nodeID, fCommand[1])
			console.Ready <- true
		case "":
			if console.expectParamsNum(fCommand, 0, NODE, 1) {
				break
			}
			console.Ready <- true
		case "help":
			if console.expectParamsNum(fCommand, 1, NODE, 1) {
				break
			}
			ShowNodeHelp()
			console.Ready <- true
		case "exit":
			if console.expectParamsNum(fCommand, 1, NODE, 1) {
				break
			}
			return
		default:
			fmt.Print("\n[*]Unknown Command!\n")
			ShowNodeHelp()
			console.Ready <- true
		}
	}
}

func (console *Console) handleShellPanelCommand(sMessage protocol.Message, route string, nodeID string) {
	header := protocol.Header{
		Sender:      protocol.ADMIN_UUID,
		Accepter:    nodeID,
		MessageType: protocol.SHELLCOMMAND,
		RouteLen:    uint32(len([]byte(route))),
		Route:       route,
	}

	console.Ready <- true

	for {
		tCommand := <-console.getCommand

		if tCommand == "exit" {
			return
		}

		console.Ready <- true

		fCommand := tCommand + "\n"

		shellCommandMess := protocol.ShellCommand{
			CommandLen: uint64(len(fCommand)),
			Command:    fCommand,
		}

		protocol.ConstructMessage(sMessage, header, shellCommandMess)
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
		console.Ready <- true
		return true
	}

	if needToBeInt != 0 {
		_, err := utils.Str2Int(params[needToBeInt])
		if err != nil {
			console.Ready <- true
			return true
		}
	}

	return false
}
