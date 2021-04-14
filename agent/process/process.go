/*
 * @Author: ph4ntom
 * @Date: 2021-03-10 15:27:30
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-04-03 13:47:15
 */

package process

import (
	"Stowaway/agent/handler"
	"Stowaway/agent/manager"
	"Stowaway/global"
	"Stowaway/protocol"
	"Stowaway/share"
	"Stowaway/utils"
	"log"
	"os"
)

type Agent struct {
	UUID string
	Memo string
}

func NewAgent() *Agent {
	agent := new(Agent)
	agent.UUID = protocol.TEMP_UUID
	return agent
}

func (agent *Agent) Run() {
	agent.sendMyInfo()
	// run manager
	mgr := manager.NewManager(share.NewFile())
	go mgr.Run()
	// run dispatchers to dispatch all kinds of message
	go handler.DispathSocksMess(mgr)
	go handler.DispatchForwardMess(mgr)
	go handler.DispatchBackwardMess(mgr)
	go handler.DispatchFileMess(mgr)
	go handler.DispatchSSHMess(mgr)
	go handler.DispatchShellMess(mgr)
	// process data from upstream
	agent.handleDataFromUpstream(mgr)
	//agent.handleDataFromDownstream()
}

func (agent *Agent) sendMyInfo() {
	sMessage := protocol.PrepareAndDecideWhichSProtoToUpper(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)
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

func (agent *Agent) handleDataFromUpstream(mgr *manager.Manager) {
	rMessage := protocol.PrepareAndDecideWhichRProtoFromUpper(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)

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
				fallthrough
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
