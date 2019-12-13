package socks

import (
	"fmt"
	"io"
	"net"
	"strconv"

	"github.com/sirupsen/logrus"
)

func AuthClient(conn net.Conn, username string, secret string) {
	buffer := make([]byte, 1024)
	for {
		_, err := conn.Read(buffer)
		if err != nil {
			logrus.Error(err)
			return
		}
		if buffer[0] == 0x05 {
			if buffer[2] == 0x02 {
				conn.Write([]byte{0x05, 0x02})
				_, err := conn.Read(buffer)
				if err != nil {
					logrus.Error(err)
					return
				}

				ulen := int(buffer[1])
				slen := int(buffer[2+ulen])
				clientname := string(buffer[2 : 2+ulen])
				clientpass := string(buffer[3+ulen : 3+ulen+slen])

				if clientname != username || clientpass != secret {
					logrus.Error("Illegal client!")
					conn.Write([]byte{0x01, 0x01})
					conn.Close()
					return
				} else {
					conn.Write([]byte{0x01, 0x00})
					go Proxyhttp(conn)
					return
				}
			} else if buffer[2] == 0x00 && (username == "" || secret == "") {
				conn.Write([]byte{0x05, 0x00})
				go Proxyhttp(conn)
				return
			} else {
				conn.Close()
				logrus.Error("Illegal client!")
				return
			}
		} else {
			logrus.Error("Not socks5 message!")
			return
		}
	}
}

func Proxyhttp(conn net.Conn) {
	buffer := make([]byte, 409600)
	len, _ := conn.Read(buffer)
	if buffer[0] == 0x05 {
		switch buffer[1] {
		case 0x01:
			go TcpConnect(conn, buffer, len)
		case 0x02:
			go TcpBind(conn, buffer, len)
		case 0x03:
			go UdpAssociate(conn, buffer, len)
		}
	}
}

func TcpConnect(client net.Conn, buffer []byte, len int) {
	host := ""
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
		client.Write([]byte{0x05, 0x04, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
		logrus.Error("Cannot connect to remote server")
		return
	}
	client.Write([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
	go io.Copy(server, client)
	io.Copy(client, server)
	defer server.Close()
	defer client.Close()
}

func TcpBind(client net.Conn, buffer []byte, len int) {
	fmt.Println("Not ready") //limited use, add to Todo
}

func UdpAssociate(client net.Conn, buffer []byte, len int) {
	fmt.Println("Not ready") //limited use, add to Todo
}
