package agent

import (
	"Stowaway/common"
	"Stowaway/node"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

/*-------------------------重连功能相关代码--------------------------*/
func TryReconnect(gap string, monitor string, listenPort string) {
	lag, _ := strconv.Atoi(gap)
	for {
		time.Sleep(time.Duration(lag) * time.Second)

		controlConnToAdmin, dataConnToAdmin, _, err := node.StartNodeConn(monitor, listenPort, NODEID, AESKey)
		if err != nil {
			fmt.Println("[*]Admin seems still down")
		} else {
			fmt.Println("[*]Admin up! Reconnect successful!")
			ControlConnToAdmin = controlConnToAdmin
			DataConnToAdmin = dataConnToAdmin
			go common.SendHeartBeatControl(ControlConnToAdmin, NODEID, AESKey)
			return
		}
	}
}

/*-------------------------命令传递相关代码--------------------------*/
//将命令传递到下一节点
func ProxyCommToNextNode(proxyCommand []byte) {
	Proxy_Command_Chan <- proxyCommand
}

//将数据传递到下一节点
func ProxyDataToNextNode(proxyData []byte) {
	Proxy_Data_Chan <- proxyData
}

/*-------------------------程序控制相关代码--------------------------*/
//捕捉程序退出信号
func WaitForExit(NODEID uint32) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, os.Kill, syscall.SIGHUP)
	<-signalChan
	if NotLastOne {
		offlineMess, _ := common.ConstructCommand("OFFLINE", "", NODEID+1, AESKey)
		Proxy_Command_Chan <- offlineMess
	}
	time.Sleep(5 * time.Second)
	os.Exit(1)
}

/*-------------------------清除现存连接及发送FIN信号相关代码--------------------------*/
//当admin下线后，清除并关闭所有现存的socket
func ClearAllConn() {
	CurrentConn.Lock()
	for key, conn := range CurrentConn.Payload {
		err := conn.Close()
		if err != nil {
		}
		delete(CurrentConn.Payload, key)
	}
	CurrentConn.Unlock()

	SocksDataChanMap.Lock()
	for key, _ := range SocksDataChanMap.Payload {
		if !common.IsClosed(SocksDataChanMap.Payload[key]) {
			close(SocksDataChanMap.Payload[key])
		}
		delete(SocksDataChanMap.Payload, key)
	}
	SocksDataChanMap.Unlock()

	PortFowardMap.Lock()
	for key, _ := range PortFowardMap.Payload {
		if !common.IsClosed(PortFowardMap.Payload[key]) {
			close(PortFowardMap.Payload[key])
		}
		delete(PortFowardMap.Payload, key)
	}
	PortFowardMap.Unlock()

	ForwardConnMap.Lock()
	for key, conn := range ForwardConnMap.Payload {
		err := conn.Close()
		if err != nil {
		}
		delete(ForwardConnMap.Payload, key)
	}
	ForwardConnMap.Unlock()

	ReflectConnMap.Lock()
	for key, conn := range ReflectConnMap.Payload {
		err := conn.Close()
		if err != nil {
		}
		delete(ForwardConnMap.Payload, key)
	}
	ReflectConnMap.Unlock()

	for _, listener := range CurrentPortReflectListener {
		listener.Close()
	}

}

//发送server offline通知
func SendFin(conn net.Conn, num uint32) {
	nodeid := strconv.Itoa(int(NODEID))
	SocksDataChanMap.RLock()
	if _, ok := SocksDataChanMap.Payload[num]; ok {
		SocksDataChanMap.RUnlock()
		//fmt.Println("send fin!!! number is ", num)
		respData, _ := common.ConstructDataResult(0, num, " ", "FIN", nodeid, AESKey, 0)
		conn.Write(respData)
		return
	} else {
		SocksDataChanMap.RUnlock()
		//fmt.Print("out!!!!!,number is ", num)
		return
	}
}
