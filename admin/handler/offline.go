/*
 * @Author: ph4ntom
 * @Date: 2021-03-23 14:20:35
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-03-26 16:48:39
 */
package handler

import (
	"Stowaway/global"
	"Stowaway/protocol"
)

func LetOffline(route string, uuid string) {
	sMessage := protocol.PrepareAndDecideWhichSProtoToLower(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)

	header := &protocol.Header{
		Sender:      protocol.ADMIN_UUID,
		Accepter:    uuid,
		MessageType: protocol.OFFLINE,
		RouteLen:    uint32(len([]byte(route))),
		Route:       route,
	}

	offlineMess := &protocol.Offline{
		OK: 1,
	}

	protocol.ConstructMessage(sMessage, header, offlineMess, false)
	sMessage.SendMessage()
}
