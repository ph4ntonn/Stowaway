package agent

import (
	"fmt"
	"log"
	"net"
	"strconv"

	"Stowaway/utils"
)

/*-------------------------Socks5功能代码-------------------------*/
//判断是否需要用户名/密码
func CheckMethod(conntoupper net.Conn, buffer []byte, username string, secret string, clientid uint32, key []byte, currentid string) string {
	if buffer[0] == 0x05 {
		if buffer[2] == 0x02 && (username != "") {
			respdata, _ := utils.ConstructPayload(utils.AdminId, "", "DATA", "SOCKSDATARESP", " ", string([]byte{0x05, 0x02}), clientid, currentid, key, false)
			conntoupper.Write(respdata)
			return "PASSWORD"
		} else if buffer[2] == 0x00 && (username == "" && secret == "") {
			respdata, _ := utils.ConstructPayload(utils.AdminId, "", "DATA", "SOCKSDATARESP", " ", string([]byte{0x05, 0x00}), clientid, currentid, key, false)
			conntoupper.Write(respdata)
			return "NONE"
		} else if buffer[2] == 0x00 && (username != "") {
			respdata, _ := utils.ConstructPayload(utils.AdminId, "", "DATA", "SOCKSDATARESP", " ", string([]byte{0x05, 0x02}), clientid, currentid, key, false)
			conntoupper.Write(respdata)
			return "ILLEGAL"
		} else if buffer[2] == 0x02 && (username == "") {
			respdata, _ := utils.ConstructPayload(utils.AdminId, "", "DATA", "SOCKSDATARESP", " ", string([]byte{0x05, 0x00}), clientid, currentid, key, false)
			conntoupper.Write(respdata)
			return "ILLEGAL"
		}
	}
	return "RETURN"
}

//如果需要用户名/密码，验证用户合法性
func AuthClient(conntoupper net.Conn, buffer []byte, username string, secret string, clientid uint32, key []byte, currentid string) bool {
	ulen := int(buffer[1])
	slen := int(buffer[2+ulen])
	clientname := string(buffer[2 : 2+ulen])
	clientpass := string(buffer[3+ulen : 3+ulen+slen])

	if clientname != username || clientpass != secret {
		log.Println("Illegal client!")
		respdata, _ := utils.ConstructPayload(utils.AdminId, "", "DATA", "SOCKSDATARESP", " ", string([]byte{0x01, 0x01}), clientid, currentid, key, false)
		conntoupper.Write(respdata)
		return false
	} else {
		respdata, _ := utils.ConstructPayload(utils.AdminId, "", "DATA", "SOCKSDATARESP", " ", string([]byte{0x01, 0x00}), clientid, currentid, key, false)
		conntoupper.Write(respdata)
		return true
	}
}

//判断代理方式
func ConfirmTarget(conntoupper net.Conn, buffer []byte, checknum uint32, key []byte, currentid string) (net.Conn, bool, bool) {
	len := len(buffer)
	connected := false
	var server net.Conn
	var serverflag bool

	if buffer[0] == 0x05 {
		switch buffer[1] {
		case 0x01:
			server, connected, serverflag = TCPConnect(conntoupper, buffer, len, checknum, key, currentid)
		case 0x02:
			connected = TCPBind(conntoupper, buffer, len, checknum, key)
		case 0x03:
			connected = UDPAssociate(conntoupper, buffer, len, checknum, key)
		}
	}

	return server, connected, serverflag
}

//如果是代理tcp
func TCPConnect(conntoupper net.Conn, buffer []byte, len int, checknum uint32, key []byte, currentid string) (net.Conn, bool, bool) {
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
		respdata, _ := utils.ConstructPayload(utils.AdminId, "", "DATA", "SOCKSDATARESP", " ", string([]byte{0x05, 0x04, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}), checknum, currentid, key, false)
		conntoupper.Write(respdata)
		return server, false, false
	}

	respdata, _ := utils.ConstructPayload(utils.AdminId, "", "DATA", "SOCKSDATARESP", " ", string([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}), checknum, currentid, key, false)
	conntoupper.Write(respdata)

	return server, true, true
}

//转发流量
func Proxyhttp(conntoupper net.Conn, server net.Conn, checknum uint32, key []byte, currentid string) error {
	serverbuffer := make([]byte, 20480)

	for {
		len, err := server.Read(serverbuffer)
		if err != nil {
			server.Close()
			return err
		}
		respdata, _ := utils.ConstructPayload(utils.AdminId, "", "DATA", "SOCKSDATARESP", " ", string(serverbuffer[:len]), checknum, currentid, key, false)
		conntoupper.Write(respdata)
	}
}

func TCPBind(client net.Conn, buffer []byte, len int, checknum uint32, AESKey []byte) bool {
	fmt.Println("Not ready") //limited use, add to Todo
	return false
}

func UDPAssociate(client net.Conn, buffer []byte, len int, checknum uint32, AESKey []byte) bool {
	fmt.Println("Not ready") //limited use, add to Todo
	return false
}
