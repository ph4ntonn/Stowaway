package agent

import (
	"fmt"
	"net"
	"strings"

	"Stowaway/utils"
)

var (
	CurrentPortReflectListener []net.Listener
)

/*-------------------------Port-reflect启动相关代码--------------------------*/

// TestReflect 检查是否能够监听
func TestReflect(portCombine string) {
	var num uint32
	ports := strings.Split(portCombine, ":")
	reflectAddr := fmt.Sprintf("0.0.0.0:%s", ports[1])
	//尝试监听指定的端口
	reflectListenerForClient, err := net.Listen("tcp", reflectAddr)
	//检查是否可以监听
	if err != nil {
		respCommand, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "REFLECTFAIL", " ", " ", 0, AgentStatus.Nodeid, AgentStatus.AESKey, false)
		AgentStuff.ProxyChan.ProxyChanToUpperNode <- respCommand
		return
	}

	defer reflectListenerForClient.Close()

	respCommand, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "REFLECTOK", " ", " ", 0, AgentStatus.Nodeid, AgentStatus.AESKey, false)
	AgentStuff.ProxyChan.ProxyChanToUpperNode <- respCommand

	//记录此listener
	CurrentPortReflectListener = append(CurrentPortReflectListener, reflectListenerForClient)
	//等待连接
	for {
		conn, err := reflectListenerForClient.Accept()

		if err != nil {
			return
		}

		respCommand, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "GETREFLECTNUM", " ", ports[0], 0, AgentStatus.Nodeid, AgentStatus.AESKey, false)
		AgentStuff.ProxyChan.ProxyChanToUpperNode <- respCommand

		num = <-AgentStuff.ReflectStatus.ReflectNum

		respCommand, _ = utils.ConstructPayload(utils.AdminId, "", "COMMAND", "REFLECT", " ", ports[0], num, AgentStatus.Nodeid, AgentStatus.AESKey, false)
		AgentStuff.ProxyChan.ProxyChanToUpperNode <- respCommand

		AgentStuff.ReflectConnMap.Lock()
		AgentStuff.ReflectConnMap.Payload[num] = conn
		go HandleReflectPort(AgentStuff.ReflectConnMap.Payload[num], num, AgentStatus.Nodeid)
		AgentStuff.ReflectConnMap.Unlock()
	}
}

// HandleReflectPort 处理传入连接
func HandleReflectPort(reflectconn net.Conn, num uint32, nodeid string) {
	buffer := make([]byte, 20480)

	for {
		len, err := reflectconn.Read(buffer)

		if err != nil {
			finMessage, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "REFLECTFIN", " ", " ", num, AgentStatus.Nodeid, AgentStatus.AESKey, false)
			AgentStuff.ProxyChan.ProxyChanToUpperNode <- finMessage
			return
		}

		respData, _ := utils.ConstructPayload(utils.AdminId, "", "DATA", "REFLECTDATA", " ", string(buffer[:len]), num, AgentStatus.Nodeid, AgentStatus.AESKey, false)
		AgentStuff.ProxyChan.ProxyChanToUpperNode <- respData
	}
}
