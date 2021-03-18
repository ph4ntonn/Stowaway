/*
 * @Author: ph4ntom
 * @Date: 2021-03-18 18:05:46
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-03-18 18:45:07
 */
package handler

import (
	"Stowaway/protocol"
	"Stowaway/utils"
	"fmt"
)

func LetListen(sMessage protocol.Message, route string, nodeID string, addr string) {
	normalAddr, _, err := utils.CheckIPPort(addr)
	if err != nil {
		fmt.Printf("[*]Error: %s\n", err.Error())
		return
	}

	header := protocol.Header{
		Sender:      protocol.ADMIN_UUID,
		Accepter:    nodeID,
		MessageType: protocol.LISTENREQ,
		RouteLen:    uint32(len([]byte(route))),
		Route:       route,
	}

	listenReqMess := protocol.ListenReq{
		AddrLen: uint64(len(normalAddr)),
		Addr:    normalAddr,
	}

	protocol.ConstructMessage(sMessage, header, listenReqMess)
	sMessage.SendMessage()
}
