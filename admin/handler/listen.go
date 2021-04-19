/*
 * @Author: ph4ntom
 * @Date: 2021-03-18 18:05:46
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-03-30 16:40:07
 */
package handler

import (
	"Stowaway/admin/manager"
	"Stowaway/admin/topology"
	"Stowaway/global"
	"Stowaway/protocol"
	"Stowaway/utils"
	"fmt"
)

func LetListen(mgr *manager.Manager, route, uuid, addr string) error {
	normalAddr, _, err := utils.CheckIPPort(addr)
	if err != nil {
		return err
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

	protocol.ConstructMessage(sMessage, header, listenReqMess, false)
	sMessage.SendMessage()

	if <-mgr.ListenManager.ListenReady {
		fmt.Printf("\r\n[*]Node is listening on %s", addr)
	} else {
		fmt.Printf("\r\n[*]Node cannot listen on %s", addr)
	}

	return nil
}

// this function is SPECIAL,handling childuuidreq from both "listen" and "node reuse" and "connect" condition
func dispatchChildUUID(mgr *manager.Manager, topo *topology.Topology, parentUUID, ip string) {
	uuid := utils.GenerateUUID()
	node := topology.NewNode(uuid, ip)
	topoTask := &topology.TopoTask{
		Mode:    topology.ADDNODE,
		Target:  node,
		UUID:    parentUUID,
		IsFirst: false,
	}
	topo.TaskChan <- topoTask
	topoResult := <-topo.ResultChan
	childIDNum := topoResult.IDNum

	topoTask = &topology.TopoTask{
		Mode: topology.CALCULATE,
	}
	topo.TaskChan <- topoTask
	<-topo.ResultChan

	topoTask = &topology.TopoTask{
		Mode: topology.GETROUTE,
		UUID: parentUUID,
	}
	topo.TaskChan <- topoTask
	topoResult = <-topo.ResultChan
	route := topoResult.Route

	sMessage := protocol.PrepareAndDecideWhichSProtoToLower(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)

	header := &protocol.Header{
		Sender:      protocol.ADMIN_UUID,
		Accepter:    parentUUID,
		MessageType: protocol.CHILDUUIDRES,
		RouteLen:    uint32(len([]byte(route))),
		Route:       route,
	}

	cUUIDResMess := &protocol.ChildUUIDRes{
		UUIDLen: uint16(len(uuid)),
		UUID:    uuid,
	}

	protocol.ConstructMessage(sMessage, header, cUUIDResMess, false)
	sMessage.SendMessage()

	fmt.Printf("\r\n[*]New node come! Node id is %d\n", childIDNum)
}

func DispatchListenMess(mgr *manager.Manager, topo *topology.Topology) {
	for {
		message := <-mgr.ListenManager.ListenMessChan

		switch message.(type) {
		case *protocol.ListenRes:
			mess := message.(*protocol.ListenRes)
			if mess.OK == 1 {
				mgr.ListenManager.ListenReady <- true
			} else {
				mgr.ListenManager.ListenReady <- false
			}
		case *protocol.ChildUUIDReq:
			mess := message.(*protocol.ChildUUIDReq)
			go dispatchChildUUID(mgr, topo, mess.ParentUUID, mess.IP)
		}
	}
}
