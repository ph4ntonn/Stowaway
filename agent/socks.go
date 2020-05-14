package agent

import (
	"net"

	"Stowaway/utils"
)

var CurrentConn *utils.Uint32ConnMap

func init() {
	CurrentConn = utils.NewUint32ConnMap()
}

/*-------------------------Socks启动相关代码--------------------------*/

// StartSocks 暂时没啥用，仅做回复socks开启命令之用
func StartSocks() {
	socksstartmess, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "SOCKSRESP", " ", "SUCCESS", 0, AgentStatus.Nodeid, AgentStatus.AESKey, false)
	ProxyChan.ProxyChanToUpperNode <- socksstartmess
}

// HanleClientSocksConn 处理socks请求
func HanleClientSocksConn(info chan string, socksUsername, socksPass string, checknum uint32, currentid string) {
	var (
		server       net.Conn
		serverflag   bool
		isAuthed     bool   = false
		method       string = ""
		tcpconnected bool   = false
	)
	for {
		if isAuthed == false && method == "" {
			data, ok := <-info
			if !ok { //重连后原先引用失效，当chan释放后，若不捕捉，会无限循环
				return
			}

			method = CheckMethod(ConnToAdmin, []byte(data), socksUsername, socksPass, checknum, AgentStatus.AESKey, AgentStatus.Nodeid)

			if method == "NONE" {
				isAuthed = true
			}
		} else if isAuthed == false && method == "PASSWORD" {
			data, ok := <-info
			if !ok {
				return
			}

			isAuthed = AuthClient(ConnToAdmin, []byte(data), socksUsername, socksPass, checknum, AgentStatus.AESKey, AgentStatus.Nodeid)
		} else if isAuthed == true && tcpconnected == false {
			data, ok := <-info
			if !ok {
				return
			}

			server, tcpconnected, serverflag = ConfirmTarget(ConnToAdmin, []byte(data), checknum, AgentStatus.AESKey, AgentStatus.Nodeid)
			if serverflag == false {
				return
			}

			CurrentConn.Lock() //这个 “concurrent map writes” 错误调了好久，死活没看出来，控制台日志贼长看不见错哪儿，重定向到文件之后想让他报错又tm不报错了（笑）
			CurrentConn.Payload[checknum] = server
			CurrentConn.Unlock()
		} else if isAuthed == true && tcpconnected == true && serverflag == true { //All done!
			go func() {
				for {
					data, ok := <-info
					if !ok {
						return
					}
					_, err := server.Write([]byte(data))
					if err != nil {
						SocksDataChanMap.RLock()
						if _, ok := SocksDataChanMap.Payload[checknum]; ok {
							SocksDataChanMap.RUnlock()
							continue
						} else {
							SocksDataChanMap.RUnlock()
							return
						}
					}
				}
			}()

			err := Proxyhttp(ConnToAdmin, server, checknum, AgentStatus.AESKey, currentid)

			if err != nil {
				go SendFin(checknum)
				return
			}

		} else {
			return
		}
	}
}

// SendFin 发送server offline通知
func SendFin(num uint32) {
	SocksDataChanMap.RLock()
	if _, ok := SocksDataChanMap.Payload[num]; ok {
		respData, _ := utils.ConstructPayload(utils.AdminId, "", "DATA", "FIN", " ", " ", num, AgentStatus.Nodeid, AgentStatus.AESKey, false)
		ProxyChan.ProxyChanToUpperNode <- respData
	}
	SocksDataChanMap.RUnlock()
	return
}
