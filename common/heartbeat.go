package common

import (
	"net"
	"time"
)

/*-------------------------心跳包相关代码--------------------------*/
func SendHeartBeatControl(controlConnToUpperNode *net.Conn, nodeid uint32, key []byte) {
	hbcommpack, _ := ConstructPayload(0, "", "COMMAND", "HEARTBEAT", " ", " ", 0, nodeid, key, false) //这里nodeid写0，但实际上并不会被传递到admin端，会在HandleConnFromLowerNode中被解析，属于特例
	for {
		time.Sleep(30 * time.Second)
		_, err := (*controlConnToUpperNode).Write(hbcommpack)
		if err != nil {
			continue
		}
	}
}
