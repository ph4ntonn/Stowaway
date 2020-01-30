package agent

import (
	"Stowaway/common"
	"Stowaway/node"
	"fmt"
	"net"
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
			go node.SendHeartBeat(ControlConnToAdmin, DataConnToAdmin, NODEID, AESKey)
			return
		}
	}
}

func SendHeartBeatControl(controlConnToUpperNode net.Conn, nodeid uint32, key []byte) {
	hbcommpack, _ := common.ConstructCommand("HEARTBEAT", "", nodeid, key)
	for {
		time.Sleep(5 * time.Second)
		_, err := controlConnToUpperNode.Write(hbcommpack)
		if err != nil {
			return
		}
	}
}

func SendHeartBeatData(dataConnForLowerNode net.Conn, nodeid uint32, key []byte) {
	hbdatapack, _ := common.ConstructDataResult(nodeid+1, 0, " ", "HEARTBEAT", " ", AESKey, 0)
	for {
		time.Sleep(5 * time.Second)
		_, err := dataConnForLowerNode.Write(hbdatapack)
		if err != nil {
			return
		}
	}
}
