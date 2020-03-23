package agent

import (
	"Stowaway/common"
	"fmt"
	"net"
	"strings"
)

var (
	ReflectConnMap             *common.Uint32ConnMap
	CurrentPortReflectListener []net.Listener
	ReflectNum                 uint32
)

func init() {
	ReflectNum = 0
	ReflectConnMap = common.NewUint32ConnMap()
}

/*-------------------------Port-reflect启动相关代码--------------------------*/
//检查是否能够监听
func TestReflect(portCombine string) {
	ports := strings.Split(portCombine, ":")
	reflectAddr := fmt.Sprintf("0.0.0.0:%s", ports[1])
	reflectListenerForClient, err := net.Listen("tcp", reflectAddr)
	if err != nil {
		defer reflectListenerForClient.Close()
		respCommand, _ := common.ConstructPayload(0, "COMMAND", "REFLECTFAIL", " ", " ", 0, NODEID, AESKey, false)
		ProxyChanToUpperNode <- respCommand
		return
	} else {
		respCommand, _ := common.ConstructPayload(0, "COMMAND", "REFLECTOK", " ", " ", 0, NODEID, AESKey, false)
		ProxyChanToUpperNode <- respCommand
	}

	CurrentPortReflectListener = append(CurrentPortReflectListener, reflectListenerForClient)

	for {
		conn, err := reflectListenerForClient.Accept()
		if err != nil {
			return
		} else {
			respCommand, _ := common.ConstructPayload(0, "DATA", "REFLECT", " ", ports[0], ReflectNum, NODEID, AESKey, false)
			ProxyChanToUpperNode <- respCommand
		}
		ReflectConnMap.Lock()
		ReflectConnMap.Payload[ReflectNum] = conn
		ReflectConnMap.Unlock()
		ReflectConnMap.RLock()
		go HandleReflectPort(ReflectConnMap.Payload[ReflectNum], ReflectNum, NODEID)
		ReflectConnMap.RUnlock()
		ReflectNum++
	}
}

//处理传入连接
func HandleReflectPort(reflectconn net.Conn, num uint32, nodeid uint32) {
	buffer := make([]byte, 10240)
	for {
		len, err := reflectconn.Read(buffer)
		if err != nil {
			finMessage, _ := common.ConstructPayload(0, "DATA", "REFLECT", " ", " ", num, NODEID, AESKey, false)
			ProxyChanToUpperNode <- finMessage
			return
		} else {
			respData, _ := common.ConstructPayload(0, "DATA", "REFLECTDATA", " ", string(buffer[:len]), num, NODEID, AESKey, false)
			ProxyChanToUpperNode <- respData
		}
	}
}
