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
func TestForward(target string, controlconn *net.Conn) {
	forwardConn, err := net.Dial("tcp", target)
	if err != nil {
		respCommand, _ := common.ConstructCommand("FORWARDFAIL", " ", 0, AESKey)
		(*controlconn).Write(respCommand)
	} else {
		defer forwardConn.Close()
		respCommand, _ := common.ConstructCommand("FORWARDOK", " ", 0, AESKey)
		(*controlconn).Write(respCommand)
	}

}

//连接指定端口
func TryForward(controlconn *net.Conn, target string, num uint32) {
	forwardConn, err := net.Dial("tcp", target)
	if err == nil {
		ForwardConnMap.Lock()
		ForwardConnMap.Payload[num] = forwardConn
		ForwardConnMap.Unlock()
	} else {
		return
	}
}

//转发并处理port-forward所传递的数据
func HandleForward(dataConn *net.Conn, forwardDataChan chan string, num uint32) {
	ForwardConnMap.RLock()
	forwardConn := ForwardConnMap.Payload[num]
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
				forwardConn.Close()
				respdata, _ := common.ConstructDataResult(0, num, " ", "FORWARDOFFLINE", " ", AESKey, NODEID)
				(*dataConn).Write(respdata)
				ForwardConnMap.Lock()
				if _, ok := ForwardConnMap.Payload[num]; ok {
					ForwardConnMap.Payload[num].Close()
					delete(ForwardConnMap.Payload, num)
				}
				ForwardConnMap.Unlock()
				PortFowardMap.Lock()
				if _, ok := PortFowardMap.Payload[num]; ok {
					if !IsClosed(PortFowardMap.Payload[num]) {
						close(PortFowardMap.Payload[num])
					}
				}
				PortFowardMap.Unlock()
				return
			}
			respdata, _ := common.ConstructDataResult(0, num, " ", "FORWARDDATARESP", string(serverbuffer[:len]), AESKey, NODEID)
			(*dataConn).Write(respdata)
		}
	}()
}
