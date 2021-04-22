package process

import (
	"Stowaway/agent/initial"
	"Stowaway/agent/manager"
	"Stowaway/global"
	"Stowaway/protocol"
	"os"
)

func upstreamOffline(mgr *manager.Manager, options *initial.Options) {
	if options.Mode == initial.IPTABLES_REUSE_PASSIVE {
		initial.DeletePortReuseRules(options.Listen, options.ReusePort)
	}
	os.Exit(0)

	if options.Reconnect == 0 && options.Listen == "" { // if no reconnecting || not passive,then exit immediately
		os.Exit(0)
	}

	broadcastOfflineMess(mgr)

	if options.Reconnect != 0 {
		if !options.RhostReuse { // upstream is not reusing port

		}
	}
}

func broadcastOfflineMess(mgr *manager.Manager) {
	childrenTask := &manager.ChildrenTask{
		Mode: manager.C_GETCHILDREN,
	}

	mgr.ChildrenManager.TaskChan <- childrenTask
	result := <-mgr.ChildrenManager.ResultChan

	for _, childUUID := range result.Children {
		task := &manager.ChildrenTask{
			Mode: manager.C_GETCONN,
			UUID: childUUID,
		}
		mgr.ChildrenManager.TaskChan <- task
		result = <-mgr.ChildrenManager.ResultChan

	}
}

func downStreamOffline(mgr *manager.Manager, options *initial.Options, uuid string) {
	childrenTask := &manager.ChildrenTask{ // del the child
		Mode: manager.C_DELCHILD,
		UUID: uuid,
	}

	mgr.ChildrenManager.TaskChan <- childrenTask
	<-mgr.ChildrenManager.ResultChan

	sMessage := protocol.PrepareAndDecideWhichSProtoToUpper(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)

	header := &protocol.Header{
		Sender:      global.G_Component.UUID,
		Accepter:    protocol.ADMIN_UUID,
		MessageType: protocol.NODEOFFLINE,
		RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))),
		Route:       protocol.TEMP_ROUTE,
	}

	offlineMess := &protocol.NodeOffline{
		UUIDLen: uint16(len(uuid)),
		UUID:    uuid,
	}

	protocol.ConstructMessage(sMessage, header, offlineMess, false)
	sMessage.SendMessage()
}
