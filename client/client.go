package client

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

//Some variables
type reServer struct {
	remoteAddr  string
	requestPort string
}

type loServer struct {
	localPort string
}

var allowProtocol []string

//Function definition below
func init() {
	allowProtocol = []string{"tcp", "udp"}
}

func newClient(c *cli.Context) error {

	var secret = ""
	var reserver reServer
	var loserver loServer

	protocol := c.String("protocol")
	tunnel := c.StringSlice("tunnel")
	if c.String("secret") != "" {
		secret = c.String("secret")
	}

	for _, remote := range tunnel {
		args := strings.Split(remote, "|")
		reserver.remoteAddr = args[0]
		loserver.localPort = args[1]
		reserver.requestPort = args[2]
	}

	addrStatus := checkAddress(reserver.remoteAddr)
	portStatus := checkPort(loserver.localPort)
	remotePortStatus := checkPort(reserver.requestPort)

	if addrStatus != nil || portStatus != nil || remotePortStatus != nil {
		if addrStatus != nil {
			logrus.Println(addrStatus)
		} else if portStatus != nil {
			logrus.Println(portStatus)
		} else {
			logrus.Println(remotePortStatus)
		}
		os.Exit(1)
	} else {
		logrus.Printf("Redirect tcp stream to remote server: %s\n", reserver.remoteAddr)
	}

	logrus.Println("Begin to Auth......")
	conn := connectReServer(reserver.remoteAddr, secret, protocol, reserver.requestPort)

	//go proxyStream(conn, reserver.requestPort)

	listenStatus := listenLocalPort(loserver.localPort, reserver.remoteAddr, reserver.requestPort, protocol, conn)

	if listenStatus != nil {
		logrus.Println(listenStatus)
		os.Exit(1)
	}

	logrus.Println(secret, protocol, tunnel, reserver.remoteAddr)
	return nil
}

func checkAddress(ipv4 string) error {
	address := net.ParseIP(strings.Split(ipv4, ":")[0])
	if address != nil {
		return nil
	}

	return fmt.Errorf("Invalid ip address")

}

func checkPort(port string) error {
	po, _ := strconv.Atoi(port)
	if po <= 65535 && po > 0 {
		return nil
	}

	return fmt.Errorf("Invalid port")
}

func listenLocalPort(port string, remote string, requestPort string, protocol string, conn net.Conn) error {
	for _, pro := range allowProtocol {
		if protocol == pro {
			break
		} else {
			return fmt.Errorf("Unsupported protocol")
		}
	}

	localAddr := fmt.Sprintf("127.0.0.1:%s", port)
	localListener, err := net.Listen(protocol, localAddr)
	if err != nil {
		return fmt.Errorf("Listening port failed")
	}

	for {
		fromsth, err := localListener.Accept()
		if err != nil {
			return fmt.Errorf("Accept data failed")
		}
		logrus.Printf("New connection: %s", fromsth.RemoteAddr().String())
		go handleConnection(fromsth, conn)
		go handleConnection(conn, fromsth)
	}
}

func handleConnection(read, write net.Conn) {
	var buffer = make([]byte, 100000)
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
	// defer read.Close()
	// defer write.Close()
}

func connectReServer(remoteAddr string, secret string, protocol string, requestPort string) net.Conn {
	conn, err := net.Dial(protocol, remoteAddr)
	if err != nil {
		logrus.Error("Cannot connect to reServer")
		os.Exit(1)
	}

	buffer := make([]byte, 10000)
	conn.Write([]byte(secret + ":::" + requestPort + ":::" + protocol))
	for {
		num, err := conn.Read(buffer)
		if err == nil {
			if string(buffer[0:num]) == "Auth failed" {
				logrus.Error("Auth failed, please check your password")
				os.Exit(1)
			} else {
				logrus.Println("Authenication success")
			}
		} else {
			logrus.Error("Auth Failed, socket closed by peer")
			os.Exit(1)
		}
		if num > 0 && string(buffer[0:num]) == "Auth success" {
			logrus.Println("Ready to proxy.....")
			return conn
		}
	}
}
