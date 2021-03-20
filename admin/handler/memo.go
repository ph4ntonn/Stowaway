/*
 * @Author: ph4ntom
 * @Date: 2021-03-18 16:59:46
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-03-20 13:35:53
 */
package handler

import (
	"Stowaway/admin/topology"
	"Stowaway/protocol"
	"fmt"
)

func AddMemo(component *protocol.MessageComponent, taskChan chan *topology.TopoTask, info []string, nodeID string, route string) {
	var memo string

	for _, i := range info {
		memo = memo + " " + i
	}

	sMessage := protocol.PrepareAndDecideWhichSProto(component.Conn, component.Secret, component.ID)

	topoTask := &topology.TopoTask{
		Mode: topology.UPDATEMEMO,
		ID:   nodeID,
		Memo: memo,
	}
	taskChan <- topoTask

	header := protocol.Header{
		Sender:      protocol.ADMIN_UUID,
		Accepter:    nodeID,
		MessageType: protocol.MYMEMO,
		RouteLen:    uint32(len([]byte(route))),
		Route:       route,
	}

	myMemoMess := protocol.MyMemo{
		MemoLen: uint64(len(memo)),
		Memo:    memo,
	}

	protocol.ConstructMessage(sMessage, header, myMemoMess)
	sMessage.SendMessage()

	fmt.Print("\n[*]Memo added!")
}

func DelMemo(component *protocol.MessageComponent, taskChan chan *topology.TopoTask, nodeID string, route string) {
	topoTask := &topology.TopoTask{
		Mode: topology.UPDATEMEMO,
		ID:   nodeID,
		Memo: "",
	}
	taskChan <- topoTask

	sMessage := protocol.PrepareAndDecideWhichSProto(component.Conn, component.Secret, component.ID)

	header := protocol.Header{
		Sender:      protocol.ADMIN_UUID,
		Accepter:    nodeID,
		MessageType: protocol.MYMEMO,
		RouteLen:    uint32(len([]byte(route))),
		Route:       route,
	}

	myMemoMess := protocol.MyMemo{
		MemoLen: uint64(len("")),
		Memo:    "",
	}

	protocol.ConstructMessage(sMessage, header, myMemoMess)
	sMessage.SendMessage()

	fmt.Print("\n[*]Memo deleted!")
}
