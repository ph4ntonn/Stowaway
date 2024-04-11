//go:build !windows

package process

import (
	"os"

	"Stowaway/admin/cli"
	"Stowaway/admin/handler"
	"Stowaway/admin/initial"
	"Stowaway/admin/manager"
	"Stowaway/admin/printer"
	"Stowaway/admin/topology"
	"Stowaway/global"
	"Stowaway/protocol"
	"Stowaway/share"

	"github.com/nsf/termbox-go"
)

type Admin struct {
	mgr      *manager.Manager
	options  *initial.Options
	topology *topology.Topology
}

func NewAdmin(opt *initial.Options, topo *topology.Topology) *Admin {
	admin := new(Admin)
	admin.topology = topo
	admin.options = opt
	return admin
}

func (admin *Admin) Run() {
	admin.mgr = manager.NewManager(share.NewFile())
	go admin.mgr.Run()
	// Init console
	console := cli.NewConsole()
	console.Init(admin.topology, admin.mgr)
	// hanle all message comes from downstream
	go admin.handleMessFromDownstream(console)
	// run a dispatcher to dispatch different kinds of message
	go handler.DispatchListenMess(admin.mgr, admin.topology)
	go handler.DispatchConnectMess(admin.mgr)
	go handler.DispathSocksMess(admin.mgr, admin.topology)
	go handler.DispatchForwardMess(admin.mgr)
	go handler.DispatchBackwardMess(admin.mgr, admin.topology)
	go handler.DispatchFileMess(admin.mgr)
	go handler.DispatchSSHMess(admin.mgr)
	go handler.DispatchSSHTunnelMess(admin.mgr)
	go handler.DispatchShellMess(admin.mgr)
	go handler.DispatchInfoMess(admin.mgr, admin.topology)
	go DispatchChildrenMess(admin.mgr, admin.topology)
	// if options.Heartbeat set, send hearbeat packet to agent
	if admin.options.Heartbeat {
		go handler.LetHeartbeat(admin.topology)
	}
	// start interactive panel
	console.Run()
}

func (admin *Admin) handleMessFromDownstream(console *cli.Console) {
	rMessage := protocol.NewDownMsg(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)

	for {
		header, message, err := protocol.DestructMessage(rMessage)
		if err != nil {
			printer.Fail("\r\n[*] Peer node seems offline!")
			// wait for user to exit
			printer.Fail("\r\n[*] Press any key to exit")
			termbox.PollEvent()
			// close termbox
			termbox.Close()
			os.Exit(0)
		}

		switch header.MessageType {
		case protocol.MYINFO:
			admin.mgr.InfoManager.InfoMessChan <- message
		case protocol.SHELLRES:
			fallthrough
		case protocol.SHELLRESULT:
			fallthrough
		case protocol.SHELLEXIT:
			admin.mgr.ShellManager.ShellMessChan <- message
		case protocol.SSHRES:
			fallthrough
		case protocol.SSHRESULT:
			fallthrough
		case protocol.SSHEXIT:
			admin.mgr.SSHManager.SSHMessChan <- message
		case protocol.SSHTUNNELRES:
			admin.mgr.SSHTunnelManager.SSHTunnelMessChan <- message
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
		case protocol.BACKWARDSTOPDONE:
			fallthrough
		case protocol.BACKWARDSTART:
			admin.mgr.BackwardManager.BackwardMessChan <- message
		case protocol.CHILDUUIDREQ: // include "connect" && "listen" func, let ListenManager do all this stuff,ConnectManager can just watch
			fallthrough
		case protocol.LISTENRES:
			admin.mgr.ListenManager.ListenMessChan <- message
		case protocol.CONNECTDONE:
			admin.mgr.ConnectManager.ConnectMessChan <- message
		case protocol.NODEREONLINE:
			fallthrough
		case protocol.NODEOFFLINE:
			admin.mgr.ChildrenManager.ChildrenMessChan <- message
		default:
			printer.Fail("\r\n[*] Unknown Message!")
		}
	}
}
