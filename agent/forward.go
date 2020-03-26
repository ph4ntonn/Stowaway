package agent

import (
	"Stowaway/common"
	"net"
)

var (
	PortFowardMap  *common.Uint32ChanStrMap
	ForwardConnMap *common.Uint32ConnMap
)

func init() {
	PortFowardMap = common.NewUint32ChanStrMap()
	ForwardConnMap = common.NewUint32ConnMap()
}

/*-------------------------Port-forward启动相关代码--------------------------*/
//检查需要映射的端口是否listen
func TestForward(target string) {
	forwardConn, err := net.Dial("tcp", target)
	if err != nil {
		respCommand, _ := common.ConstructPayload(0, "COMMAND", "FORWARDFAIL", " ", " ", 0, AgentStatus.NODEID, AgentStatus.AESKey, false)
		ProxyChan.ProxyChanToUpperNode <- respCommand
	} else {
		defer forwardConn.Close()
		respCommand, _ := common.ConstructPayload(0, "COMMAND", "FORWARDOK", " ", " ", 0, AgentStatus.NODEID, AgentStatus.AESKey, false)
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
		respdata, _ := common.ConstructPayload(0, "DATA", "FORWARDTIMEOUT", " ", " ", num, AgentStatus.NODEID, AgentStatus.AESKey, false)
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
		serverbuffer := make([]byte, 10240)
		for {
			len, err := forwardConn.Read(serverbuffer)
			if err != nil {
				respdata, _ := common.ConstructPayload(0, "DATA", "FORWARDOFFLINE", " ", " ", forwardNum, AgentStatus.NODEID, AgentStatus.AESKey, false)
				ProxyChan.ProxyChanToUpperNode <- respdata
				return
			}
			respdata, _ := common.ConstructPayload(0, "DATA", "FORWARDDATARESP", " ", string(serverbuffer[:len]), forwardNum, AgentStatus.NODEID, AgentStatus.AESKey, false)
			ProxyChan.ProxyChanToUpperNode <- respdata
		}
	}()
}
