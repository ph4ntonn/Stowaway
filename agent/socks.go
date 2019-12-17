package agent

import (
	"Stowaway/common"
	"Stowaway/node"
	"Stowaway/socks"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

func StartSocks(controlConnToAdmin net.Conn, socksPort, socksUsername, socksPass string) {
	var err error
	tempport, _ := strconv.Atoi(socksPort)
	socksPort = strconv.Itoa(tempport + int(NODEID))
	socksAddr := fmt.Sprintf("0.0.0.0:%s", socksPort)
	SocksServer, err = net.Listen("tcp", socksAddr)

	if err != nil {
		socksstartmess, _ := common.ConstructCommand("SOCKSRESP", "FAILED", NODEID, AESKey)
		controlConnToAdmin.Write(socksstartmess)
		fmt.Println(err)
		return
	} else {
		socksstartmess, _ := common.ConstructCommand("SOCKSRESP", "SUCCESS", NODEID, AESKey)
		controlConnToAdmin.Write(socksstartmess)
	}

	for {
		socksConn, err := SocksServer.Accept()
		if err != nil {
			return
		}
		go socks.AuthClient(socksConn, socksUsername, socksPass)
	}
}

func StartSocksProxy(info string) {
	requestport := strings.Split(info, ":")[0]
	tempport, _ := strconv.Atoi(requestport)
	socksPort := strconv.Itoa(tempport + int(NODEID))
	socksAddr := fmt.Sprintf("0.0.0.0:%s", socksPort)
	socksServer, _ := net.Listen("tcp", socksAddr)
	for {
		socksConn, _ := socksServer.Accept()
		go SocksProxy(socksConn, socksPort)
	}
}

func SocksProxy(client net.Conn, socksPort string) {
	var socksAddr string
	tempport, _ := strconv.Atoi(socksPort)
	socksPort = strconv.Itoa(tempport + 1)
	socksAddr = fmt.Sprintf("%s:%s", node.PeerNode, socksPort)
	socksproxyconn, err := net.Dial("tcp", socksAddr)
	if err != nil {
		logrus.Error("Cannot connect to socks server")
	}
	go io.Copy(client, socksproxyconn)
	io.Copy(socksproxyconn, client)
	defer client.Close()
	defer socksproxyconn.Close()
}
