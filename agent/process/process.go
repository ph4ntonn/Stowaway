/*
 * @Author: ph4ntom
 * @Date: 2021-03-10 15:27:30
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-03-20 13:41:47
 */

package process

import (
	"Stowaway/agent/handler"
	"Stowaway/agent/initial"
	"Stowaway/crypto"
	"Stowaway/protocol"
	"Stowaway/utils"
	"log"
	"net"
)

type Agent struct {
	ID           string
	Conn         net.Conn
	Memo         string
	CryptoSecret []byte
	UserOptions  *initial.Options
}

func NewAgent(options *initial.Options) *Agent {
	agent := new(Agent)
	agent.ID = protocol.TEMP_UUID
	agent.CryptoSecret, _ = crypto.KeyPadding([]byte(options.Secret))
	agent.UserOptions = options
	return agent
}

func (agent *Agent) Run() {
	agent.sendMyInfo()
	agent.handleDataFromUpstream()
	//agent.handleDataFromDownstream()
}

func (agent *Agent) sendMyInfo() {
	sMessage := protocol.PrepareAndDecideWhichSProto(agent.Conn, agent.UserOptions.Secret, agent.ID)
	header := protocol.Header{
		Sender:      agent.ID,
		Accepter:    protocol.ADMIN_UUID,
		MessageType: protocol.MYINFO,
		RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))), // No need to set route when agent send mess to admin
		Route:       protocol.TEMP_ROUTE,
	}

	hostname, username := utils.GetSystemInfo()

	myInfoMess := protocol.MyInfo{
		UsernameLen: uint64(len(username)),
		Username:    username,
		HostnameLen: uint64(len(hostname)),
		Hostname:    hostname,
	}

	protocol.ConstructMessage(sMessage, header, myInfoMess)
	sMessage.SendMessage()
}

func (agent *Agent) handleDataFromUpstream() {
	rMessage := protocol.PrepareAndDecideWhichRProto(agent.Conn, agent.UserOptions.Secret, agent.ID)
	//sMessage := protocol.PrepareAndDecideWhichSProto(agent.Conn, agent.UserOptions.Secret, agent.ID)
	shell := handler.NewShell()
	mySSH := handler.NewSSH()

	for {
		fHeader, fMessage, err := protocol.DestructMessage(rMessage)
		if err != nil {
			log.Println("[*]Peer node seems offline!")
			break
		}
		switch fHeader.MessageType {
		case protocol.MYMEMO:
			message := fMessage.(*protocol.MyMemo)
			agent.Memo = message.Memo
		case protocol.SHELLREQ:
			// No need to check member "start"
			go shell.Start(agent.Conn, agent.ID, agent.UserOptions.Secret)
		case protocol.SHELLCOMMAND:
			message := fMessage.(*protocol.ShellCommand)
			shell.Input(message.Command)
		case protocol.LISTENREQ:
			//message := fMessage.(*protocol.ListenReq)
			//go handler.StartListen(message.Addr)
		case protocol.SSHREQ:
			message := fMessage.(*protocol.SSHReq)
			mySSH.Addr = message.Addr
			mySSH.Method = int(message.Method)
			mySSH.Username = message.Username
			mySSH.Password = message.Password
			mySSH.Certificate = message.Certificate
			go mySSH.Start(agent.Conn, agent.ID, agent.UserOptions.Secret)
		case protocol.SSHCOMMAND:
			message := fMessage.(*protocol.SSHCommand)
			mySSH.Input(message.Command)
		default:
			log.Println("[*]Unknown Message!")
		}
	}
}
