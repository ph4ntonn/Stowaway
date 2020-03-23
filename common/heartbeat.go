package common

import (
	"net"
	"time"
)

/*-------------------------心跳包相关代码--------------------------*/
func SendHeartBeatControl(controlConnToUpperNode *net.Conn, nodeid uint32, key []byte) {
	hbcommpack, _ := ConstructPayload(nodeid-1, "COMMAND", "HEARTBEAT", " ", " ", 0, nodeid, key, false)
	for {
		time.Sleep(30 * time.Second)
		_, err := (*controlConnToUpperNode).Write(hbcommpack)
		if err != nil {
			continue
		}
	}
}
