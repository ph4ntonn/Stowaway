package process

import (
	"log"
	"net"
	"os"
	"strings"

	"Stowaway/agent/handler"
	"Stowaway/agent/initial"
	"Stowaway/agent/manager"
	"Stowaway/global"
	"Stowaway/protocol"
	"Stowaway/share"
	"Stowaway/utils"
)

type Agent struct {
	UUID string
	Memo string

	options *initial.Options
	mgr     *manager.Manager

	childrenMessChan chan *ChildrenMess
}

type ChildrenMess struct {
	cHeader  *protocol.Header
	cMessage []byte
}

func NewAgent(options *initial.Options) *Agent {
	agent := new(Agent)
	agent.UUID = protocol.TEMP_UUID
	agent.childrenMessChan = make(chan *ChildrenMess, 5)
	agent.options = options
	return agent
}

func (agent *Agent) Run() {
	agent.sendMyInfo()
	// run manager
	agent.mgr = manager.NewManager(share.NewFile())
	go agent.mgr.Run()
	// run dispatchers to dispatch all kinds of message
	go handler.DispatchListenMess(agent.mgr, agent.options)
	go handler.DispatchConnectMess(agent.mgr)
	go handler.DispathSocksMess(agent.mgr)
	go handler.DispatchForwardMess(agent.mgr)
	go handler.DispatchBackwardMess(agent.mgr)
	go handler.DispatchFileMess(agent.mgr)
	go handler.DispatchSSHMess(agent.mgr)
	go handler.DispatchSSHTunnelMess(agent.mgr)
	go handler.DispatchShellMess(agent.mgr, agent.options)
	go DispatchOfflineMess(agent)
	// run dispatcher to dispatch children's message
	go agent.dispatchChildrenMess()
	// waiting for child
	go agent.waitingChild()
	// process data from upstream
	agent.handleDataFromUpstream()
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
		MemoLen:     uint64(len(agent.Memo)),
		Memo:        agent.Memo,
	}

	protocol.ConstructMessage(sMessage, header, myInfoMess, false)
	sMessage.SendMessage()
}

func (agent *Agent) handleDataFromUpstream() {
	rMessage := protocol.PrepareAndDecideWhichRProtoFromUpper(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)

	for {
		header, message, err := protocol.DestructMessage(rMessage)
		if err != nil {
			upstreamOffline(agent.mgr, agent.options)
			// Update rMessage
			rMessage = protocol.PrepareAndDecideWhichRProtoFromUpper(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)
			go agent.sendMyInfo()
			continue
		}

		if header.Accepter == agent.UUID {
			switch header.MessageType {
			case protocol.MYMEMO:
				message := message.(*protocol.MyMemo)
				agent.Memo = message.Memo // no need to pass this like all the message below,just change memo directly
			case protocol.SHELLREQ:
				fallthrough
			case protocol.SHELLCOMMAND:
				agent.mgr.ShellManager.ShellMessChan <- message
			case protocol.SSHREQ:
				fallthrough
			case protocol.SSHCOMMAND:
				agent.mgr.SSHManager.SSHMessChan <- message
			case protocol.SSHTUNNELREQ:
				agent.mgr.SSHTunnelManager.SSHTunnelMessChan <- message
			case protocol.FILESTATREQ:
				fallthrough
			case protocol.FILESTATRES:
				fallthrough
			case protocol.FILEDATA:
				fallthrough
			case protocol.FILEERR:
				fallthrough
			case protocol.FILEDOWNREQ:
				agent.mgr.FileManager.FileMessChan <- message
			case protocol.SOCKSSTART:
				fallthrough
			case protocol.SOCKSTCPDATA:
				fallthrough
			case protocol.SOCKSTCPFIN:
				fallthrough
			case protocol.UDPASSRES:
				fallthrough
			case protocol.SOCKSUDPDATA:
				agent.mgr.SocksManager.SocksMessChan <- message
			case protocol.FORWARDTEST:
				fallthrough
			case protocol.FORWARDSTART:
				fallthrough
			case protocol.FORWARDDATA:
				fallthrough
			case protocol.FORWARDFIN:
				agent.mgr.ForwardManager.ForwardMessChan <- message
			case protocol.BACKWARDTEST:
				fallthrough
			case protocol.BACKWARDSEQ:
				fallthrough
			case protocol.BACKWARDFIN:
				fallthrough
			case protocol.BACKWARDSTOP:
				fallthrough
			case protocol.BACKWARDDATA:
				agent.mgr.BackwardManager.BackwardMessChan <- message
			case protocol.CHILDUUIDRES:
				fallthrough
			case protocol.LISTENREQ:
				agent.mgr.ListenManager.ListenMessChan <- message
			case protocol.CONNECTSTART:
				agent.mgr.ConnectManager.ConnectMessChan <- message
			case protocol.UPSTREAMOFFLINE:
				fallthrough
			case protocol.UPSTREAMREONLINE:
				agent.mgr.OfflineManager.OfflineMessChan <- message
			case protocol.SHUTDOWN:
				os.Exit(0)
			default:
				log.Println("[*] Unknown Message!")
			}
		} else {
			agent.childrenMessChan <- &ChildrenMess{
				cHeader:  header,
				cMessage: message.([]byte),
			}
		}
	}
}

func (agent *Agent) dispatchChildrenMess() {
	for {
		childrenMess := <-agent.childrenMessChan

		childUUID := changeRoute(childrenMess.cHeader)

		task := &manager.ChildrenTask{
			Mode: manager.C_GETCONN,
			UUID: childUUID,
		}
		agent.mgr.ChildrenManager.TaskChan <- task
		result := <-agent.mgr.ChildrenManager.ResultChan
		if !result.OK {
			continue
		}

		sMessage := protocol.PrepareAndDecideWhichSProtoToLower(result.Conn, global.G_Component.Secret, global.G_Component.UUID)

		protocol.ConstructMessage(sMessage, childrenMess.cHeader, childrenMess.cMessage, true)
		sMessage.SendMessage()
	}
}

func (agent *Agent) waitingChild() {
	for {
		childInfo := <-agent.mgr.ChildrenManager.ChildComeChan
		go agent.handleDataFromDownstream(childInfo.Conn, childInfo.UUID)
	}
}

func (agent *Agent) handleDataFromDownstream(conn net.Conn, uuid string) {
	rMessage := protocol.PrepareAndDecideWhichRProtoFromLower(conn, global.G_Component.Secret, global.G_Component.UUID)

	for {
		header, message, err := protocol.DestructMessage(rMessage)
		if err != nil {
			downStreamOffline(agent.mgr, agent.options, uuid)
			return
		}

		sMessage := protocol.PrepareAndDecideWhichSProtoToUpper(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)

		protocol.ConstructMessage(sMessage, header, message, true)
		sMessage.SendMessage()
	}
}

func changeRoute(header *protocol.Header) string {
	route := header.Route
	// find next uuid
	routes := strings.Split(route, ":")
	if len(routes) == 1 {
		header.Route = ""
		header.RouteLen = 0
		return routes[0]
	}

	header.Route = strings.Join(routes[1:], ":")
	header.RouteLen = uint32(len(header.Route))
	return routes[0]

}
