/*
 * @Author: ph4ntom
 * @Date: 2021-03-16 17:46:15
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-03-16 19:20:11
 */
package handler

import "Stowaway/protocol"

func LetShellStart(sMessage protocol.Message, route string, nodeID string) {
	header := protocol.Header{
		Sender:      protocol.ADMIN_UUID,
		Accepter:    nodeID,
		MessageType: protocol.SHELLREQ,
		RouteLen:    uint32(len([]byte(route))),
		Route:       route,
	}

	shellReqMess := protocol.ShellReq{
		Start: 1,
	}
	protocol.ConstructMessage(sMessage, header, shellReqMess)
	sMessage.SendMessage()
}
