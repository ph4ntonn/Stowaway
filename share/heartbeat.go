package share

import (
	"net"
	"time"

	"Stowaway/utils"
)

/*-------------------------心跳包相关代码--------------------------*/

// SendHeartBeatControl 发送心跳包
func SendHeartBeatControl(controlConnToUpperNode *net.Conn, nodeid string, key []byte) {
	hbCommPack, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "HEARTBEAT", " ", " ", 0, nodeid, key, false) //这里nodeid写adminid，但实际上并不会被传递到admin端，会在HandleConnFromLowerNode中被解析
	for {
		time.Sleep(30 * time.Second)
		_, err := (*controlConnToUpperNode).Write(hbCommPack)
		if err != nil {
			continue
		}
	}
}
