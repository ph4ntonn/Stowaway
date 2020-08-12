package agent

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"Stowaway/utils"
)

/*-------------------------Socks5功能代码-------------------------*/

// CheckMethod 判断是否需要用户名/密码
func CheckMethod(connToUpper net.Conn, buffer []byte, username string, secret string, clientid uint32, key []byte, currentid string) string {
	if buffer[0] == 0x05 {
		if buffer[2] == 0x02 && (username != "") {
			utils.ConstructPayloadAndSend(connToUpper, utils.AdminId, "", "DATA", "TSOCKSDATARESP", " ", string([]byte{0x05, 0x02}), clientid, currentid, key, false)
			return "PASSWORD"
		} else if buffer[2] == 0x00 && (username == "" && secret == "") {
			utils.ConstructPayloadAndSend(connToUpper, utils.AdminId, "", "DATA", "TSOCKSDATARESP", " ", string([]byte{0x05, 0x00}), clientid, currentid, key, false)
			return "NONE"
		} else if buffer[2] == 0x00 && (username != "") {
			utils.ConstructPayloadAndSend(connToUpper, utils.AdminId, "", "DATA", "TSOCKSDATARESP", " ", string([]byte{0x05, 0x02}), clientid, currentid, key, false)
			return "ILLEGAL"
		} else if buffer[2] == 0x02 && (username == "") {
			utils.ConstructPayloadAndSend(connToUpper, utils.AdminId, "", "DATA", "TSOCKSDATARESP", " ", string([]byte{0x05, 0x00}), clientid, currentid, key, false)
			return "ILLEGAL"
		}
	}
	return "RETURN"
}

// AuthClient 如果需要用户名/密码，验证用户合法性
func AuthClient(connToUpper net.Conn, buffer []byte, username string, secret string, clientid uint32, key []byte, currentid string) bool {
	ulen := int(buffer[1])
	slen := int(buffer[2+ulen])
	clientName := string(buffer[2 : 2+ulen])
	clientPass := string(buffer[3+ulen : 3+ulen+slen])

	if clientName != username || clientPass != secret {
		utils.ConstructPayloadAndSend(connToUpper, utils.AdminId, "", "DATA", "TSOCKSDATARESP", " ", string([]byte{0x01, 0x01}), clientid, currentid, key, false)
		return false
	}
	utils.ConstructPayloadAndSend(connToUpper, utils.AdminId, "", "DATA", "TSOCKSDATARESP", " ", string([]byte{0x01, 0x00}), clientid, currentid, key, false)
	return true
}

// ConfirmTarget 判断代理方式
func ConfirmTarget(connToUpper net.Conn, buffer []byte, checkNum uint32, key []byte, currentid string) (net.Conn, bool, bool, bool, bool) {
	len := len(buffer)
	connected := false
	var server net.Conn
	var serverFlag bool
	var isUDP bool
	var success bool

	if buffer[0] == 0x05 {
		switch buffer[1] {
		case 0x01:
			server, connected, serverFlag = TCPConnect(connToUpper, buffer, len, checkNum, key, currentid)
		case 0x02:
			connected = TCPBind(connToUpper, buffer, len, checkNum, key)
		case 0x03:
			success = UDPAssociate(connToUpper, buffer, len, checkNum, key, currentid)
			isUDP = true
		default:
			return server, connected, false, isUDP, success
		}
	}

	return server, connected, serverFlag, isUDP, success
}

// TCPConnect 如果是代理tcp
func TCPConnect(connToUpper net.Conn, buffer []byte, len int, checkNum uint32, key []byte, currentid string) (net.Conn, bool, bool) {
	host := ""
	var server net.Conn

	switch buffer[3] {
	case 0x01:
		host = net.IPv4(buffer[4], buffer[5], buffer[6], buffer[7]).String()
	case 0x03:
		host = string(buffer[5 : len-2])
	case 0x04:
		host = net.IP{buffer[4], buffer[5], buffer[6], buffer[7],
			buffer[8], buffer[9], buffer[10], buffer[11], buffer[12],
			buffer[13], buffer[14], buffer[15], buffer[16], buffer[17],
			buffer[18], buffer[19]}.String()
	default:
		return server, false, false
	}

	port := strconv.Itoa(int(buffer[len-2])<<8 | int(buffer[len-1]))

	server, err := net.Dial("tcp", net.JoinHostPort(host, port))

	if err != nil {
		utils.ConstructPayloadAndSend(connToUpper, utils.AdminId, "", "DATA", "TSOCKSDATARESP", " ", string([]byte{0x05, 0x04, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}), checkNum, currentid, key, false)
		return server, false, false
	}

	utils.ConstructPayloadAndSend(connToUpper, utils.AdminId, "", "DATA", "TSOCKSDATARESP", " ", string([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}), checkNum, currentid, key, false)

	return server, true, true
}

// ProxyTCP 转发流量
func ProxyTCP(connToUpper net.Conn, server net.Conn, checkNum uint32, key []byte, currentid string) error {
	serverBuffer := make([]byte, 20480)

	for {
		len, err := server.Read(serverBuffer)
		if err != nil {
			server.Close()
			return err
		}
		utils.ConstructPayloadAndSend(connToUpper, utils.AdminId, "", "DATA", "TSOCKSDATARESP", " ", string(serverBuffer[:len]), checkNum, currentid, key, false)
	}
}

// 基于rfc1928编写，如果客户端没有严格按照rfc1928规定发送数据包，可能导致agent崩溃！
// UDPAssociate UDPAssociate方式
func UDPAssociate(connToUpper net.Conn, buffer []byte, len int, checkNum uint32, key []byte, currentid string) bool {
	var host string
	newUDPAssociate := utils.NewUDPAssociateInfo()

	switch buffer[3] {
	case 0x01:
		host = net.IPv4(buffer[4], buffer[5], buffer[6], buffer[7]).String()
	case 0x03:
		host = string(buffer[5 : len-2])
	case 0x04:
		host = net.IP{buffer[4], buffer[5], buffer[6], buffer[7],
			buffer[8], buffer[9], buffer[10], buffer[11], buffer[12],
			buffer[13], buffer[14], buffer[15], buffer[16], buffer[17],
			buffer[18], buffer[19]}.String()
	default:
		return false
	}

	port := strconv.Itoa(int(buffer[len-2])<<8 | int(buffer[len-1])) //先拿到客户端想要发送数据的ip:port地址

	udpListenerAddr, err := net.ResolveUDPAddr("udp", "0.0.0.0:0")
	if err != nil {
		utils.ConstructPayloadAndSend(connToUpper, utils.AdminId, "", "DATA", "TSOCKSDATARESP", " ", string([]byte{0x05, 0x01, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}), checkNum, currentid, key, false)
		return false
	}

	udpListener, err := net.ListenUDP("udp", udpListenerAddr)
	if err != nil {
		utils.ConstructPayloadAndSend(connToUpper, utils.AdminId, "", "DATA", "TSOCKSDATARESP", " ", string([]byte{0x05, 0x01, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}), checkNum, currentid, key, false)
		return false
	}

	newUDPAssociate.Listener = udpListener
	newUDPAssociate.SourceAddr = net.JoinHostPort(host, port)

	AgentStuff.Socks5UDPAssociate.Info[checkNum] = newUDPAssociate

	utils.ConstructPayloadAndSend(connToUpper, utils.AdminId, "", "COMMAND", "STARTUDPASS", " ", net.JoinHostPort(host, port), checkNum, currentid, key, false)

	if adminResponse := <-AgentStuff.Socks5UDPAssociate.Info[checkNum].Ready; adminResponse != "" {
		temp := strings.Split(adminResponse, ":")
		adminAddr := temp[0]
		adminPort, _ := strconv.Atoi(temp[1])

		localAddr := utils.SocksLocalAddr{adminAddr, adminPort}
		buf := make([]byte, 10)
		copy(buf, []byte{0x05, 0x00, 0x00, 0x01})
		copy(buf[4:], localAddr.ByteArray())

		utils.ConstructPayloadAndSend(connToUpper, utils.AdminId, "", "DATA", "TSOCKSDATARESP", " ", string(buf), checkNum, currentid, key, false)
		return true
	}
	utils.ConstructPayloadAndSend(connToUpper, utils.AdminId, "", "DATA", "TSOCKSDATARESP", " ", string([]byte{0x05, 0x01, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}), checkNum, currentid, key, false)
	return false
}

// ProxyUDP 代理udp流量
func ProxyUDP(connToUpper net.Conn, checkNum uint32, key []byte, currentid string) error {
	serverBuffer := make([]byte, 20480)
	var data []byte

	AgentStuff.Socks5UDPAssociate.Lock()
	udpConn := AgentStuff.Socks5UDPAssociate.Info[checkNum].Listener
	AgentStuff.Socks5UDPAssociate.Unlock()

	for {
		length, addr, err := udpConn.ReadFromUDP(serverBuffer)
		if err != nil {
			return err
		}
		AgentStuff.Socks5UDPAssociate.Lock()
		if header, ok := AgentStuff.Socks5UDPAssociate.Info[checkNum].Pair[addr.String()]; ok {
			data = make([]byte, 0, len(header)+length)
			data = append(data, header...)
			data = append(data, serverBuffer[:length]...)
		} else {
			AgentStuff.Socks5UDPAssociate.Unlock()
			continue
		}
		AgentStuff.Socks5UDPAssociate.Unlock()
		utils.ConstructPayloadAndSend(connToUpper, utils.AdminId, "", "DATA", "USOCKSDATARESP", " ", string(data), checkNum, currentid, key, false)
	}
}

// TCPBind TCPBind方式
func TCPBind(connToUpper net.Conn, buffer []byte, len int, checkNum uint32, AESKey []byte) bool {
	fmt.Println("Not ready") //limited use, add to Todo
	return false
}
