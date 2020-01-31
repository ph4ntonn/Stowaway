package agent

import (
	"Stowaway/common"
	"Stowaway/socks"
	"net"
	"strconv"
	"time"
)

//暂时没啥用，仅做回复socks开启命令之用
func StartSocks(controlConnToAdmin *net.Conn) {
	socksstartmess, _ := common.ConstructCommand("SOCKSRESP", "SUCCESS", NODEID, AESKey)
	(*controlConnToAdmin).Write(socksstartmess)
}

//处理socks请求
func HanleClientSocksConn(info chan string, socksUsername, socksPass string, checknum uint32, currentid uint32) {
	var (
		server       net.Conn
		serverflag   bool
		isAuthed     bool   = false
		method       string = ""
		tcpconnected bool   = false
	)
	for {
		if isAuthed == false && method == "" {
			data := <-info
			if data == "" { //重连后原先引用失效，当chan释放后，若不捕捉，会无限循环
				return
			}
			method = socks.CheckMethod(DataConnToAdmin, []byte(data), socksUsername, socksPass, checknum, AESKey, NODEID)
			if method == "NONE" {
				isAuthed = true
			}
		} else if isAuthed == false && method == "PASSWORD" {
			data := <-info
			if data == "" {
				return
			}
			isAuthed = socks.AuthClient(DataConnToAdmin, []byte(data), socksUsername, socksPass, checknum, AESKey, NODEID)
		} else if isAuthed == true && tcpconnected == false {
			data := <-info
			if data == "" {
				return
			}
			server, tcpconnected, serverflag = socks.ConfirmTarget(DataConnToAdmin, []byte(data), checknum, AESKey, NODEID)
			if serverflag == false {
				return
			}
		} else if isAuthed == true && tcpconnected == true && serverflag == true {
			go func() {
				for {
					data := <-info
					if data == "" {
						return
					}
					_, err := server.Write([]byte(data))
					if err != nil {
						SocksDataChanMap.RLock()
						if _, ok := SocksDataChanMap.SocksDataChan[checknum]; ok {
							SocksDataChanMap.RUnlock()
							continue
						} else {
							SocksDataChanMap.RUnlock()
							return
						}
					}
				}
			}()
			err := socks.Proxyhttp(DataConnToAdmin, server, checknum, AESKey, currentid)
			if err != nil {
				go SendFIN(DataConnToAdmin, checknum)
				return
			}
		} else {
			return
		}
	}
}

//发送server offline通知
func SendFIN(conn net.Conn, num uint32) {
	nodeid := strconv.Itoa(int(NODEID))
	for {
		SocksDataChanMap.RLock()
		if _, ok := SocksDataChanMap.SocksDataChan[num]; ok {
			SocksDataChanMap.RUnlock()
			//fmt.Println("send fin!!! number is ", num)
			respData, _ := common.ConstructDataResult(0, num, " ", "FIN", nodeid, AESKey, 0)
			conn.Write(respData)
		} else {
			SocksDataChanMap.RUnlock()
			//fmt.Print("out!!!!!,number is ", num)
			return
		}
		time.Sleep(5 * time.Second)
	}

}
