package admin

import (
	"Stowaway/common"
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var (
	AdminCommandChan = make(chan []string)
	Nodes            = make(map[uint32]string)
)

/*-------------------------控制台相关代码--------------------------*/
// 启动控制台
func Controlpanel() {
	inputReader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("(%s) >> ", *CliStatus)
		input, err := inputReader.ReadString('\n')
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		command := strings.Replace(input, "\n", "", -1)
		execCommand := strings.Split(command, " ")
		AdminCommandChan <- execCommand

		<-ReadyChange
		<-IsShellMode
	}
}

/*-------------------------节点拓扑相关代码--------------------------*/
// 显示节点拓扑信息
func ShowChain() {
	if StartNode != "0.0.0.0" {
		fmt.Printf("StartNode:[1] %s\n", StartNode)
		for Nodeid, Nodeaddress := range Nodes {
			id := fmt.Sprint(Nodeid)
			fmt.Printf("Nodes [%s]: %s\n", id, Nodeaddress)
		}
	} else {
		fmt.Println("There is no agent connected!")
	}

}

// 将节点加入拓扑
func AddToChain() {
	for {
		newNode := <-NodesReadyToadd
		for key, value := range newNode {
			Nodes[key] = value
		}
	}
}

/*-------------------------Socks5功能相关代码--------------------------*/
// 启动socks5 for client
func StartSocksServiceForClient(command []string, startNodeControlConn net.Conn, nodeID uint32) {
	var err error
	var ClientNum uint32 = 0
	socksPort := command[1]
	checkPort, _ := strconv.Atoi(socksPort)
	if checkPort <= 0 || checkPort > 65535 {
		log.Println("[*]Port Illegal!")
		return
	}

	socks5Addr := fmt.Sprintf("0.0.0.0:%s", socksPort)
	SocksListenerForClient, err = net.Listen("tcp", socks5Addr)
	if err != nil {
		respCommand, _ := common.ConstructCommand("SOCKSOFF", " ", nodeID, AESKey)
		_, err = startNodeControlConn.Write(respCommand)
		if err != nil {
			log.Println("[*]Cannot stop agent's socks service,check the connection!")
		}
		log.Println("[*]Cannot listen this port!")
		return
	}
	for {
		conn, err := SocksListenerForClient.Accept()
		if err != nil {
			log.Println("[*]Socks service stoped")
			return
		}
		ClientSockets.Lock()
		ClientSockets.ClientSocketsMap[ClientNum] = conn
		ClientSockets.Unlock()
		ClientSockets.RLock()
		go HandleNewSocksConn(ClientSockets.ClientSocketsMap[ClientNum], ClientNum, nodeID)
		ClientSockets.RUnlock()
		ClientNum++
	}
}

func HandleNewSocksConn(clientsocks net.Conn, num uint32, nodeID uint32) {
	buffer := make([]byte, 10240)
	for {
		len, err := clientsocks.Read(buffer)
		if err != nil {
			clientsocks.Close()
			return
		} else {
			respData, _ := common.ConstructDataResult(nodeID, num, " ", "SOCKSDATA", string(buffer[:len]), AESKey, 0)
			DataConn.Write(respData)
		}
	}
}

/*-------------------------Ssh功能启动相关代码--------------------------*/
// 发送ssh开启命令
func StartSSHService(startNodeControlConn net.Conn, info []string, nodeid uint32) {
	information := fmt.Sprintf("%s::%s::%s", info[1], info[2], info[3])
	sshCommand, _ := common.ConstructCommand("SSH", information, nodeid, AESKey)
	startNodeControlConn.Write(sshCommand)
}

/*-------------------------功能控制相关代码--------------------------*/
//发送下线消息
func SendOffLineToStartNode(startNodeControlConn net.Conn) {
	respCommand, _ := common.ConstructCommand("ADMINOFFLINE", "", 1, AESKey)
	_, err := startNodeControlConn.Write(respCommand)
	if err != nil {
		log.Println("[*]Startnode seems offline!Message cannot be transmitted")
		return
	}
}

//捕捉退出信号
func MonitorCtrlC(startNodeControlConn net.Conn) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan
	SendOffLineToStartNode(startNodeControlConn)
	time.Sleep(2 * time.Second)
	os.Exit(1)
}
