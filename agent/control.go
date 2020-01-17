package agent

import (
	"Stowaway/node"
	"time"
)

var (
	Reconnsuccess = make(chan bool, 2)
	ExitSuccess   = make(chan bool)
)

//not in use,add todo
func TryReconnect() {
	var err error
	for {
		time.Sleep(10 * time.Second)

		ControlConnToAdmin, DataConnToAdmin, NODEID, err = node.StartNodeConn(Monitor, ListenPort, NODEID, AESKey)
		if err == nil {
			Reconnsuccess <- true
			Reconnsuccess <- true
		} else {
			continue
		}
		CmdResult <- []byte("")
		<-ExitSuccess
		go HandleStartNodeConn(ControlConnToAdmin, DataConnToAdmin, Monitor, NODEID)
		go ProxyLowerNodeCommToUpperNode(ControlConnToAdmin, LowerNodeCommChan)
		select {
		case <-Reconnsuccess:
			return
		default:
		}
	}
}
