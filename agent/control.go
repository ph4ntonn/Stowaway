package agent

import (
	"Stowaway/common"
	"Stowaway/node"
	"fmt"
	"net"
	"os"
	"os/signal"
	"runtime"
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

/*-------------------------chan状态判断相关代码--------------------------*/
//判断chan是否已经被释放
func IsClosed(ch chan string) bool {
	select {
	case <-ch:
		return true
	default:
	}
	return false
}

/*-------------------------操作系统判断相关代码--------------------------*/
func CheckSystem() (sysType uint32) {
	var os = runtime.GOOS
	switch os {
	case "windows":
		sysType = 0x01
	default:
		sysType = 0xff
	}
	return
}

/*-------------------------Socks启动相关代码--------------------------*/
//暂时没啥用，仅做回复socks开启命令之用
func StartSocks(controlConnToAdmin *net.Conn) {
	socksstartmess, _ := common.ConstructCommand("SOCKSRESP", "SUCCESS", NODEID, AESKey)
	(*controlConnToAdmin).Write(socksstartmess)
}

/*-------------------------清除现存连接及发送FIN信号相关代码--------------------------*/
//当admin下线后，清除并关闭所有现存的socket
func ClearAllConn() {
	for key, conn := range CurrentConn.Payload {
		err := conn.Close()
		if err != nil {
		}
		delete(CurrentConn.Payload, key)
	}
	SocksDataChanMap.Lock()
	for key, _ := range SocksDataChanMap.Payload {
		if !IsClosed(SocksDataChanMap.Payload[key]) {
			close(SocksDataChanMap.Payload[key])
		}
		delete(SocksDataChanMap.Payload, key)
	}
	SocksDataChanMap.Unlock()
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
