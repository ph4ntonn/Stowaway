package socks5

import (
	"fmt"
	"io"
	"net"
	"os"
	"strconv"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

func newSocks5(c *cli.Context) {
	username := c.String("username")
	password := c.String("secret")
	port := c.String("port")
	protocol := c.String("protocol")

	checkPort, err := strconv.Atoi(port)
	if (checkPort > 65535 || checkPort < 0) && err != nil {
		logrus.Error("Port should be between 0 and 65535")
		os.Exit(1)
	}

	listenSetting := fmt.Sprintf("0.0.0.0:%s", port)

	status := startListening(listenSetting, protocol, username, password)

	if status != nil {
		fmt.Println(status)
		os.Exit(1)
	}
}

func startListening(listenSetting string, protocol string, username string, secret string) error {
	localListener, err := net.Listen(protocol, listenSetting)
	if err != nil {
		return fmt.Errorf("Cannot listen %s", listenSetting)
	}
	logrus.Infof("Socks5 server is now working on %s ......", listenSetting)
	for {
		conn, err := localListener.Accept()
		if err != nil {
			return fmt.Errorf("Cannot read data from socket")
		}
		go authClient(conn, username, secret)
	}
}

func authClient(conn net.Conn, username string, secret string) {
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
					return
				} else {
					conn.Write([]byte{0x01, 0x00})
					go proxyhttp(conn)
					return
				}
			} else if buffer[2] == 0x00 && (username == "" || secret == "") {
				conn.Write([]byte{0x05, 0x00})
				go proxyhttp(conn)
				return
			} else {
				logrus.Error("Illegal client!")
				return
			}
		} else {
			logrus.Error("Not socks5 message!")
			return
		}
	}
}

func proxyhttp(conn net.Conn) {
	buffer := make([]byte, 409600)
	len, _ := conn.Read(buffer)
	if buffer[0] == 0x05 {
		switch buffer[1] {
		case 0x01:
			go tcpConnect(conn, buffer, len)
		case 0x02:
			go tcpBind(conn, buffer, len)
		case 0x03:
			go udpAssociate(conn, buffer, len)
		}
	}
}

func tcpConnect(client net.Conn, buffer []byte, len int) {
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

func tcpBind(client net.Conn, buffer []byte, len int) {
	fmt.Println("Not ready")
}

func udpAssociate(client net.Conn, buffer []byte, len int) {
	fmt.Println("Not ready")
}
