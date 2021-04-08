/*
 * @Author: ph4ntom
 * @Date: 2021-03-18 16:59:46
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-03-30 16:40:15
 */
package handler

import (
	"Stowaway/admin/manager"
	"Stowaway/admin/topology"
	"Stowaway/protocol"
	"fmt"
)

func AddMemo(component *protocol.MessageComponent, taskChan chan *topology.TopoTask, info []string, uuid string, route string) {
	var memo string

	for _, i := range info {
		memo = memo + " " + i
	}

	sMessage := protocol.PrepareAndDecideWhichSProtoToLower(component.Conn, component.Secret, component.UUID)

	topoTask := &topology.TopoTask{
		Mode: topology.UPDATEMEMO,
		UUID: uuid,
		Memo: memo,
	}
	taskChan <- topoTask

	header := &protocol.Header{
		Sender:      protocol.ADMIN_UUID,
		Accepter:    uuid,
		MessageType: protocol.MYMEMO,
		RouteLen:    uint32(len([]byte(route))),
		Route:       route,
	}

	myMemoMess := &protocol.MyMemo{
		MemoLen: uint64(len(memo)),
		Memo:    memo,
	}

	protocol.ConstructMessage(sMessage, header, myMemoMess)
	sMessage.SendMessage()

	fmt.Print("\n[*]Memo added!")
}

func DelMemo(component *protocol.MessageComponent, taskChan chan *topology.TopoTask, uuid string, route string) {
	topoTask := &topology.TopoTask{
		Mode: topology.UPDATEMEMO,
		UUID: uuid,
		Memo: "",
	}
	taskChan <- topoTask

	sMessage := protocol.PrepareAndDecideWhichSProtoToLower(component.Conn, component.Secret, component.UUID)

	header := &protocol.Header{
		Sender:      protocol.ADMIN_UUID,
		Accepter:    uuid,
		MessageType: protocol.MYMEMO,
		RouteLen:    uint32(len([]byte(route))),
		Route:       route,
	}

	myMemoMess := &protocol.MyMemo{
		MemoLen: uint64(len("")),
		Memo:    "",
	}

	protocol.ConstructMessage(sMessage, header, myMemoMess)
	sMessage.SendMessage()

	fmt.Print("\n[*]Memo deleted!")
}

func DispatchInfoMess(mgr *manager.Manager, topo *topology.Topology) {
	for {
		message := <-mgr.InfoManager.InfoMessChan

		switch message.Mess.(type) {
		case *protocol.MyInfo:
			mess := message.Mess.(*protocol.MyInfo)

			task := &topology.TopoTask{
				Mode:     topology.UPDATEDETAIL,
				UUID:     message.UUID,
				UserName: mess.Username,
				HostName: mess.Hostname,
			}
			topo.TaskChan <- task
		}
	}
}
