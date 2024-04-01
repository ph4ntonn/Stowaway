package handler

import (
	"crypto/tls"
	"errors"
	"net"
	"time"

	"Stowaway/agent/manager"
	"Stowaway/global"
	"Stowaway/protocol"
	"Stowaway/share"
	"Stowaway/share/transport"
)

type Connect struct {
	Addr string
}

func newConnect(addr string) *Connect {
	connect := new(Connect)
	connect.Addr = addr
	return connect
}

func (connect *Connect) start(mgr *manager.Manager) {
	var sUMessage, sLMessage, rMessage protocol.Message

	sUMessage = protocol.NewUpMsg(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)

	hiHeader := &protocol.Header{
		Sender:      protocol.ADMIN_UUID, // fake admin
		Accepter:    protocol.TEMP_UUID,
		MessageType: protocol.HI,
		RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))),
		Route:       protocol.TEMP_ROUTE,
	}

	// fake admin
	hiMess := &protocol.HIMess{
		GreetingLen: uint16(len("Shhh...")),
		Greeting:    "Shhh...",
		UUIDLen:     uint16(len(protocol.ADMIN_UUID)),
		UUID:        protocol.ADMIN_UUID,
		IsAdmin:     1,
		IsReconnect: 0,
	}

	doneHeader := &protocol.Header{
		Sender:      global.G_Component.UUID,
		Accepter:    protocol.ADMIN_UUID,
		MessageType: protocol.CONNECTDONE,
		RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))),
		Route:       protocol.TEMP_ROUTE,
	}

	doneSuccMess := &protocol.ConnectDone{
		OK: 1,
	}

	doneFailMess := &protocol.ConnectDone{
		OK: 0,
	}

	var (
		conn net.Conn
		err  error
	)

	defer func() {
		if err != nil {
			protocol.ConstructMessage(sUMessage, doneHeader, doneFailMess, false)
			sUMessage.SendMessage()
		}
	}()

	conn, err = net.DialTimeout("tcp", connect.Addr, 10*time.Second)

	if err != nil {
		return
	}

	if global.G_TLSEnable {
		var tlsConfig *tls.Config
		// Set domain as null since we are in the intranet
		tlsConfig, err = transport.NewClientTLSConfig("")
		if err != nil {
			conn.Close()
			return
		}
		conn = transport.WrapTLSClientConn(conn, tlsConfig)
	}
	// There's no need for the "domain" parameter between intranet nodes
	param := new(protocol.NegParam)
	param.Conn = conn
	proto := protocol.NewDownProto(param)
	proto.CNegotiate()

	if err = share.ActivePreAuth(conn); err != nil {
		return
	}

	sLMessage = protocol.NewDownMsg(conn, global.G_Component.Secret, protocol.ADMIN_UUID)

	protocol.ConstructMessage(sLMessage, hiHeader, hiMess, false)
	sLMessage.SendMessage()

	rMessage = protocol.NewDownMsg(conn, global.G_Component.Secret, protocol.ADMIN_UUID)
	fHeader, fMessage, err := protocol.DestructMessage(rMessage)

	if err != nil {
		conn.Close()
		return
	}

	var childUUID string

	if fHeader.MessageType == protocol.HI {
		mmess := fMessage.(*protocol.HIMess)
		if mmess.Greeting == "Keep slient" && mmess.IsAdmin == 0 {
			if mmess.IsReconnect == 0 {
				childIP := conn.RemoteAddr().String()

				cUUIDReqHeader := &protocol.Header{
					Sender:      global.G_Component.UUID,
					Accepter:    protocol.ADMIN_UUID,
					MessageType: protocol.CHILDUUIDREQ,
					RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))),
					Route:       protocol.TEMP_ROUTE,
				}

				cUUIDMess := &protocol.ChildUUIDReq{
					ParentUUIDLen: uint16(len(global.G_Component.UUID)),
					ParentUUID:    global.G_Component.UUID,
					IPLen:         uint16(len(childIP)),
					IP:            childIP,
				}

				protocol.ConstructMessage(sUMessage, cUUIDReqHeader, cUUIDMess, false)
				sUMessage.SendMessage()

				childUUID = <-mgr.ListenManager.ChildUUIDChan

				uuidHeader := &protocol.Header{
					Sender:      protocol.ADMIN_UUID, // Fake admin LOL
					Accepter:    protocol.TEMP_UUID,
					MessageType: protocol.UUID,
					RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))),
					Route:       protocol.TEMP_ROUTE,
				}

				uuidMess := &protocol.UUIDMess{
					UUIDLen: uint16(len(childUUID)),
					UUID:    childUUID,
				}

				protocol.ConstructMessage(sLMessage, uuidHeader, uuidMess, false)
				sLMessage.SendMessage()
			} else {
				reheader := &protocol.Header{
					Sender:      global.G_Component.UUID,
					Accepter:    protocol.ADMIN_UUID,
					MessageType: protocol.NODEREONLINE,
					RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))),
					Route:       protocol.TEMP_ROUTE,
				}

				reMess := &protocol.NodeReonline{
					ParentUUIDLen: uint16(len(global.G_Component.UUID)),
					ParentUUID:    global.G_Component.UUID,
					UUIDLen:       uint16(len(mmess.UUID)),
					UUID:          mmess.UUID,
					IPLen:         uint16(len(conn.RemoteAddr().String())),
					IP:            conn.RemoteAddr().String(),
				}

				protocol.ConstructMessage(sUMessage, reheader, reMess, false)
				sUMessage.SendMessage()

				childUUID = mmess.UUID
			}

			childrenTask := &manager.ChildrenTask{
				Mode: manager.C_NEWCHILD,
				UUID: childUUID,
				Conn: conn,
			}
			mgr.ChildrenManager.TaskChan <- childrenTask
			<-mgr.ChildrenManager.ResultChan

			mgr.ChildrenManager.ChildComeChan <- &manager.ChildInfo{UUID: childUUID, Conn: conn}

			protocol.ConstructMessage(sUMessage, doneHeader, doneSuccMess, false)
			sUMessage.SendMessage()

			return
		}
	}

	conn.Close()
	err = errors.New("node seems illegal")
}

func DispatchConnectMess(mgr *manager.Manager) {
	for {
		message := <-mgr.ConnectManager.ConnectMessChan

		switch mess := message.(type) {
		case *protocol.ConnectStart:
			connect := newConnect(mess.Addr)
			go connect.start(mgr)
		}
	}
}
