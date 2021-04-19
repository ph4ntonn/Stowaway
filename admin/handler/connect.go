package handler

import (
	"Stowaway/admin/manager"
	"Stowaway/global"
	"Stowaway/protocol"
	"Stowaway/utils"
	"fmt"
)

type Connect struct {
	IsReuse uint16
	Addr    string
}

func NewConnect(addr string, isReuse uint16) *Connect {
	connect := new(Connect)
	connect.IsReuse = isReuse
	connect.Addr = addr
	return connect
}

func (connect *Connect) LetConnect(mgr *manager.Manager, route string, uuid string) error {
	normalAddr, _, err := utils.CheckIPPort(connect.Addr)
	if err != nil {
		return err
	}

	sMessage := protocol.PrepareAndDecideWhichSProtoToLower(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)

	header := &protocol.Header{
		Sender:      protocol.ADMIN_UUID,
		Accepter:    uuid,
		MessageType: protocol.CONNECTSTART,
		RouteLen:    uint32(len([]byte(route))),
		Route:       route,
	}

	connMess := &protocol.ConnectStart{
		IsReuse: connect.IsReuse,
		AddrLen: uint16(len([]byte(normalAddr))),
		Addr:    normalAddr,
	}

	protocol.ConstructMessage(sMessage, header, connMess, false)
	sMessage.SendMessage()

	if ok := <-mgr.ConnectManager.ConnectReady; !ok {
		fmt.Printf("\r\n[*]Cannot connect to node %s", connect.Addr)
	}

	return nil
}

func DispatchConnectMess(mgr *manager.Manager) {
	for {
		message := <-mgr.ConnectManager.ConnectMessChan

		switch message.(type) {
		case *protocol.ConnectDone:
			mess := message.(*protocol.ConnectDone)
			if mess.OK == 1 {
				mgr.ConnectManager.ConnectReady <- true
			} else {
				mgr.ConnectManager.ConnectReady <- false
			}
		}
	}
}
