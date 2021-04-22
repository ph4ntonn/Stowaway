package process

import (
	"Stowaway/admin/manager"
	"Stowaway/admin/topology"
	"Stowaway/protocol"
)

func nodeOffline(mgr *manager.Manager, topo *topology.Topology, uuid string) {
	// topoTask := &topology.TopoTask{
	// 	Mode: topology.DEACTIVENODE,
	// 	UUID: uuid,
	// }
	// topo.TaskChan <- topoTask
	// result := <-topo.ResultChan
	// allNodes := result.AllNodes

	// backwardTask := &manager.BackwardTask{
	// 	Mode: manager.B_GETDATACHAN_WITHOUTUUID,
	// 	Seq:  mess.Seq,
	// }
}

func DispatchChildrenMess(mgr *manager.Manager, topo *topology.Topology) {
	for {
		message := <-mgr.ChildrenManager.ChildrenMessChan

		switch message.(type) {
		case *protocol.NodeOffline:
			mess := message.(*protocol.NodeOffline)
			nodeOffline(mgr, topo, mess.UUID)
		}
	}
}
