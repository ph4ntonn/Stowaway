package client

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto/ecies"

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
var newbee int = 0
var heartbeatChan = make(chan bool)
var serverstatusChan = make(chan bool)

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
	heartbeat := c.Bool("heartbeat")
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

	if heartbeat {
		heartbeatTimer := time.NewTimer(time.Second * 7)
		go startheartbeat(reserver.remoteAddr, protocol, heartbeatTimer)
		go listenheartbeat(heartbeatTimer)
	}

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

	listenStatus := listenLocalPort(loserver.localPort, reserver.remoteAddr, reserver.requestPort, protocol, conn, newbee, secret)
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

func listenLocalPort(port string, remote string, requestPort string, protocol string, conn net.Conn, newbee int, secret string) error {
	for _, pro := range allowProtocol {
		if protocol == pro {
			break
		} else {
			return fmt.Errorf("Unsupported protocol")
		}
	}

	localAddr := fmt.Sprintf("0.0.0.0:%s", port)
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
		if newbee != 0 {
			conn = connectReServer(remote, secret, protocol, requestPort)
		}
		newbee = 1
		go handleConnection(fromsth, conn)
		go handleConnection(conn, fromsth)
	}
}

func handleConnection(read, write net.Conn) {
	_, err := io.Copy(write, read)
	if err != nil {
		logrus.Errorf("Fatal error:%s", err)
		return
	}
	defer read.Close()
	defer write.Close()
}

func connectReServer(remoteAddr string, secret string, protocol string, requestPort string) net.Conn {
	conn, err := net.Dial(protocol, remoteAddr)
	if err != nil {
		logrus.Error("Cannot connect to reServer")
		os.Exit(1)
	}

	buffer := make([]byte, 409600)
	time := time.Now().UnixNano() / 1e6
	strtime := strconv.FormatInt(time, 10)
	conn.Write([]byte("Client Hello!"))

	for {
		num, err := conn.Read(buffer)
		if err == nil {
			if string(buffer[0:num]) == "Auth failed" {
				logrus.Error("Auth failed, please check your password")
				conn.Close()
				os.Exit(1)
			} else if strings.Split(string(buffer[0:num]), ":::")[0] == "Publickey" {
				publickey := strings.Split(string(buffer[0:num]), ":::")[1]
				preparingPB := ToECDSAPub([]byte(publickey))
				readyPB := ecies.ImportECDSAPublic(preparingPB)
				tempMessage, _ := ECCEncrypt([]byte(secret+":::"+requestPort+":::"+protocol+":::"+strtime), *readyPB)
				combineMessage := [][]byte{[]byte("Legal:"), tempMessage}
				authMessage := bytes.Join(combineMessage, []byte{})
				conn.Write(authMessage)
			} else {
				logrus.Println("Authenication success")
			}
		} else {
			logrus.Error("Auth Failed, socket closed by peer")
			conn.Close()
			os.Exit(1)
		}
		if num > 0 && string(buffer[0:num]) == "Auth success" {
			logrus.Println("Ready to proxy.....")
			return conn
		}
	}
}

func startheartbeat(remote string, protocol string, heartbeatTimer *time.Timer) {
	buffer := make([]byte, 1024)
	addr := strings.Split(remote, ":")[0]
	port := strings.Split(remote, ":")[1]
	heartbeatport, _ := strconv.Atoi(port)
	heartbeatport++
	remoteheartbeat := addr + ":" + strconv.Itoa(heartbeatport)
	conn, err := net.Dial(protocol, remoteheartbeat)
	if err != nil {
		logrus.Errorf("Cannot connect the reserver's port %s", strconv.Itoa(heartbeatport))
		logrus.Error("Turning off heartbeat function......")
		heartbeatChan <- false
		return
	} else {
		heartbeatChan <- true
		conn.Write([]byte("Ping"))
	}
	for {
		len, err := conn.Read(buffer)
		if err != nil {
			logrus.Errorf("Server seems down, Waiting for heartbeat......")
			<-serverstatusChan
			conn.Close()
			break
		}
		if string(buffer[:len]) == "Alive" {
			heartbeatTimer.Reset(time.Second * 7)
		}
	}
}

func listenheartbeat(heartbeatTimer *time.Timer) {
	startornot := <-heartbeatChan
	if startornot == false {
		return
	}
	<-heartbeatTimer.C
	for loop := 0; loop < 5; loop++ {
		logrus.Error("Server seems down! Please check the server if it's still up ")
		time.Sleep(time.Duration(2) * time.Second)
	}
	logrus.Error("Server down! Please reconnect!")
	logrus.Error("Client Exiting......")
	serverstatusChan <- true
	os.Exit(1)
}
