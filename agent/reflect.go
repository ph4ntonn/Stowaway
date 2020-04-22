package agent

import (
	"Stowaway/utils"
	"fmt"
	"net"
	"strings"
)

var (
	ReflectConnMap             *utils.Uint32ConnMap
	CurrentPortReflectListener []net.Listener
	ReflectStatus              *utils.ReflectStatus
)

func init() {
	ReflectStatus = utils.NewReflectStatus()
	ReflectConnMap = utils.NewUint32ConnMap()
}

/*-------------------------Port-reflect启动相关代码--------------------------*/
//检查是否能够监听
func TestReflect(portCombine string) {
	var num uint32
	ports := strings.Split(portCombine, ":")
	reflectAddr := fmt.Sprintf("0.0.0.0:%s", ports[1])
	reflectListenerForClient, err := net.Listen("tcp", reflectAddr)
	if err != nil {
		respCommand, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "REFLECTFAIL", " ", " ", 0, AgentStatus.Nodeid, AgentStatus.AESKey, false)
		ProxyChan.ProxyChanToUpperNode <- respCommand
		return
	} else {
		defer reflectListenerForClient.Close()
		respCommand, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "REFLECTOK", " ", " ", 0, AgentStatus.Nodeid, AgentStatus.AESKey, false)
		ProxyChan.ProxyChanToUpperNode <- respCommand
	}

	CurrentPortReflectListener = append(CurrentPortReflectListener, reflectListenerForClient)

	for {
		conn, err := reflectListenerForClient.Accept()
		if err != nil {
			return
		} else {
			respCommand, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "GETREFLECTNUM", " ", ports[0], 0, AgentStatus.Nodeid, AgentStatus.AESKey, false)
			ProxyChan.ProxyChanToUpperNode <- respCommand
			num = <-ReflectStatus.ReflectNum
			respCommand, _ = utils.ConstructPayload(utils.AdminId, "", "DATA", "REFLECT", " ", ports[0], num, AgentStatus.Nodeid, AgentStatus.AESKey, false)
			ProxyChan.ProxyChanToUpperNode <- respCommand
		}
		ReflectConnMap.Lock()
		ReflectConnMap.Payload[num] = conn
		go HandleReflectPort(ReflectConnMap.Payload[num], num, AgentStatus.Nodeid)
		ReflectConnMap.Unlock()
	}
}

//处理传入连接
func HandleReflectPort(reflectconn net.Conn, num uint32, nodeid string) {
	buffer := make([]byte, 10240)
	for {
		len, err := reflectconn.Read(buffer)
		if err != nil {
			finMessage, _ := utils.ConstructPayload(utils.AdminId, "", "DATA", "REFLECTFIN", " ", " ", num, AgentStatus.Nodeid, AgentStatus.AESKey, false)
			ProxyChan.ProxyChanToUpperNode <- finMessage
			return
		} else {
			respData, _ := utils.ConstructPayload(utils.AdminId, "", "DATA", "REFLECTDATA", " ", string(buffer[:len]), num, AgentStatus.Nodeid, AgentStatus.AESKey, false)
			ProxyChan.ProxyChanToUpperNode <- respData
		}
	}
}
