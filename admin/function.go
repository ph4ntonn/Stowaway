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
	Nodes            map[uint32]string
	Nodenote         map[uint32]string
	AdminCommandChan chan []string
	ForwardIsValid   chan bool

	ForwardNum                 uint32 //Forward socket编号，必须全局，不然stopforward后，无法获得最新的编号
	ClientNum                  uint32
	CurrentPortForwardListener []net.Listener //记录一下当前开启的port-forward listener，以供关闭

	ReflectConnMap *common.Uint32ConnMap
	PortReflectMap *common.Uint32ChanStrMap
)

func init() {
	ForwardNum = 0
	ClientNum = 0
	Nodes = make(map[uint32]string)
	Nodenote = make(map[uint32]string)
	AdminCommandChan = make(chan []string)
	ForwardIsValid = make(chan bool, 1)
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
		fmt.Printf("StartNode:[1] %s   note:%s\n", StartNode, Nodenote[1])
		for Nodeid, Nodeaddress := range Nodes {
			id := fmt.Sprint(Nodeid)
			fmt.Printf("Nodes [%s]: %s   note:%s\n", id, Nodeaddress, Nodenote[Nodeid])
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

//为node添加note
func AddNote(data []string, nodeid uint32) bool {
	info := ""
	data = data[1:len(data)]
	for _, i := range data {
		info = info + " " + i
	}
	if _, ok := Nodenote[nodeid]; ok {
		Nodenote[nodeid] = info
		return true
	}
	return false
}

//为node删除note
func DelNote(nodeid uint32) bool {
	if _, ok := Nodenote[nodeid]; ok {
		Nodenote[nodeid] = ""
		return true
	}
	return false
}

/*-------------------------Socks5功能相关代码--------------------------*/
// 启动socks5 for client
func StartSocksServiceForClient(command []string, startNodeConn net.Conn, nodeID uint32) {
	var err error
	socksPort := command[1]
	checkPort, _ := strconv.Atoi(socksPort)
	if checkPort <= 0 || checkPort > 65535 {
		log.Println("[*]Port Illegal!")
		return
	}

	socks5Addr := fmt.Sprintf("0.0.0.0:%s", socksPort)
	socksListenerForClient, err := net.Listen("tcp", socks5Addr)
	if err != nil {
		respCommand, _ := common.ConstructPayload(nodeID, "COMMAND", "SOCKSOFF", " ", " ", 0, 0, AESKey, false)
		_, err = startNodeConn.Write(respCommand)
		if err != nil {
			log.Println("[*]Cannot stop agent's socks service,check the connection!")
		}
		log.Println("[*]Cannot listen this port!")
		return
	}
	SocksListenerForClient = append(SocksListenerForClient, socksListenerForClient)
	for {
		conn, err := socksListenerForClient.Accept()
		if err != nil {
			log.Println("[*]Socks service stoped")
			return
		}
		ClientSockets.Lock()
		ClientSockets.Payload[ClientNum] = conn
		go HandleNewSocksConn(startNodeConn, ClientSockets.Payload[ClientNum], ClientNum, nodeID)
		ClientNum++
		ClientSockets.Unlock()
	}
}

//处理每一个单个的socks socket
func HandleNewSocksConn(startNodeConn net.Conn, clientsocks net.Conn, num uint32, nodeID uint32) {
	buffer := make([]byte, 10240)
	for {
		len, err := clientsocks.Read(buffer)
		if err != nil {
			clientsocks.Close()
			finMessage, _ := common.ConstructPayload(nodeID, "DATA", "FIN", " ", " ", num, 0, AESKey, false)
			startNodeConn.Write(finMessage)
			return
		} else {
			respData, _ := common.ConstructPayload(nodeID, "DATA", "SOCKSDATA", " ", string(buffer[:len]), num, 0, AESKey, false)
			startNodeConn.Write(respData)
		}
	}
}

/*-------------------------Ssh功能启动相关代码--------------------------*/
// 发送ssh开启命令
func StartSSHService(startNodeConn net.Conn, info []string, nodeid uint32) {
	information := fmt.Sprintf("%s::%s::%s", info[1], info[2], info[3])
	sshCommand, _ := common.ConstructPayload(nodeid, "COMMAND", "SSH", " ", information, 0, 0, AESKey, false)
	startNodeConn.Write(sshCommand)
}

/*-------------------------Port Forward功能启动相关代码--------------------------*/
// 发送forward开启命令
func HandleForwardPort(forwardconn net.Conn, target string, startNodeConn net.Conn, num uint32, nodeid uint32) {
	forwardCommand, _ := common.ConstructPayload(nodeid, "DATA", "FORWARD", " ", target, num, 0, AESKey, false)
	startNodeConn.Write(forwardCommand)

	buffer := make([]byte, 10240)
	for {
		len, err := forwardconn.Read(buffer)
		if err != nil {
			finMessage, _ := common.ConstructPayload(nodeid, "DATA", "FORWARDFIN", " ", " ", num, 0, AESKey, false)
			startNodeConn.Write(finMessage)
			return
		} else {
			respData, _ := common.ConstructPayload(nodeid, "DATA", "FORWARDDATA", " ", string(buffer[:len]), num, 0, AESKey, false)
			startNodeConn.Write(respData)
		}
	}
}

func StartPortForwardForClient(info []string, startNodeConn net.Conn, nodeid uint32, AESKey []byte) {
	TestIfValid("FORWARDTEST", startNodeConn, info[2], nodeid)
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
		go HandleForwardPort(PortForWardMap.Payload[ForwardNum], info[2], startNodeConn, ForwardNum, nodeid)
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
func StartReflectForClient(info []string, startNodeConn net.Conn, nodeid uint32, AESKey []byte) {
	tempInfo := fmt.Sprintf("%s:%s", info[1], info[2])
	TestIfValid("REFLECTTEST", startNodeConn, tempInfo, nodeid)
}

func TryReflect(startNodeConn net.Conn, nodeid uint32, id uint32, port string) {
	target := fmt.Sprintf("0.0.0.0:%s", port)
	reflectConn, err := net.Dial("tcp", target)
	if err == nil {
		ReflectConnMap.Lock()
		ReflectConnMap.Payload[id] = reflectConn
		ReflectConnMap.Unlock()
	} else {
		respdata, _ := common.ConstructPayload(nodeid, "DATA", "REFLECTTIMEOUT", " ", " ", id, 0, AESKey, false)
		startNodeConn.Write(respdata)
		return
	}
}

func HandleReflect(startNodeConn net.Conn, reflectDataChan chan string, num uint32, nodeid uint32) {
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
				respdata, _ := common.ConstructPayload(nodeid, "DATA", "REFLECTOFFLINE", " ", " ", num, 0, AESKey, false)
				startNodeConn.Write(respdata)
				return
			}
			respdata, _ := common.ConstructPayload(nodeid, "DATA", "REFLECTDATARESP", " ", string(serverbuffer[:len]), num, 0, AESKey, false)
			startNodeConn.Write(respdata)
		}
	}()
}

func StopReflect(startNodeConn net.Conn, nodeId uint32) {
	fmt.Println("[*]Stop command has been sent")
	command, _ := common.ConstructPayload(nodeId, "COMMAND", "STOPREFLECT", " ", " ", 0, 0, AESKey, false)
	startNodeConn.Write(command)
}

/*-------------------------测试相关代码--------------------------*/
//测试是否端口可用
func TestIfValid(commandtype string, startNodeConn net.Conn, target string, nodeid uint32) {
	command, _ := common.ConstructPayload(nodeid, "COMMAND", commandtype, " ", target, 0, 0, AESKey, false)
	startNodeConn.Write(command)
}

/*-------------------------功能控制相关代码--------------------------*/
//发送下线消息
func SendOffLineToStartNode(startNodeConn net.Conn) {
	respCommand, _ := common.ConstructPayload(1, "COMMAND", "ADMINOFFLINE", " ", " ", 0, 0, AESKey, false)
	_, err := startNodeConn.Write(respCommand)
	if err != nil {
		log.Println("[*]Startnode seems offline!Message cannot be transmitted")
		return
	}
}

//捕捉退出信号
func MonitorCtrlC(startNodeConn net.Conn) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan
	SendOffLineToStartNode(startNodeConn)
	time.Sleep(3 * time.Second)
	os.Exit(1)
}

//当有一个节点下线，强制关闭所有的服务
func CloseAll() {
	ClientSockets.Lock()
	for key, conn := range ClientSockets.Payload {
		conn.Close()
		delete(ClientSockets.Payload, key)
	}
	ClientSockets.Unlock()
	for _, listener := range SocksListenerForClient {
		err := listener.Close()
		if err != nil {
		}
	}

	StopForward()
	PortForWardMap.Lock()
	for key, conn := range PortForWardMap.Payload {
		conn.Close()
		delete(PortForWardMap.Payload, key)
	}
	PortForWardMap.Unlock()

	ReflectConnMap.Lock()
	for key, conn := range ReflectConnMap.Payload {
		conn.Close()
		delete(ReflectConnMap.Payload, key)
	}
	ReflectConnMap.Unlock()

}
