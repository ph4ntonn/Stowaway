package agent

import (
	"fmt"
	"net"

	"Stowaway/utils"
)

/*-------------------------Socks启动相关代码--------------------------*/

// StartSocks 暂时没啥用，仅做回复socks开启命令之用
func StartSocks() {
	socksStartMess, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "SOCKSRESP", " ", "SUCCESS", 0, AgentStatus.Nodeid, AgentStatus.AESKey, false)
	AgentStuff.ProxyChan.ProxyChanToUpperNode <- socksStartMess
}

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

			go func() {
				for {
					data, ok := <-info
					if !ok {
						return
					}
					_, err := server.Write([]byte(data))
					if err != nil {
						AgentStuff.SocksDataChanMap.Lock()
						if _, ok := AgentStuff.SocksDataChanMap.Payload[checkNum]; ok {
							AgentStuff.SocksDataChanMap.Unlock()
							continue
						} else {
							AgentStuff.SocksDataChanMap.Unlock()
							return
						}
					}
				}
			}()

			if err := ProxyTCP(ConnToAdmin, server, checkNum, AgentStatus.AESKey, currentid); err != nil {
				return
			}
		} else if isAuthed == true && isUDP && success {
			defer SendUDPFin(checkNum)

			go func() {
				AgentStuff.Socks5UDPAssociate.Lock()
				listener := AgentStuff.Socks5UDPAssociate.Info[checkNum].Listener
				AgentStuff.Socks5UDPAssociate.Unlock()

				for {
					data, ok := <-AgentStuff.Socks5UDPAssociate.Info[checkNum].UDPData
					if !ok {
						return
					}

					buf := []byte(data)

					if buf[0] != 0x00 || buf[1] != 0x00 || buf[2] != 0x00 {
						continue
					}

					udpHeader := make([]byte, 0, 1024)
					addrtype := buf[3]
					var remote string
					var udpData []byte
					if addrtype == 0x01 { //IPV4
						ip := net.IPv4(buf[4], buf[5], buf[6], buf[7])
						remote = fmt.Sprintf("%s:%d", ip.String(), uint(buf[8])<<8+uint(buf[9]))
						udpData = buf[10:]
						udpHeader = append(udpHeader, buf[:10]...)
					} else if addrtype == 0x03 { //DOMAIN
						nmlen := int(buf[4])
						nmbuf := buf[5 : 5+nmlen+2]
						remote = fmt.Sprintf("%s:%d", nmbuf[:nmlen], uint(nmbuf[nmlen])<<8+uint(nmbuf[nmlen+1]))
						udpData = buf[8+nmlen:]
						udpHeader = append(udpHeader, buf[:8+nmlen]...)
					} else if addrtype == 0x04 { //IPV6
						ip := net.IP{buf[4], buf[5], buf[6], buf[7],
							buf[8], buf[9], buf[10], buf[11], buf[12],
							buf[13], buf[14], buf[15], buf[16], buf[17],
							buf[18], buf[19]}
						remote = fmt.Sprintf("[%s]:%d", ip.String(), uint(buf[20])<<8+uint(buf[21]))
						udpData = buf[22:]
						udpHeader = append(udpHeader, buf[:22]...)
					} else {
						continue
					}

					remoteAddr, err := net.ResolveUDPAddr("udp", remote)
					if err != nil {
						continue
					}

					AgentStuff.Socks5UDPAssociate.Lock()
					AgentStuff.Socks5UDPAssociate.Info[checkNum].Pair[remote] = udpHeader
					AgentStuff.Socks5UDPAssociate.Unlock()

					listener.WriteToUDP(udpData, remoteAddr)
				}
			}()

			if err := ProxyUDP(ConnToAdmin, checkNum, AgentStatus.AESKey, currentid); err != nil {
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
