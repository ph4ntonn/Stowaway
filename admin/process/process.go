/*
 * @Author: ph4ntom
 * @Date: 2021-03-16 16:10:23
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-04-04 11:45:55
 */
package process

import (
	"Stowaway/admin/cli"
	"Stowaway/admin/handler"
	"Stowaway/admin/initial"
	"Stowaway/admin/manager"
	"Stowaway/admin/topology"
	"Stowaway/protocol"
	"Stowaway/share"
	"fmt"
	"log"
	"net"
	"os"
)

type Admin struct {
	Conn        net.Conn
	Topology    *topology.Topology
	UserOptions *initial.Options

	BufferChan chan *BufferData
	// manager that needs to be shared with console
	mgr *manager.Manager
}

type BufferData struct {
	fHeader  *protocol.Header
	fMessage interface{}
}

func NewAdmin(options *initial.Options) *Admin {
	admin := new(Admin)
	admin.UserOptions = options
	admin.BufferChan = make(chan *BufferData, 10)
	return admin
}

func (admin *Admin) Run() {
	// Run a manager
	admin.mgr = manager.NewManager(share.NewFile())
	go admin.mgr.Run()
	// Init console
	console := cli.NewConsole()
	console.Init(admin.Topology, admin.mgr, admin.Conn, admin.UserOptions.Secret)
	// hanle all message comes from downstream
	go admin.handleConnFromDownstream(console)
	go admin.handleDataFromDownstream(console)
	// run a dispatcher to dispatch all socks TCP/UDP data
	go handler.DispathTCPData(admin.mgr)
	go handler.DispathUDPData(admin.mgr)
	// run a dispatcher to dispatch all forward data
	go handler.DispatchForwardData(admin.mgr)
	// start interactive panel
	console.Run()
}

func (admin *Admin) handleConnFromDownstream(console *cli.Console) {
	rMessage := protocol.PrepareAndDecideWhichRProtoFromUpper(admin.Conn, admin.UserOptions.Secret, protocol.ADMIN_UUID)

	for {
		fHeader, fMessage, err := protocol.DestructMessage(rMessage)
		if err != nil {
			log.Print("\r\n[*]Peer node seems offline!")
			os.Exit(0)
		}
		// Buffer chan to let data processing quicker, handleConnFromDownstream && handleDataFromDownstream can cooperate
		admin.BufferChan <- &BufferData{fHeader: fHeader, fMessage: fMessage}
	}

}

func (admin *Admin) handleDataFromDownstream(console *cli.Console) {
	for {
		data := <-admin.BufferChan

		switch data.fHeader.MessageType {
		case protocol.MYINFO:
			message := data.fMessage.(*protocol.MyInfo)
			// register new node
			task := &topology.TopoTask{
				Mode:     topology.UPDATEDETAIL,
				UUID:     data.fHeader.Sender,
				UserName: message.Username,
				HostName: message.Hostname,
			}
			admin.Topology.TaskChan <- task
		case protocol.SHELLRES:
			message := data.fMessage.(*protocol.ShellRes)
			if message.OK == 1 {
				console.OK <- true
			} else {
				console.OK <- false
			}
		case protocol.SHELLRESULT:
			message := data.fMessage.(*protocol.ShellResult)
			fmt.Print(message.Result)
		case protocol.LISTENRES:
			message := data.fMessage.(*protocol.ListenRes)
			if message.OK == 1 {
				fmt.Print("\n[*]Listen successfully!")
			} else {
				fmt.Print("\n[*]Listen failed!")
			}
		case protocol.SSHRES:
			message := data.fMessage.(*protocol.SSHRes)
			if message.OK == 1 {
				console.OK <- true
			} else {
				console.OK <- false
			}
		case protocol.SSHRESULT:
			message := data.fMessage.(*protocol.SSHResult)
			fmt.Printf("\r\033[K%s", message.Result)
			fmt.Printf("\r\n%s", console.Status)
		case protocol.FILESTATREQ:
			message := data.fMessage.(*protocol.FileStatReq)
			admin.mgr.File.FileSize = int64(message.FileSize)
			admin.mgr.File.SliceNum = message.SliceNum
			console.OK <- true
		case protocol.FILEDOWNRES:
			console.OK <- false
		case protocol.FILESTATRES:
			message := data.fMessage.(*protocol.FileStatRes)
			if message.OK == 1 {
				console.OK <- true
			} else {
				admin.mgr.File.Handler.Close()
				console.OK <- false
			}
		case protocol.FILEDATA:
			message := data.fMessage.(*protocol.FileData)
			admin.mgr.File.DataChan <- message.Data
		case protocol.FILEERR:
			admin.mgr.File.ErrChan <- true
		case protocol.SOCKSREADY:
			fallthrough
		case protocol.SOCKSTCPDATA:
			fallthrough
		case protocol.SOCKSTCPFIN:
			admin.mgr.SocksManager.SocksTCPDataChan <- data.fMessage
		case protocol.UDPASSSTART:
			message := data.fMessage.(*protocol.UDPAssStart)
			go handler.StartUDPAss(admin.mgr, admin.Topology, admin.Conn, admin.UserOptions.Secret, message.Seq)
		case protocol.SOCKSUDPDATA:
			message := data.fMessage.(*protocol.SocksUDPData)
			admin.mgr.SocksManager.SocksUDPDataChan <- message
		case protocol.FORWARDREADY:
			fallthrough
		case protocol.FORWARDDATA:
			fallthrough
		case protocol.FORWARDFIN:
			admin.mgr.ForwardManager.ForwardDataChan <- data.fMessage
		default:
			log.Print("\n[*]Unknown Message!")
		}
	}
}
