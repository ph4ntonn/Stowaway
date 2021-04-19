/*
 * @Author: ph4ntom
 * @Date: 2021-03-18 18:33:57
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-03-18 18:55:47
 */
package handler

import (
	"Stowaway/agent/manager"
	"Stowaway/global"
	"Stowaway/protocol"
	"Stowaway/share"
	"net"
)

type Listen struct {
	addr string
}

func newListen(addr string) *Listen {
	listen := new(Listen)
	listen.addr = addr
	return listen
}

func (listen *Listen) start(mgr *manager.Manager) {
	sUMessage := protocol.PrepareAndDecideWhichSProtoToUpper(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)

	resHeader := &protocol.Header{
		Sender:      global.G_Component.UUID,
		Accepter:    protocol.ADMIN_UUID,
		MessageType: protocol.LISTENRES,
		RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))), // No need to set route when agent send mess to admin
		Route:       protocol.TEMP_ROUTE,
	}

	succMess := &protocol.ListenRes{
		OK: 1,
	}

	failMess := &protocol.ListenRes{
		OK: 0,
	}

	listener, err := net.Listen("tcp", listen.addr)
	if err != nil {
		protocol.ConstructMessage(sUMessage, resHeader, failMess, false)
		sUMessage.SendMessage()
		return
	}

	defer listener.Close()

	protocol.ConstructMessage(sUMessage, resHeader, succMess, false)
	sUMessage.SendMessage()

	for {
		conn, err := listener.Accept()
		if err != nil {
			conn.Close()
			continue
		}

		if err := share.PassivePreAuth(conn, global.G_Component.Secret); err != nil {
			conn.Close()
			continue
		}

		rMessage := protocol.PrepareAndDecideWhichRProtoFromLower(conn, global.G_Component.Secret, protocol.ADMIN_UUID) //fake admin
		fHeader, fMessage, err := protocol.DestructMessage(rMessage)

		if err != nil {
			conn.Close()
			continue
		}

		if fHeader.MessageType == protocol.HI {
			mmess := fMessage.(*protocol.HIMess)
			if mmess.Greeting == "Shhh..." && mmess.IsAdmin == 0 {
				var childUUID string

				sLMessage := protocol.PrepareAndDecideWhichSProtoToLower(conn, global.G_Component.Secret, protocol.ADMIN_UUID) //fake admin

				hiMess := &protocol.HIMess{
					GreetingLen: uint16(len("Keep slient")),
					Greeting:    "Keep slient",
					UUIDLen:     uint16(len(protocol.ADMIN_UUID)),
					UUID:        protocol.ADMIN_UUID,
					IsAdmin:     1,
					IsReconnect: 0,
				}

				hiHeader := &protocol.Header{
					Sender:      protocol.ADMIN_UUID,
					Accepter:    protocol.TEMP_UUID,
					MessageType: protocol.HI,
					RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))),
					Route:       protocol.TEMP_ROUTE,
				}

				protocol.ConstructMessage(sLMessage, hiHeader, hiMess, false)
				sLMessage.SendMessage()

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

					// Problem:If two listen port ready,and at the same time,two agent come,then maybe it would cause wrong uuid dispatching,but i think such coincidence hardly happen
					// (Fine..I just don't want to maintain status Orz
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

				}

				childrenTask := &manager.ChildrenTask{
					Mode: manager.C_NEWCHILD,
					UUID: childUUID,
					Conn: conn,
					Addr: listen.addr,
				}
				mgr.ChildrenManager.TaskChan <- childrenTask
				<-mgr.ChildrenManager.ResultChan

				mgr.ChildrenManager.ChildComeChan <- conn

				return
			}
		}

		conn.Close()
	}
}

func DispatchListenMess(mgr *manager.Manager) {
	for {
		message := <-mgr.ListenManager.ListenMessChan

		switch message.(type) {
		case *protocol.ListenReq:
			mess := message.(*protocol.ListenReq)
			listen := newListen(mess.Addr)
			go listen.start(mgr)
		case *protocol.ChildUUIDRes:
			mess := message.(*protocol.ChildUUIDRes)
			mgr.ListenManager.ChildUUIDChan <- mess.UUID
		}
	}
}
