package agent

import (
	"net"

	"Stowaway/utils"
)

/*-------------------------Port-forward启动相关代码--------------------------*/

// TestForward 检查需要映射的端口是否listen
func TestForward(target string) {
	forwardConn, err := net.Dial("tcp", target)
	if err != nil {
		respCommand, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "FORWARDFAIL", " ", " ", 0, AgentStatus.Nodeid, AgentStatus.AESKey, false)
		AgentStuff.ProxyChan.ProxyChanToUpperNode <- respCommand
	} else {
		defer forwardConn.Close()
		respCommand, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "FORWARDOK", " ", " ", 0, AgentStatus.Nodeid, AgentStatus.AESKey, false)
		AgentStuff.ProxyChan.ProxyChanToUpperNode <- respCommand
	}
}

// TryForward 连接指定端口
func TryForward(target string, num uint32) {
	forwardConn, err := net.Dial("tcp", target)
	if err == nil {
		AgentStuff.ForwardConnMap.Lock()
		AgentStuff.ForwardConnMap.Payload[num] = forwardConn
		AgentStuff.ForwardConnMap.Unlock()
	} else {
		respdata, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "FORWARDTIMEOUT", " ", " ", num, AgentStatus.Nodeid, AgentStatus.AESKey, false)
		AgentStuff.ProxyChan.ProxyChanToUpperNode <- respdata
		return
	}
}

// HandleForward 转发并处理port-forward所传递的数据
func HandleForward(forwardDataChan chan string, forwardNum uint32) {
	forwardConn := utils.GetInfoViaLockMap(AgentStuff.ForwardConnMap, forwardNum).(net.Conn)

	go func() {
		for {
			forwardData, ok := <-forwardDataChan
			if ok {
				forwardConn.Write([]byte(forwardData))
			} else {
				return
			}
		}
	}()

	go func() {
		serverbuffer := make([]byte, 20480)
		for {
			len, err := forwardConn.Read(serverbuffer)
			if err != nil {
				respdata, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "FORWARDOFFLINE", " ", " ", forwardNum, AgentStatus.Nodeid, AgentStatus.AESKey, false)
				AgentStuff.ProxyChan.ProxyChanToUpperNode <- respdata
				return
			}
			respdata, _ := utils.ConstructPayload(utils.AdminId, "", "DATA", "FORWARDDATARESP", " ", string(serverbuffer[:len]), forwardNum, AgentStatus.Nodeid, AgentStatus.AESKey, false)
			AgentStuff.ProxyChan.ProxyChanToUpperNode <- respdata
		}
	}()
}
