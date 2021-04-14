/*
 * @Author: ph4ntom
 * @Date: 2021-03-18 18:05:46
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-03-30 16:40:07
 */
package handler

import (
	"Stowaway/global"
	"Stowaway/protocol"
	"Stowaway/utils"
	"fmt"
)

func LetListen(route string, uuid string, addr string) {
	normalAddr, _, err := utils.CheckIPPort(addr)
	if err != nil {
		fmt.Printf("[*]Error: %s\n", err.Error())
		return
	}

	sMessage := protocol.PrepareAndDecideWhichSProtoToLower(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)

	header := &protocol.Header{
		Sender:      protocol.ADMIN_UUID,
		Accepter:    uuid,
		MessageType: protocol.LISTENREQ,
		RouteLen:    uint32(len([]byte(route))),
		Route:       route,
	}

	listenReqMess := &protocol.ListenReq{
		AddrLen: uint64(len(normalAddr)),
		Addr:    normalAddr,
	}

	protocol.ConstructMessage(sMessage, header, listenReqMess)
	sMessage.SendMessage()
}
