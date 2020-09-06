package agent

import (
	"net"

	"Stowaway/utils"
)

/*-------------------------Socks启动相关代码--------------------------*/
// HanleClientSocksConn 处理socks请求
func HanleClientSocksConn(info chan string, socksUsername, socksPass string, checkNum uint32, currentid string) {
	var (
		server       net.Conn
		serverFlag   bool
		isAuthed     bool
		method       string
		tcpConnected bool
		isUDP        bool
		success      bool
	)

	for {
		if isAuthed == false && method == "" {
			data, ok := <-info
			if !ok { //重连后原先引用失效，当chan释放后，若不捕捉，会无限循环
				return
			}

			method = CheckMethod(ConnToAdmin, []byte(data), socksUsername, socksPass, checkNum, AgentStatus.AESKey, AgentStatus.Nodeid)

			if method == "NONE" {
				isAuthed = true
			}
		} else if isAuthed == false && method == "PASSWORD" {
			data, ok := <-info
			if !ok {
				return
			}

			isAuthed = AuthClient(ConnToAdmin, []byte(data), socksUsername, socksPass, checkNum, AgentStatus.AESKey, AgentStatus.Nodeid)
		} else if isAuthed == true && tcpConnected == false && !isUDP {
			data, ok := <-info
			if !ok {
				return
			}

			server, tcpConnected, serverFlag, isUDP, success = ConfirmTarget(ConnToAdmin, []byte(data), checkNum, AgentStatus.AESKey, AgentStatus.Nodeid)
			if serverFlag == false && !isUDP {
				return
			}

			if !isUDP {
				AgentStuff.CurrentSocks5Conn.Lock() //这个 “concurrent map writes” 错误调了好久，死活没看出来，控制台日志贼长看不见错哪儿，重定向到文件之后想让他报错又tm不报错了（笑）
				AgentStuff.CurrentSocks5Conn.Payload[checkNum] = server
				AgentStuff.CurrentSocks5Conn.Unlock()
			}
		} else if isAuthed == true && tcpConnected == true && serverFlag == true && !isUDP { //All done!
			defer SendTCPFin(checkNum)

			go ProxyC2STCP(info,server,checkNum)

			if err := ProxyS2CTCP(ConnToAdmin, server, checkNum, AgentStatus.AESKey, currentid); err != nil {
				return
			}
		} else if isAuthed == true && isUDP && success {
			defer SendUDPFin(checkNum)

			go ProxyC2SUDP(checkNum)

			if err := ProxyS2CUDP(ConnToAdmin, checkNum, AgentStatus.AESKey, currentid); err != nil {
				return
			}
		} else {
			return
		}
	}
}

// SendTCPFin 发送tcp server offline通知
func SendTCPFin(num uint32) {
	respData, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "FIN", " ", " ", num, AgentStatus.Nodeid, AgentStatus.AESKey, false)
	AgentStuff.ProxyChan.ProxyChanToUpperNode <- respData
}

// SendUDPFin 发送udp listener offline通知
func SendUDPFin(num uint32) {
	respData, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "UDPFIN", " ", " ", num, AgentStatus.Nodeid, AgentStatus.AESKey, false)
	AgentStuff.ProxyChan.ProxyChanToUpperNode <- respData
}
