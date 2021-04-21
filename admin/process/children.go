package process

import (
	"Stowaway/admin/manager"
	"Stowaway/protocol"
)

func nodeOffline(mgr *manager.Manager, uuid string) {

}

func DispatchChildrenMess(mgr *manager.Manager) {
	for {
		message := <-mgr.ChildrenManager.ChildrenMessChan

		switch message.(type) {
		case *protocol.NodeOffline:
			mess := message.(*protocol.NodeOffline)
			nodeOffline(mgr, mess.UUID)
		}
	}
}
