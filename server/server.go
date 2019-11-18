package server

import (
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

type loServer struct {
	controlPort string
}

var portChan = make(chan string)
var protocolChan = make(chan string)
var localChan = make(chan net.Conn)

func init() {
}

func newServer(c *cli.Context) {
	protocol := c.String("protocol")
	listenPort := c.String("port")
	secret := c.String("secret")

	listenStatus := startListen(listenPort, secret, protocol)

	if listenStatus != nil {
		logrus.Print(listenStatus)
		os.Exit(1)
	}
}

func startListen(listenPort string, secret string, protocol string) error {
	localAddr := fmt.Sprintf("0.0.0.0:%s", listenPort)
	localListener, err := net.Listen(protocol, localAddr)
	if err != nil {
		return fmt.Errorf("Cannot listen %s", localAddr)
	}
	for {
		conn, err := localListener.Accept()
		if err != nil {
			return fmt.Errorf("Cannot read data from socket")
		}
		logrus.Printf("New Client: %s\n", conn.RemoteAddr().String())
		go handleConnection(conn, secret)
		go dial(conn)
		go sliceStream(conn)
	}
}

func handleConnection(conn net.Conn, secret string) {

	buf := make([]byte, 4096000)
	len, err := conn.Read(buf)
	if err != nil {
		logrus.Errorf("Client %s disconnected!", conn.RemoteAddr().String())

	}

	clientSecret := strings.Split(string(buf[0:len]), ":::")[0]
	requestPort := strings.Split(string(buf[0:len]), ":::")[1]
	protocol := strings.Split(string(buf[0:len]), ":::")[2]

	if clientSecret == secret {
		logrus.Info("Auth success")
		conn.Write([]byte("Auth success"))
		portChan <- requestPort
		protocolChan <- protocol
	} else {
		logrus.Error("Auth Failed!")
		conn.Write([]byte("Auth failed"))
		conn.Close()

	}

}

func proxyStream(read, write net.Conn) {

	var buffer = make([]byte, 4096000)
	for {
		readTemp, err := read.Read(buffer)
		if err != nil {
			break
		}
		readTemp, err = write.Write(buffer[:readTemp])
		if err != nil {
			break
		}
	}
	defer read.Close()
	defer write.Close()

}

func dial(sock net.Conn) {
	requestPort := <-portChan
	requestProtocol := <-protocolChan

	remoteAddr := fmt.Sprintf("127.0.0.1:%s", requestPort)
	conn, err := net.Dial(requestProtocol, remoteAddr)
	if err != nil {
		logrus.Error("Cannot dial to localport")
		os.Exit(1)
	}
	localChan <- conn

}

func sliceStream(conn net.Conn) {
	write, ok := <-localChan
	if ok {
		go proxyStream(conn, write)
		go proxyStream(write, conn)
	}
}
