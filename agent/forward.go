package agent

import (
	"net"

	"Stowaway/utils"
)

var (
	PortFowardMap  *utils.Uint32ChanStrMap
	ForwardConnMap *utils.Uint32ConnMap
)

func init() {
	PortFowardMap = utils.NewUint32ChanStrMap()
	ForwardConnMap = utils.NewUint32ConnMap()
}

/*-------------------------Port-forward启动相关代码--------------------------*/
//检查需要映射的端口是否listen
func TestForward(target string) {
	forwardConn, err := net.Dial("tcp", target)
	if err != nil {
		respCommand, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "FORWARDFAIL", " ", " ", 0, AgentStatus.Nodeid, AgentStatus.AESKey, false)
		ProxyChan.ProxyChanToUpperNode <- respCommand
	} else {
		defer forwardConn.Close()
		respCommand, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "FORWARDOK", " ", " ", 0, AgentStatus.Nodeid, AgentStatus.AESKey, false)
		ProxyChan.ProxyChanToUpperNode <- respCommand
	}

}

//连接指定端口
func TryForward(target string, num uint32) {
	forwardConn, err := net.Dial("tcp", target)
	if err == nil {
		ForwardConnMap.Lock()
		ForwardConnMap.Payload[num] = forwardConn
		ForwardConnMap.Unlock()
	} else {
		respdata, _ := utils.ConstructPayload(utils.AdminId, "", "DATA", "FORWARDTIMEOUT", " ", " ", num, AgentStatus.Nodeid, AgentStatus.AESKey, false)
		ProxyChan.ProxyChanToUpperNode <- respdata
		return
	}
}

//转发并处理port-forward所传递的数据
func HandleForward(forwardDataChan chan string, forwardNum uint32) {
	ForwardConnMap.RLock()
	forwardConn := ForwardConnMap.Payload[forwardNum]
	ForwardConnMap.RUnlock()

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
				respdata, _ := utils.ConstructPayload(utils.AdminId, "", "DATA", "FORWARDOFFLINE", " ", " ", forwardNum, AgentStatus.Nodeid, AgentStatus.AESKey, false)
				ProxyChan.ProxyChanToUpperNode <- respdata
				return
			}
			respdata, _ := utils.ConstructPayload(utils.AdminId, "", "DATA", "FORWARDDATARESP", " ", string(serverbuffer[:len]), forwardNum, AgentStatus.Nodeid, AgentStatus.AESKey, false)
			ProxyChan.ProxyChanToUpperNode <- respdata
		}
	}()
}
