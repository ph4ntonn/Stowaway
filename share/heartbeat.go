package share

import (
	"Stowaway/utils"
	"net"
	"time"
)

/*-------------------------心跳包相关代码--------------------------*/
func SendHeartBeatControl(controlConnToUpperNode *net.Conn, nodeid string, key []byte) {
	hbcommpack, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "HEARTBEAT", " ", " ", 0, nodeid, key, false) //这里nodeid写adminid，但实际上并不会被传递到admin端，会在HandleConnFromLowerNode中被解析
	for {
		time.Sleep(30 * time.Second)
		_, err := (*controlConnToUpperNode).Write(hbcommpack)
		if err != nil {
			continue
		}
	}
}
