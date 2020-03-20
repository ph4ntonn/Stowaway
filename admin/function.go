package admin

import (
	"Stowaway/common"
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var (
	Nodes            = make(map[uint32]string)
	AdminCommandChan = make(chan []string)
	ForwardIsValid   = make(chan bool, 1)

	ForwardNum                 uint32         //Forward socket编号，必须全局，不然stopforward后，无法获得最新的编号
	CurrentPortForwardListener []net.Listener //记录一下当前开启的port-forward listener，以供关闭

	ReflectConnMap *common.Uint32ConnMap
	PortReflectMap *common.Uint32ChanStrMap
)

func init() {
	ForwardNum = 0
	ReflectConnMap = common.NewUint32ConnMap()
	PortReflectMap = common.NewUint32ChanStrMap()
}

/*-------------------------控制台相关代码--------------------------*/
// 启动控制台
func Controlpanel() {
	inputReader := bufio.NewReader(os.Stdin)
	var command string
	for {
		fmt.Printf("(%s) >> ", *CliStatus)
		input, err := inputReader.ReadString('\n')
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		if runtime.GOOS == "windows" {
			command = strings.Replace(input, "\r\n", "", -1)
		} else {
			command = strings.Replace(input, "\n", "", -1)
		}
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
		ClientSockets.Payload[ClientNum] = conn
		ClientSockets.Unlock()
		ClientSockets.RLock()
		go HandleNewSocksConn(ClientSockets.Payload[ClientNum], ClientNum, nodeID)
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
			finMessage, _ := common.ConstructDataResult(nodeID, num, " ", "FIN", " ", AESKey, 0)
			DataConn.Write(finMessage)
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

/*-------------------------Port Forward功能启动相关代码--------------------------*/
// 发送forward开启命令
func HandleForwardPort(forwardconn net.Conn, target string, dataconn net.Conn, num uint32, nodeid uint32) {
	forwardCommand, _ := common.ConstructDataResult(nodeid, num, " ", "FORWARD", target, AESKey, 0)
	dataconn.Write(forwardCommand)

	buffer := make([]byte, 10240)
	for {
		len, err := forwardconn.Read(buffer)
		if err != nil {
			forwardconn.Close()
			finMessage, _ := common.ConstructDataResult(nodeid, num, " ", "FORWARDFIN", " ", AESKey, 0)
			dataconn.Write(finMessage)
			PortForWardMap.Lock()
			if _, ok := PortForWardMap.Payload[num]; ok {
				delete(PortForWardMap.Payload, num)
			}
			PortForWardMap.Unlock()
			return
		} else {
			respData, _ := common.ConstructDataResult(nodeid, num, " ", "FORWARDDATA", string(buffer[:len]), AESKey, 0)
			dataconn.Write(respData)
		}
	}
}

func StartPortForwardForClient(info []string, dataconn net.Conn, controlconn net.Conn, nodeid uint32, AESKey []byte) {
	TestIfValid("FORWARDTEST", controlconn, info[2], nodeid)
	if <-ForwardIsValid {
	} else {
		return
	}

	localPort := info[1]
	forwardAddr := fmt.Sprintf("0.0.0.0:%s", localPort)
	forwardListenerForClient, err := net.Listen("tcp", forwardAddr)
	if err != nil {
		log.Println("[*]Cannot forward this local port!")
		return
	}

	CurrentPortForwardListener = append(CurrentPortForwardListener, forwardListenerForClient)

	for {
		conn, err := forwardListenerForClient.Accept()
		if err != nil {
			log.Println("[*]PortForward service stoped")
			return
		}
		PortForWardMap.Lock()
		PortForWardMap.Payload[ForwardNum] = conn
		PortForWardMap.Unlock()
		PortForWardMap.RLock()
		go HandleForwardPort(PortForWardMap.Payload[ForwardNum], info[2], dataconn, ForwardNum, nodeid)
		PortForWardMap.RUnlock()
		ForwardNum++
	}
}

func StopForward() {
	for _, listener := range CurrentPortForwardListener {
		listener.Close()
	}
}

/*-------------------------Reflect Port相关代码--------------------------*/
//测试agent是否能够监听
func StartReflectForClient(info []string, dataconn net.Conn, controlconn net.Conn, nodeid uint32, AESKey []byte) {
	tempInfo := fmt.Sprintf("%s:%s", info[1], info[2])
	TestIfValid("REFLECTTEST", controlconn, tempInfo, nodeid)
}

func TryReflect(dataconn net.Conn, nodeid uint32, id uint32, port string) {
	target := fmt.Sprintf("0.0.0.0:%s", port)
	reflectConn, err := net.Dial("tcp", target)
	if err == nil {
		ReflectConnMap.Lock()
		ReflectConnMap.Payload[id] = reflectConn
		ReflectConnMap.Unlock()
	} else {
		respdata, _ := common.ConstructDataResult(nodeid, id, " ", "REFLECTTIMEOUT", " ", AESKey, 0)
		dataconn.Write(respdata)
		return
	}
}

func HandleReflect(dataConn net.Conn, reflectDataChan chan string, num uint32, nodeid uint32) {
	ReflectConnMap.RLock()
	reflectConn := ReflectConnMap.Payload[num]
	ReflectConnMap.RUnlock()

	go func() {
		for {
			reflectData, ok := <-reflectDataChan
			if ok {
				reflectConn.Write([]byte(reflectData))
			} else {
				return
			}
		}
	}()

	go func() {
		serverbuffer := make([]byte, 10240)
		for {
			len, err := reflectConn.Read(serverbuffer)
			if err != nil {
				reflectConn.Close()
				respdata, _ := common.ConstructDataResult(nodeid, num, " ", "REFLECTOFFLINE", " ", AESKey, 0)
				dataConn.Write(respdata)
				ReflectConnMap.Lock()
				if _, ok := ReflectConnMap.Payload[num]; ok {
					ReflectConnMap.Payload[num].Close()
					delete(ReflectConnMap.Payload, num)
				}
				ReflectConnMap.Unlock()
				PortReflectMap.Lock()
				if _, ok := PortReflectMap.Payload[num]; ok {
					if !common.IsClosed(PortReflectMap.Payload[num]) {
						close(PortReflectMap.Payload[num])
					}
				}
				PortReflectMap.Unlock()
				return
			}
			respdata, _ := common.ConstructDataResult(nodeid, num, " ", "REFLECTDATARESP", string(serverbuffer[:len]), AESKey, 0)
			dataConn.Write(respdata)
		}
	}()
}

func StopReflect(controlConn net.Conn, nodeId uint32) {
	fmt.Println("[*]Stop command has been sent")
	command, _ := common.ConstructCommand("STOPREFLECT", " ", nodeId, AESKey)
	controlConn.Write(command)
}

/*-------------------------测试相关代码--------------------------*/
//测试是否端口可用
func TestIfValid(commandtype string, controlconn net.Conn, target string, nodeid uint32) {
	command, _ := common.ConstructCommand(commandtype, target, nodeid, AESKey)
	controlconn.Write(command)
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
