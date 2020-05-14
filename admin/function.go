package admin

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"

	"Stowaway/utils"
)

var (
	AdminStuff     *utils.AdminStuff
	NodeStatus     *utils.NodeStatus
	ForwardStatus  *utils.ForwardStatus
	ReflectConnMap *utils.Uint32ConnMap
	PortReflectMap *utils.Uint32ChanStrMap
)
var WaitForFindAll chan bool

func init() {
	ReflectConnMap = utils.NewUint32ConnMap()
	PortReflectMap = utils.NewUint32ChanStrMap()
	NodeStatus = utils.NewNodeStatus()
	ForwardStatus = utils.NewForwardStatus()
	AdminStuff = utils.NewAdminStuff()
	WaitForFindAll = make(chan bool, 1)
}

//admin端功能及零碎代码
//admin端相关功能代码都比较简单一些，大多数功能实现都在agent端
//所以admin端就写在一个文件里了，分太多文件也不好
//agent端分文件写

/*-------------------------Socks5功能相关代码--------------------------*/

// StartSocksServiceForClient 启动socks5 for client
func StartSocksServiceForClient(command []string, startNodeConn net.Conn, nodeID string) {
	route := utils.GetInfoViaLockMap(Route, nodeID).(string)

	socksPort := command[1]
	checkPort, _ := strconv.Atoi(socksPort)
	if checkPort <= 0 || checkPort > 65535 {
		log.Println("[*]Port Illegal!")
		return
	}
	//监听指定的socks5端口
	socks5Addr := fmt.Sprintf("0.0.0.0:%s", socksPort)
	socksListenerForClient, err := net.Listen("tcp", socks5Addr)
	if err != nil {
		err = utils.ConstructPayloadAndSend(startNodeConn, nodeID, route, "COMMAND", "SOCKSOFF", " ", " ", 0, utils.AdminId, AdminStatus.AESKey, false)
		if err != nil {
			log.Println("[*]Cannot stop agent's socks service,check the connection!")
		}
		log.Println("[*]Cannot listen this port!")
		return
	}
	//把此监听地址记录
	AdminStuff.Lock()
	AdminStuff.SocksListenerForClient.Payload[nodeID] = append(AdminStuff.SocksListenerForClient.Payload[nodeID], socksListenerForClient)
	AdminStuff.Unlock()

	for {
		//开始监听
		conn, err := socksListenerForClient.Accept()
		if err != nil {
			log.Println("[*]Socks service stopped")
			return
		}
		//有请求时记录此socket，并启动HandleNewSocksConn对此socket进行处理
		ClientSockets.Lock()
		ClientSockets.Payload[AdminStuff.SocksNum] = conn
		go HandleNewSocksConn(startNodeConn, ClientSockets.Payload[AdminStuff.SocksNum], AdminStuff.SocksNum, nodeID)
		ClientSockets.Unlock()

		AdminStuff.Lock()
		AdminStuff.SocksMapping.Payload[nodeID] = append(AdminStuff.SocksMapping.Payload[nodeID], AdminStuff.SocksNum)
		AdminStuff.SocksNum++
		AdminStuff.Unlock()

	}
}

// HandleNewSocksConn 处理每一个单个的socks socket
func HandleNewSocksConn(startNodeConn net.Conn, clientsocks net.Conn, num uint32, nodeID string) {
	route := utils.GetInfoViaLockMap(Route, nodeID).(string)

	buffer := make([]byte, 20480)

	for {
		len, err := clientsocks.Read(buffer)
		if err != nil {
			clientsocks.Close()
			utils.ConstructPayloadAndSend(startNodeConn, nodeID, route, "DATA", "FIN", " ", " ", num, utils.AdminId, AdminStatus.AESKey, false)
			return
		}
		utils.ConstructPayloadAndSend(startNodeConn, nodeID, route, "DATA", "SOCKSDATA", " ", string(buffer[:len]), num, utils.AdminId, AdminStatus.AESKey, false)
	}
}

// StopSocks stopsocks命令执行代码
func StopSocks() {
	AdminStuff.Lock()
	defer AdminStuff.Unlock()

	//检查是否启动过socks服务
	if len(AdminStuff.SocksListenerForClient.Payload) == 0 {
		log.Println("[*]You have never started socks service!")
	} else {
		//启动过则遍历，关闭所有listener
		for nodeid, listeners := range AdminStuff.SocksListenerForClient.Payload {
			for _, listener := range listeners {
				listener.Close()
			}
			delete(AdminStuff.SocksListenerForClient.Payload, nodeid)
		}

		log.Println("[*]All socks listeners are closed successfully!")
		//关闭所有socks连接
		for key, conn := range ClientSockets.Payload {
			conn.Close()
			delete(ClientSockets.Payload, key)
		}

		log.Println("[*]All socks sockets are closed successfully!")

	}
}

/*-------------------------Ssh功能启动相关代码--------------------------*/

// StartSSHService 发送ssh开启命令
func StartSSHService(startNodeConn net.Conn, info []string, nodeid string, method string) {
	information := fmt.Sprintf("%s:::%s:::%s:::%s", info[0], info[1], info[2], method)
	SendPayloadViaRoute(startNodeConn, nodeid, Route.Route[nodeid], "COMMAND", "SSH", " ", information, 0, utils.AdminId, AdminStatus.AESKey, false)
}

// CheckKeyFile 检查私钥文件是否存在
func CheckKeyFile(file string) []byte {
	buffer, err := ioutil.ReadFile(file)
	if err != nil {
		return nil
	}
	return buffer
}

/*-------------------------SshTunnel功能启动相关代码--------------------------*/

// SendSSHTunnel 发送SshTunnel开启命令
func SendSSHTunnel(startNodeConn net.Conn, info []string, nodeid string, method string) {
	information := fmt.Sprintf("%s:::%s:::%s:::%s:::%s", info[0], info[1], info[2], info[3], method)
	SendPayloadViaRoute(startNodeConn, nodeid, Route.Route[nodeid], "COMMAND", "SSHTUNNEL", " ", information, 0, utils.AdminId, AdminStatus.AESKey, false)
}

/*-------------------------Port Forward功能启动相关代码--------------------------*/

// HandleForwardPort 发送forward开启命令
func HandleForwardPort(forwardconn net.Conn, target string, startNodeConn net.Conn, num uint32, nodeid string) {
	route := utils.GetInfoViaLockMap(Route, nodeid).(string)

	utils.ConstructPayloadAndSend(startNodeConn, nodeid, route, "DATA", "FORWARD", " ", target, num, utils.AdminId, AdminStatus.AESKey, false)

	buffer := make([]byte, 20480)
	for {
		len, err := forwardconn.Read(buffer)
		if err != nil {
			utils.ConstructPayloadAndSend(startNodeConn, nodeid, route, "DATA", "FORWARDFIN", " ", " ", num, utils.AdminId, AdminStatus.AESKey, false)
			return
		}
		utils.ConstructPayloadAndSend(startNodeConn, nodeid, route, "DATA", "FORWARDDATA", " ", string(buffer[:len]), num, utils.AdminId, AdminStatus.AESKey, false)
	}
}

// StartPortForwardForClient 启动forward服务
func StartPortForwardForClient(info []string, startNodeConn net.Conn, nodeid string, AESKey []byte) {
	TestIfValid("FORWARDTEST", startNodeConn, info[2], nodeid)
	//如果指定的forward端口监听正常
	if <-ForwardStatus.ForwardIsValid {
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
	//记录监听的listener
	ForwardStatus.Lock()
	ForwardStatus.CurrentPortForwardListener.Payload[nodeid] = append(ForwardStatus.CurrentPortForwardListener.Payload[nodeid], forwardListenerForClient)
	ForwardStatus.Unlock()

	for {
		conn, err := forwardListenerForClient.Accept()
		if err != nil {
			log.Println("[*]PortForward service stopped")
			return
		}

		PortForWardMap.Lock()
		PortForWardMap.Payload[ForwardStatus.ForwardNum] = conn
		go HandleForwardPort(PortForWardMap.Payload[ForwardStatus.ForwardNum], info[2], startNodeConn, ForwardStatus.ForwardNum, nodeid)
		PortForWardMap.Unlock()

		ForwardStatus.Lock()
		ForwardStatus.ForwardMapping.Payload[nodeid] = append(ForwardStatus.ForwardMapping.Payload[nodeid], ForwardStatus.ForwardNum)
		ForwardStatus.ForwardNum++
		ForwardStatus.Unlock()

	}
}

// StopForward 停止所有forward功能
func StopForward() {
	ForwardStatus.Lock()
	defer ForwardStatus.Unlock()
	//逻辑同socks
	if len(ForwardStatus.CurrentPortForwardListener.Payload) == 0 {
		log.Println("[*]You have never started forward service!")
	} else {
		for nodeid, listeners := range ForwardStatus.CurrentPortForwardListener.Payload {
			for _, listener := range listeners {
				err := listener.Close()
				if err != nil {
					log.Println("[*]One forward listener seems already closed.Won't close it again...")
				}
			}
			delete(ForwardStatus.CurrentPortForwardListener.Payload, nodeid)
		}

		log.Println("[*]All forward listeners are closed successfully!")

		for key, conn := range PortForWardMap.Payload {
			conn.Close()
			delete(PortForWardMap.Payload, key)
		}

		log.Println("[*]All forward sockets are closed successfully!")

	}
}

/*-------------------------Reflect Port相关代码--------------------------*/

// StartReflectForClient 测试agent是否能够监听
func StartReflectForClient(info []string, startNodeConn net.Conn, nodeid string, AESKey []byte) {
	tempInfo := fmt.Sprintf("%s:%s", info[1], info[2])
	TestIfValid("REFLECTTEST", startNodeConn, tempInfo, nodeid)
}

// TryReflect 尝试reflect
func TryReflect(startNodeConn net.Conn, nodeid string, id uint32, port string) {
	target := fmt.Sprintf("127.0.0.1:%s", port)
	reflectConn, err := net.Dial("tcp", target)

	if err == nil {
		ReflectConnMap.Lock()
		ReflectConnMap.Payload[id] = reflectConn
		ReflectConnMap.Unlock()
	} else {
		SendPayloadViaRoute(startNodeConn, nodeid, Route.Route[nodeid], "DATA", "REFLECTTIMEOUT", " ", " ", id, utils.AdminId, AdminStatus.AESKey, false)
		return
	}
}

// HandleReflect 处理每一个reflect连接
func HandleReflect(startNodeConn net.Conn, reflectDataChan chan string, num uint32, nodeid string) {
	reflectConn := utils.GetInfoViaLockMap(ReflectConnMap, num).(net.Conn)
	route := utils.GetInfoViaLockMap(Route, nodeid).(string)

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
		serverBuffer := make([]byte, 20480)
		for {
			len, err := reflectConn.Read(serverBuffer)
			if err != nil {
				utils.ConstructPayloadAndSend(startNodeConn, nodeid, route, "DATA", "REFLECTOFFLINE", " ", " ", num, utils.AdminId, AdminStatus.AESKey, false)
				return
			}
			utils.ConstructPayloadAndSend(startNodeConn, nodeid, route, "DATA", "REFLECTDATARESP", " ", string(serverBuffer[:len]), num, utils.AdminId, AdminStatus.AESKey, false)
		}
	}()
}

// StopReflect 停止所有reflect服务
func StopReflect(startNodeConn net.Conn, nodeid string) {
	fmt.Println("[*]Stop command has been sent")
	SendPayloadViaRoute(startNodeConn, nodeid, Route.Route[nodeid], "COMMAND", "STOPREFLECT", " ", " ", 0, utils.AdminId, AdminStatus.AESKey, false)
}

/*-------------------------一些功能相关代码--------------------------*/

// SendPayloadViaRoute 获取route后发送payload
func SendPayloadViaRoute(conn net.Conn, nodeid string, route string, ptype string, command string, fileSliceNum string, info string, clientid uint32, currentid string, key []byte, pass bool) {
	Route.Lock()
	utils.ConstructPayloadAndSend(conn, nodeid, route, ptype, command, fileSliceNum, info, clientid, currentid, key, pass)
	Route.Unlock()
}

// CommandContinue 继续接收命令
func CommandContinue() {
	AdminStatus.ReadyChange <- true
	AdminStatus.IsShellMode <- true
}

// ReadChoice 读取选项
func ReadChoice() string {
	inputReader := bufio.NewReader(os.Stdin)
	input, _ := inputReader.ReadString('\n')
	input = CheckInput(input)
	return input
}

// TestIfValid 测试是否端口可用
func TestIfValid(commandtype string, startNodeConn net.Conn, target string, nodeid string) {
	SendPayloadViaRoute(startNodeConn, nodeid, Route.Route[nodeid], "COMMAND", commandtype, " ", target, 0, utils.AdminId, AdminStatus.AESKey, false)
}

// AnalysisInfo 拆分Info
func AnalysisInfo(info string) (string, string) {
	spiltInfo := strings.Split(info, ":::")
	upperNode := spiltInfo[0]
	ip := spiltInfo[1]
	return ip, upperNode
}

// CheckInput 替换无效字符
func CheckInput(input string) string {
	if runtime.GOOS == "windows" {
		input = strings.Replace(input, "\r\n", "", -1)
		input = strings.Replace(input, " ", "", -1)
	} else {
		input = strings.Replace(input, "\n", "", -1)
		input = strings.Replace(input, " ", "", -1)
	}
	return input
}

/*-------------------------控制相关代码--------------------------*/

// CloseAll 当有一个节点下线，强制关闭该节点及其子节点对应的服务
func CloseAll(id string) {
	readyToDel := FindAll(id)
	AdminStuff.Lock()
	for _, nodeid := range readyToDel {
		if _, ok := AdminStuff.SocksListenerForClient.Payload[nodeid]; ok {
			for _, listener := range AdminStuff.SocksListenerForClient.Payload[nodeid] {
				err := listener.Close()
				if err != nil {
				}
			}
		}
	}
	ClientSockets.Lock()
	for _, nodeid := range readyToDel {
		for _, connid := range AdminStuff.SocksMapping.Payload[nodeid] {
			if _, ok := ClientSockets.Payload[connid]; ok {
				ClientSockets.Payload[connid].Close()
				delete(ClientSockets.Payload, connid)
			}
		}
		AdminStuff.SocksMapping.Payload[nodeid] = make([]uint32, 0)
	}
	ClientSockets.Unlock()
	AdminStuff.Unlock()

	ForwardStatus.Lock()
	for _, nodeid := range readyToDel {
		if _, ok := ForwardStatus.CurrentPortForwardListener.Payload[nodeid]; ok {
			for _, listener := range ForwardStatus.CurrentPortForwardListener.Payload[nodeid] {
				err := listener.Close()
				if err != nil {
				}
			}
		}
	}
	PortForWardMap.Lock()
	for _, nodeid := range readyToDel {
		for _, connid := range ForwardStatus.ForwardMapping.Payload[nodeid] {
			if _, ok := PortForWardMap.Payload[connid]; ok {
				PortForWardMap.Payload[connid].Close()
				delete(PortForWardMap.Payload, connid)
			}
		}
		ForwardStatus.ForwardMapping.Payload[nodeid] = make([]uint32, 0)
	}
	PortForWardMap.Unlock()
	ForwardStatus.Unlock()
}
