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
	"Stowaway/global"
	"Stowaway/protocol"
	"fmt"
)

func AddMemo(taskChan chan *topology.TopoTask, info []string, uuid string, route string) {
	var memo string

	for _, i := range info {
		memo = memo + " " + i
	}

	sMessage := protocol.PrepareAndDecideWhichSProtoToLower(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)

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

	protocol.ConstructMessage(sMessage, header, myMemoMess, false)
	sMessage.SendMessage()

	fmt.Print("\n[*]Memo added!")
}

func DelMemo(taskChan chan *topology.TopoTask, uuid string, route string) {
	topoTask := &topology.TopoTask{
		Mode: topology.UPDATEMEMO,
		UUID: uuid,
		Memo: "",
	}
	taskChan <- topoTask

	sMessage := protocol.PrepareAndDecideWhichSProtoToLower(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)

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

	protocol.ConstructMessage(sMessage, header, myMemoMess, false)
	sMessage.SendMessage()

	fmt.Print("\n[*]Memo deleted!")
}

func DispatchInfoMess(mgr *manager.Manager, topo *topology.Topology) {
	for {
		message := <-mgr.InfoManager.InfoMessChan

		switch message.(type) {
		case *protocol.MyInfo:
			mess := message.(*protocol.MyInfo)

			task := &topology.TopoTask{
				Mode:     topology.UPDATEDETAIL,
				UUID:     mess.UUID,
				UserName: mess.Username,
				HostName: mess.Hostname,
			}
			topo.TaskChan <- task
		}
	}
}
