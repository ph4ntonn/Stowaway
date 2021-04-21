package handler

import (
	"Stowaway/agent/manager"
	"Stowaway/global"
	"Stowaway/protocol"
	"Stowaway/share"
	"errors"
	"net"
	"time"
)

type Connect struct {
	IsReuse uint16
	Addr    string
}

func newConnect(addr string, isReuse uint16) *Connect {
	connect := new(Connect)
	connect.IsReuse = isReuse
	connect.Addr = addr
	return connect
}

func (connect *Connect) start(mgr *manager.Manager) {
	var sUMessage, sLMessage, rMessage protocol.Message

	sUMessage = protocol.PrepareAndDecideWhichSProtoToUpper(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)

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

	if err = share.ActivePreAuth(conn, global.G_Component.Secret); err != nil {
		return
	}

	sLMessage = protocol.PrepareAndDecideWhichSProtoToLower(conn, global.G_Component.Secret, protocol.ADMIN_UUID)

	protocol.ConstructMessage(sLMessage, hiHeader, hiMess, false)
	sLMessage.SendMessage()

	rMessage = protocol.PrepareAndDecideWhichRProtoFromLower(conn, global.G_Component.Secret, protocol.ADMIN_UUID)
	fHeader, fMessage, err := protocol.DestructMessage(rMessage)

	if err != nil {
		conn.Close()
		return
	}

	if fHeader.MessageType == protocol.HI {
		mmess := fMessage.(*protocol.HIMess)
		if mmess.Greeting == "Keep slient" && mmess.IsAdmin == 0 {
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

			childUUID := <-mgr.ListenManager.ChildUUIDChan

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
	err = errors.New("Node seems illegal!")
	return
}

func DispatchConnectMess(mgr *manager.Manager) {
	for {
		message := <-mgr.ConnectManager.ConnectMessChan

		switch message.(type) {
		case *protocol.ConnectStart:
			mess := message.(*protocol.ConnectStart)
			connect := newConnect(mess.Addr, mess.IsReuse)
			go connect.start(mgr)
		}
	}
}
