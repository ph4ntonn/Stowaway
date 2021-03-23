/*
 * @Author: ph4ntom
 * @Date: 2021-03-23 14:20:35
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-03-23 18:30:41
 */
package handler

import "Stowaway/protocol"

func LetOffline(component *protocol.MessageComponent, route string, nodeID string) {
	sMessage := protocol.PrepareAndDecideWhichSProtoToLower(component.Conn, component.Secret, component.UUID)

	header := protocol.Header{
		Sender:      protocol.ADMIN_UUID,
		Accepter:    nodeID,
		MessageType: protocol.OFFLINE,
		RouteLen:    uint32(len([]byte(route))),
		Route:       route,
	}

	offlineMess := protocol.Offline{
		OK: 1,
	}

	protocol.ConstructMessage(sMessage, header, offlineMess)
	sMessage.SendMessage()
}
