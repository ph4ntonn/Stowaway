package process

import (
	"Stowaway/admin/manager"
	"Stowaway/admin/topology"
	"Stowaway/protocol"
)

func nodeOffline(mgr *manager.Manager, topo *topology.Topology, uuid string) {
	topoTask := &topology.TopoTask{
		Mode: topology.DELNODE,
		UUID: uuid,
	}
	topo.TaskChan <- topoTask
	result := <-topo.ResultChan
	allNodes := result.AllNodes

	for _, nodeUUID := range allNodes {
		backwardTask := &manager.BackwardTask{
			Mode: manager.B_FORCESHUTDOWN,
			UUID: nodeUUID,
		}
		mgr.BackwardManager.TaskChan <- backwardTask
		<-mgr.BackwardManager.ResultChan

		forwardTask := &manager.ForwardTask{
			Mode: manager.F_FORCESHUTDOWN,
			UUID: nodeUUID,
		}
		mgr.ForwardManager.TaskChan <- forwardTask
		<-mgr.ForwardManager.ResultChan

		socksTask := &manager.SocksTask{
			Mode: manager.S_FORCESHUTDOWN,
			UUID: nodeUUID,
		}
		mgr.SocksManager.TaskChan <- socksTask
		<-mgr.SocksManager.ResultChan
	}

	topoTask = &topology.TopoTask{
		Mode: topology.CALCULATE,
	}
	topo.TaskChan <- topoTask
	<-topo.ResultChan
}

func nodeReonline(mgr *manager.Manager, topo *topology.Topology, mess *protocol.NodeReonline) {
	node := topology.NewNode(mess.UUID, mess.IP)

	topoTask := &topology.TopoTask{
		Mode:       topology.REONLINENODE,
		Target:     node,
		ParentUUID: mess.ParentUUID,
		IsFirst:    false,
	}
	topo.TaskChan <- topoTask
	<-topo.ResultChan

	topoTask = &topology.TopoTask{
		Mode: topology.CALCULATE,
	}
	topo.TaskChan <- topoTask
	<-topo.ResultChan
}

func DispatchChildrenMess(mgr *manager.Manager, topo *topology.Topology) {
	for {
		message := <-mgr.ChildrenManager.ChildrenMessChan

		switch message.(type) {
		case *protocol.NodeOffline:
			mess := message.(*protocol.NodeOffline)
			nodeOffline(mgr, topo, mess.UUID)
		case *protocol.NodeReonline:
			mess := message.(*protocol.NodeReonline)
			nodeReonline(mgr, topo, mess)
		}
	}
}
