package process

import (
	"Stowaway/agent/initial"
	"Stowaway/agent/manager"
	"os"
)

func UpstreamOffline(mgr *manager.Manager, options *initial.Options) {
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
