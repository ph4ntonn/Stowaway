package agent

import (
	"fmt"
	"log"
	"net"
	"strconv"

	"Stowaway/utils"
)

/*-------------------------Socks5功能代码-------------------------*/

// CheckMethod 判断是否需要用户名/密码
func CheckMethod(connToUpper net.Conn, buffer []byte, username string, secret string, clientid uint32, key []byte, currentid string) string {
	if buffer[0] == 0x05 {
		if buffer[2] == 0x02 && (username != "") {
			utils.ConstructPayloadAndSend(connToUpper, utils.AdminId, "", "DATA", "SOCKSDATARESP", " ", string([]byte{0x05, 0x02}), clientid, currentid, key, false)
			return "PASSWORD"
		} else if buffer[2] == 0x00 && (username == "" && secret == "") {
			utils.ConstructPayloadAndSend(connToUpper, utils.AdminId, "", "DATA", "SOCKSDATARESP", " ", string([]byte{0x05, 0x00}), clientid, currentid, key, false)
			return "NONE"
		} else if buffer[2] == 0x00 && (username != "") {
			utils.ConstructPayloadAndSend(connToUpper, utils.AdminId, "", "DATA", "SOCKSDATARESP", " ", string([]byte{0x05, 0x02}), clientid, currentid, key, false)
			return "ILLEGAL"
		} else if buffer[2] == 0x02 && (username == "") {
			utils.ConstructPayloadAndSend(connToUpper, utils.AdminId, "", "DATA", "SOCKSDATARESP", " ", string([]byte{0x05, 0x00}), clientid, currentid, key, false)
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
		log.Println("Illegal client!")
		utils.ConstructPayloadAndSend(connToUpper, utils.AdminId, "", "DATA", "SOCKSDATARESP", " ", string([]byte{0x01, 0x01}), clientid, currentid, key, false)
		return false
	}
	utils.ConstructPayloadAndSend(connToUpper, utils.AdminId, "", "DATA", "SOCKSDATARESP", " ", string([]byte{0x01, 0x00}), clientid, currentid, key, false)
	return true
}

// ConfirmTarget 判断代理方式
func ConfirmTarget(connToUpper net.Conn, buffer []byte, checkNum uint32, key []byte, currentid string) (net.Conn, bool, bool) {
	len := len(buffer)
	connected := false
	var server net.Conn
	var serverFlag bool

	if buffer[0] == 0x05 {
		switch buffer[1] {
		case 0x01:
			server, connected, serverFlag = TCPConnect(connToUpper, buffer, len, checkNum, key, currentid)
		case 0x02:
			connected = TCPBind(connToUpper, buffer, len, checkNum, key)
		case 0x03:
			connected = UDPAssociate(connToUpper, buffer, len, checkNum, key)
		}
	}

	return server, connected, serverFlag
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
	}

	port := strconv.Itoa(int(buffer[len-2])<<8 | int(buffer[len-1]))

	server, err := net.Dial("tcp", net.JoinHostPort(host, port))

	if err != nil {
		utils.ConstructPayloadAndSend(connToUpper, utils.AdminId, "", "DATA", "SOCKSDATARESP", " ", string([]byte{0x05, 0x04, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}), checkNum, currentid, key, false)
		return server, false, false
	}

	utils.ConstructPayloadAndSend(connToUpper, utils.AdminId, "", "DATA", "SOCKSDATARESP", " ", string([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}), checkNum, currentid, key, false)

	return server, true, true
}

// Proxyhttp 转发流量
func Proxyhttp(connToUpper net.Conn, server net.Conn, checkNum uint32, key []byte, currentid string) error {
	serverBuffer := make([]byte, 20480)

	for {
		len, err := server.Read(serverBuffer)
		if err != nil {
			server.Close()
			return err
		}
		utils.ConstructPayloadAndSend(connToUpper, utils.AdminId, "", "DATA", "SOCKSDATARESP", " ", string(serverBuffer[:len]), checkNum, currentid, key, false)
	}
}

// TCPBind TCPBind方式
func TCPBind(client net.Conn, buffer []byte, len int, checkNum uint32, AESKey []byte) bool {
	fmt.Println("Not ready") //limited use, add to Todo
	return false
}

// UDPAssociate UDPAssociate方式
func UDPAssociate(client net.Conn, buffer []byte, len int, checkNum uint32, AESKey []byte) bool {
	fmt.Println("Not ready") //limited use, add to Todo
	return false
}
