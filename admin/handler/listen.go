package handler

import (
	"Stowaway/admin/manager"
	"Stowaway/admin/printer"
	"Stowaway/admin/topology"
	"Stowaway/global"
	"Stowaway/protocol"
	"Stowaway/utils"
)

const (
	NORMAL = iota
	IPTABLES
	SOREUSE
)

type Listen struct {
	Method int
	Addr   string
}

func NewListen() *Listen {
	return new(Listen)
}

func (listen *Listen) LetListen(mgr *manager.Manager, route, uuid string) error {
	var finalAddr string

	if listen.Method == NORMAL {
		var err error
		finalAddr, _, err = utils.CheckIPPort(listen.Addr)
		if err != nil {
			return err
		}
	}

	sMessage := protocol.NewDownMsg(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)

	header := &protocol.Header{
		Sender:      protocol.ADMIN_UUID,
		Accepter:    uuid,
		MessageType: protocol.LISTENREQ,
		RouteLen:    uint32(len([]byte(route))),
		Route:       route,
	}

	listenReqMess := &protocol.ListenReq{
		Method:  uint16(listen.Method),
		AddrLen: uint64(len(finalAddr)),
		Addr:    finalAddr,
	}

	protocol.ConstructMessage(sMessage, header, listenReqMess, false)
	sMessage.SendMessage()

	if <-mgr.ListenManager.ListenReady {
		if listen.Method == NORMAL {
			printer.Success("\r\n[*] Node is listening on %s", listen.Addr)
		} else {
			printer.Success("\r\n[*] Node is reusing port successfully,just waiting for child....")
		}
	} else {
		if listen.Method == NORMAL {
			printer.Success("\r\n[*] Node cannot listen on %s", listen.Addr)
		} else {
			printer.Success("\r\n[*] Node cannot reusing port,plz check if node is initialed via resusing!")
		}
	}

	return nil
}

// this function is SPECIAL,handling childuuidreq from both "listen" && "node reuse" && "connect" && "sshtunnel" condition
func dispatchChildUUID(mgr *manager.Manager, topo *topology.Topology, parentUUID, ip string) {
	uuid := utils.GenerateUUID()
	node := topology.NewNode(uuid, ip)
	topoTask := &topology.TopoTask{
		Mode:       topology.ADDNODE,
		Target:     node,
		ParentUUID: parentUUID,
		IsFirst:    false,
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

	sMessage := protocol.NewDownMsg(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)

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

	printer.Success("\r\n[*] New node online! Node id is %d\r\n", childIDNum)
}

func DispatchListenMess(mgr *manager.Manager, topo *topology.Topology) {
	for {
		message := <-mgr.ListenManager.ListenMessChan

		switch mess := message.(type) {
		case *protocol.ListenRes:
			if mess.OK == 1 {
				mgr.ListenManager.ListenReady <- true
			} else {
				mgr.ListenManager.ListenReady <- false
			}
		case *protocol.ChildUUIDReq:
			go dispatchChildUUID(mgr, topo, mess.ParentUUID, mess.IP)
		}
	}
}
