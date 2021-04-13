/*
 * @Author: ph4ntom
 * @Date: 2021-03-10 15:27:30
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-04-03 13:47:15
 */

package process

import (
	"Stowaway/agent/handler"
	"Stowaway/agent/initial"
	"Stowaway/agent/manager"
	"Stowaway/protocol"
	"Stowaway/share"
	"Stowaway/utils"
	"log"
	"net"
	"os"
)

type Agent struct {
	UUID string
	Memo string

	Conn        net.Conn
	UserOptions *initial.Options
}

func NewAgent(options *initial.Options) *Agent {
	agent := new(Agent)
	agent.UUID = protocol.TEMP_UUID
	agent.UserOptions = options
	return agent
}

func (agent *Agent) Run() {
	component := &protocol.MessageComponent{Secret: agent.UserOptions.Secret, Conn: agent.Conn, UUID: agent.UUID}
	agent.sendMyInfo()
	// run manager
	mgr := manager.NewManager(share.NewFile())
	go mgr.Run()
	// run dispatchers to dispatch all kinds of message
	go handler.DispathSocksMess(mgr, component)
	go handler.DispatchForwardMess(mgr, component)
	go handler.DispatchBackwardMess(mgr, component)
	go handler.DispatchFileMess(mgr, component)
	go handler.DispatchSSHMess(mgr, component)
	go handler.DispatchShellMess(mgr, component)
	// process data from upstream
	agent.handleDataFromUpstream(mgr, component)
	//agent.handleDataFromDownstream()
}

func (agent *Agent) sendMyInfo() {
	sMessage := protocol.PrepareAndDecideWhichSProtoToUpper(agent.Conn, agent.UserOptions.Secret, agent.UUID)
	header := &protocol.Header{
		Sender:      agent.UUID,
		Accepter:    protocol.ADMIN_UUID,
		MessageType: protocol.MYINFO,
		RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))), // No need to set route when agent send mess to admin
		Route:       protocol.TEMP_ROUTE,
	}

	hostname, username := utils.GetSystemInfo()

	myInfoMess := &protocol.MyInfo{
		UUIDLen:     uint16(len(agent.UUID)),
		UUID:        agent.UUID,
		UsernameLen: uint64(len(username)),
		Username:    username,
		HostnameLen: uint64(len(hostname)),
		Hostname:    hostname,
	}

	protocol.ConstructMessage(sMessage, header, myInfoMess)
	sMessage.SendMessage()
}

func (agent *Agent) handleDataFromUpstream(mgr *manager.Manager, component *protocol.MessageComponent) {
	rMessage := protocol.PrepareAndDecideWhichRProtoFromUpper(agent.Conn, agent.UserOptions.Secret, agent.UUID)

	for {
		header, message, err := protocol.DestructMessage(rMessage)
		if err != nil {
			log.Println("[*]Peer node seems offline!")
			os.Exit(0)
		}

		if header.Accepter == agent.UUID {
			switch header.MessageType {
			case protocol.MYMEMO:
				message := message.(*protocol.MyMemo)
				agent.Memo = message.Memo // no need to pass this like all the message below,just change memo directly
			case protocol.SHELLREQ:
				fallthrough
			case protocol.SHELLCOMMAND:
				mgr.ShellManager.ShellMessChan <- message
			case protocol.SSHREQ:
				fallthrough
			case protocol.SSHCOMMAND:
				mgr.SSHManager.SSHMessChan <- message
			case protocol.FILESTATREQ:
				fallthrough
			case protocol.FILESTATRES:
				fallthrough
			case protocol.FILEDATA:
				fallthrough
			case protocol.FILEERR:
				fallthrough
			case protocol.FILEDOWNREQ:
				mgr.FileManager.FileMessChan <- message
			case protocol.SOCKSSTART:
				fallthrough
			case protocol.SOCKSTCPDATA:
				fallthrough
			case protocol.SOCKSTCPFIN:
				mgr.SocksManager.SocksMessChan <- message
			case protocol.UDPASSRES:
				fallthrough
			case protocol.SOCKSUDPDATA:
				mgr.SocksManager.SocksMessChan <- message
			case protocol.FORWARDTEST:
				fallthrough
			case protocol.FORWARDSTART:
				fallthrough
			case protocol.FORWARDDATA:
				fallthrough
			case protocol.FORWARDFIN:
				mgr.ForwardManager.ForwardMessChan <- message
			case protocol.BACKWARDTEST:
				fallthrough
			case protocol.BACKWARDSEQ:
				fallthrough
			case protocol.BACKWARDFIN:
				fallthrough
			case protocol.BACKWARDDATA:
				mgr.BackwardManager.BackwardMessChan <- message
			case protocol.OFFLINE:
				os.Exit(0)
			default:
				log.Println("[*]Unknown Message!")
			}
		}
	}
}
