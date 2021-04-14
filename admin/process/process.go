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
	"log"
	"net"
	"os"
)

type Admin struct {
	Conn        net.Conn
	Topology    *topology.Topology
	UserOptions *initial.Options

	// manager that needs to be shared with console
	mgr *manager.Manager
}

func NewAdmin(options *initial.Options) *Admin {
	admin := new(Admin)
	admin.UserOptions = options
	return admin
}

func (admin *Admin) Run() {
	admin.mgr = manager.NewManager(share.NewFile())
	go admin.mgr.Run()
	// Init console
	console := cli.NewConsole()
	console.Init(admin.Topology, admin.mgr, admin.Conn, admin.UserOptions.Secret)
	// hanle all message comes from downstream
	go admin.handleMessFromDownstream(console)
	// run a dispatcher to dispatch different kinds of message
	go handler.DispathSocksMess(admin.mgr, admin.Topology, admin.Conn, admin.UserOptions.Secret)
	go handler.DispatchForwardMess(admin.mgr)
	go handler.DispatchBackwardMess(admin.mgr, admin.Topology, admin.Conn, admin.UserOptions.Secret)
	go handler.DispatchFileMess(admin.mgr)
	go handler.DispatchSSHMess(admin.mgr)
	go handler.DispatchShellMess(admin.mgr)
	go handler.DispatchInfoMess(admin.mgr, admin.Topology)
	// start interactive panel
	console.Run()
}

func (admin *Admin) handleMessFromDownstream(console *cli.Console) {
	rMessage := protocol.PrepareAndDecideWhichRProtoFromUpper(admin.Conn, admin.UserOptions.Secret, protocol.ADMIN_UUID)

	for {
		header, message, err := protocol.DestructMessage(rMessage)
		if err != nil {
			log.Print("\r\n[*]Peer node seems offline!")
			os.Exit(0)
		}

		switch header.MessageType {
		case protocol.MYINFO:
			admin.mgr.InfoManager.InfoMessChan <- message
		case protocol.SHELLRES:
			fallthrough
		case protocol.SHELLRESULT:
			admin.mgr.ShellManager.ShellMessChan <- message
		case protocol.SSHRES:
			fallthrough
		case protocol.SSHRESULT:
			admin.mgr.SSHManager.SSHMessChan <- message
		case protocol.FILESTATREQ:
			fallthrough
		case protocol.FILEDOWNRES:
			fallthrough
		case protocol.FILESTATRES:
			fallthrough
		case protocol.FILEDATA:
			fallthrough
		case protocol.FILEERR:
			admin.mgr.FileManager.FileMessChan <- message
		case protocol.SOCKSREADY:
			fallthrough
		case protocol.SOCKSTCPDATA:
			fallthrough
		case protocol.SOCKSTCPFIN:
			fallthrough
		case protocol.UDPASSSTART:
			fallthrough
		case protocol.SOCKSUDPDATA:
			admin.mgr.SocksManager.SocksMessChan <- message
		case protocol.FORWARDREADY:
			fallthrough
		case protocol.FORWARDDATA:
			fallthrough
		case protocol.FORWARDFIN:
			admin.mgr.ForwardManager.ForwardMessChan <- message
		case protocol.BACKWARDREADY:
			fallthrough
		case protocol.BACKWARDDATA:
			fallthrough
		case protocol.BACKWARDFIN:
			fallthrough
		case protocol.BACKWARDSTART:
			admin.mgr.BackwardManager.BackwardMessChan <- message
		default:
			log.Print("\n[*]Unknown Message!")
		}
	}
}
