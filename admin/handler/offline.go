package handler

import (
	"Stowaway/global"
	"Stowaway/protocol"
)

func LetShutdown(route string, uuid string) {
	sMessage := protocol.NewDownMsg(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)

	header := &protocol.Header{
		Sender:      protocol.ADMIN_UUID,
		Accepter:    uuid,
		MessageType: protocol.SHUTDOWN,
		RouteLen:    uint32(len([]byte(route))),
		Route:       route,
	}

	shutdownMess := &protocol.Shutdown{
		OK: 1,
	}

	protocol.ConstructMessage(sMessage, header, shutdownMess, false)
	sMessage.SendMessage()
}
