package agent

import (
	"Stowaway/node"
	"time"
)

var Reconnsuccess = make(chan bool, 2)
var ExitSuccess = make(chan bool)
var err error

//not in use,add todo
func TryReconnect() {
	for {
		time.Sleep(10 * time.Second)

		ControlConnToAdmin, DataConnToAdmin, NODEID, err = node.StartNodeConn(Monitor, ListenPort, NODEID, AESKey)
		if err == nil {
			Reconnsuccess <- true
			Reconnsuccess <- true
		} else {
			continue
		}
		cmdResult <- []byte("")
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
