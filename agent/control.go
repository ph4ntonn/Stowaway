package agent

import (
	"Stowaway/common"
	"Stowaway/node"
	"fmt"
	"strconv"
	"time"
)

func TryReconnect(gap string) {
	lag, _ := strconv.Atoi(gap)
	for {
		time.Sleep(time.Duration(lag) * time.Second)

		controlConnToAdmin, dataConnToAdmin, _, err := node.StartNodeConn(Monitor, ListenPort, NODEID, AESKey)
		if err != nil {
			fmt.Println("[*]Admin seems still down")
		} else {
			fmt.Println("[*]Admin up! Reconnect successful!")
			ControlConnToAdmin = controlConnToAdmin
			DataConnToAdmin = dataConnToAdmin
			go common.SendHeartBeatControl(ControlConnToAdmin, NODEID, AESKey)
			return
		}
	}
}

func TryWaitConn() {

}
