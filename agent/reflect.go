package agent

import (
	"fmt"
	"net"
	"strings"

	"Stowaway/utils"
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
		ProxyChan.ProxyChanToUpperNode <- respCommand
		return
	}

	defer reflectListenerForClient.Close()

	respCommand, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "REFLECTOK", " ", " ", 0, AgentStatus.Nodeid, AgentStatus.AESKey, false)
	ProxyChan.ProxyChanToUpperNode <- respCommand

	//记录此listener
	CurrentPortReflectListener = append(CurrentPortReflectListener, reflectListenerForClient)
	//等待连接
	for {
		conn, err := reflectListenerForClient.Accept()

		if err != nil {
			return
		}

		respCommand, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "GETREFLECTNUM", " ", ports[0], 0, AgentStatus.Nodeid, AgentStatus.AESKey, false)
		ProxyChan.ProxyChanToUpperNode <- respCommand

		num = <-ReflectStatus.ReflectNum

		respCommand, _ = utils.ConstructPayload(utils.AdminId, "", "DATA", "REFLECT", " ", ports[0], num, AgentStatus.Nodeid, AgentStatus.AESKey, false)
		ProxyChan.ProxyChanToUpperNode <- respCommand

		ReflectConnMap.Lock()
		ReflectConnMap.Payload[num] = conn
		go HandleReflectPort(ReflectConnMap.Payload[num], num, AgentStatus.Nodeid)
		ReflectConnMap.Unlock()
	}
}

// HandleReflectPort 处理传入连接
func HandleReflectPort(reflectconn net.Conn, num uint32, nodeid string) {
	buffer := make([]byte, 20480)

	for {
		len, err := reflectconn.Read(buffer)

		if err != nil {
			finMessage, _ := utils.ConstructPayload(utils.AdminId, "", "DATA", "REFLECTFIN", " ", " ", num, AgentStatus.Nodeid, AgentStatus.AESKey, false)
			ProxyChan.ProxyChanToUpperNode <- finMessage
			return
		}

		respData, _ := utils.ConstructPayload(utils.AdminId, "", "DATA", "REFLECTDATA", " ", string(buffer[:len]), num, AgentStatus.Nodeid, AgentStatus.AESKey, false)
		ProxyChan.ProxyChanToUpperNode <- respData
	}
}
