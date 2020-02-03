package common

import (
	"net"
	"time"
)

func SendHeartBeatControl(controlConnToUpperNode net.Conn, nodeid uint32, key []byte) {
	hbcommpack, _ := ConstructCommand("HEARTBEAT", "", nodeid, key)
	for {
		time.Sleep(5 * time.Second)
		_, err := controlConnToUpperNode.Write(hbcommpack)
		if err != nil {
			return
		}
	}
}

func SendHeartBeatData(dataConnForLowerNode net.Conn, nodeid uint32, key []byte) {
	hbdatapack, _ := ConstructDataResult(nodeid+1, 0, " ", "HEARTBEAT", " ", key, 0)
	for {
		time.Sleep(5 * time.Second)
		_, err := dataConnForLowerNode.Write(hbdatapack)
		if err != nil {
			return
		}
	}
}
