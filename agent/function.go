package agent

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"Stowaway/node"
	"Stowaway/utils"
)

//一些agent端的零碎功能代码

/*-------------------------节点发送自身信息功能相关代码--------------------------*/

// SendInfo 发送自身信息
func SendInfo(nodeID string) {
	info := utils.GetInfoViaSystem()
	respCommand, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "MYINFO", " ", info, 0, nodeID, AgentStatus.AESKey, false)
	AgentStuff.ProxyChan.ProxyChanToUpperNode <- respCommand
}

// SendNote 发送自身备忘
func SendNote(nodeID string) {
	respCommand, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "MYNOTE", " ", AgentStatus.NodeNote, 0, nodeID, AgentStatus.AESKey, false)
	AgentStuff.ProxyChan.ProxyChanToUpperNode <- respCommand
}

/*-------------------------startnode重连功能相关代码--------------------------*/

// TryReconnect 重连操作
func TryReconnect(gap string, monitor string, listenPort string) {
	lag, _ := strconv.Atoi(gap)

	for {
		//等待指定的时间
		time.Sleep(time.Duration(lag) * time.Second)
		//尝试连接admin端
		controlConnToAdmin, _, err := node.StartNodeConn(monitor, listenPort, AgentStatus.Nodeid, AgentStatus.AESKey)
		if err != nil {
			fmt.Println("[*]Admin seems still down")
		} else {
			fmt.Println("[*]Admin up! Reconnect successful!")
			ConnToAdmin = controlConnToAdmin
			return
		}
	}
}

// AdminOffline admin下线后startnode操作
func AdminOffline(reConn, monitor, listenPort string, passive bool) {
	log.Println("[*]Admin seems offline!")
	if reConn != "0" && reConn != "" && !passive { //当是主动重连时
		ClearAllConn()
		AgentStuff.SocksDataChanMap = utils.NewUint32ChanStrMap()
		if AgentStatus.NotLastOne {
			BroadCast("CLEAR")
		}
		TryReconnect(reConn, monitor, listenPort)
		if AgentStatus.NotLastOne {
			BroadCast("RECONN")
		}
	} else if passive { //被动时（包括被动以及端口复用下）
		ClearAllConn()
		AgentStuff.SocksDataChanMap = utils.NewUint32ChanStrMap()
		if AgentStatus.NotLastOne {
			BroadCast("CLEAR")
		}
		<-AgentStatus.ReConnCome
		if AgentStatus.NotLastOne {
			BroadCast("RECONN")
		}
	} else {
		os.Exit(0)
	}
}

/*-------------------------普通节点等待重连相关代码--------------------------*/

// WaitingAdmin 节点间连接断开时，等待重连的代码
func WaitingAdmin(nodeID string) {
	//清理工作
	ClearAllConn()
	AgentStuff.SocksDataChanMap = utils.NewUint32ChanStrMap()
	if AgentStatus.NotLastOne {
		BroadCast("CLEAR")
	}
	//等待重连
	ConnToAdmin = <-node.NodeStuff.Adminconn
	respCommand, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "RECONNID", " ", "", 0, nodeID, AgentStatus.AESKey, false)
	AgentStuff.ProxyChan.ProxyChanToUpperNode <- respCommand
	if AgentStatus.NotLastOne {
		BroadCast("RECONN")
	}
	node.NodeStuff.Offline = false
}

// PrepareForReOnlineNode 等待重连时，用来供上一个节点起HandleConnFromLowerNode函数
func PrepareForReOnlineNode() {
	for {
		nodeid := <-node.NodeStuff.ReOnlineID
		conn := <-node.NodeStuff.ReOnlineConn
		//如果此节点没有启动过HandleConnToLowerNode函数，启动之
		if AgentStatus.NotLastOne == false {
			AgentStuff.ProxyChan.ProxyChanToLowerNode = make(chan *utils.PassToLowerNodeData)
			go HandleConnToLowerNode()
		}
		AgentStatus.NotLastOne = true
		//记录此节点，启动HandleConnFromLowerNode
		node.NodeInfo.LowerNode.Lock()
		node.NodeInfo.LowerNode.Payload[nodeid] = conn
		node.NodeInfo.LowerNode.Unlock()
		go HandleConnFromLowerNode(conn, AgentStatus.Nodeid, nodeid)
		node.NodeStuff.PrepareForReOnlineNodeReady <- true
	}
}

/*-------------------------清除现存连接及发送FIN信号相关代码--------------------------*/

// ClearAllConn 当admin下线后，清除并关闭所有现存的socket
func ClearAllConn() {
	AgentStuff.CurrentSocks5Conn.Lock()
	for key, conn := range AgentStuff.CurrentSocks5Conn.Payload {
		err := conn.Close()
		if err != nil {
		}
		delete(AgentStuff.CurrentSocks5Conn.Payload, key)
	}
	AgentStuff.CurrentSocks5Conn.Unlock()

	AgentStuff.SocksDataChanMap.Lock()
	for key, _ := range AgentStuff.SocksDataChanMap.Payload {
		if !utils.IsClosed(AgentStuff.SocksDataChanMap.Payload[key]) {
			close(AgentStuff.SocksDataChanMap.Payload[key])
		}
		delete(AgentStuff.SocksDataChanMap.Payload, key)
	}
	AgentStuff.SocksDataChanMap.Unlock()

	AgentStuff.PortFowardMap.Lock()
	for key, _ := range AgentStuff.PortFowardMap.Payload {
		if !utils.IsClosed(AgentStuff.PortFowardMap.Payload[key]) {
			close(AgentStuff.PortFowardMap.Payload[key])
		}
		delete(AgentStuff.PortFowardMap.Payload, key)
	}
	AgentStuff.PortFowardMap.Unlock()

	AgentStuff.ForwardConnMap.Lock()
	for key, conn := range AgentStuff.ForwardConnMap.Payload {
		err := conn.Close()
		if err != nil {
		}
		delete(AgentStuff.ForwardConnMap.Payload, key)
	}
	AgentStuff.ForwardConnMap.Unlock()

	AgentStuff.ReflectConnMap.Lock()
	for key, conn := range AgentStuff.ReflectConnMap.Payload {
		err := conn.Close()
		if err != nil {
		}
		delete(AgentStuff.ReflectConnMap.Payload, key)
	}
	AgentStuff.ReflectConnMap.Unlock()

	for _, listener := range CurrentPortReflectListener {
		listener.Close()
	}

}

/*-------------------------路由相关代码--------------------------*/

// ChangeRoute 查找需要递交的路由
func ChangeRoute(AdminData *utils.Payload) string {
	route := AdminData.Route
	//找到下一个节点id号
	routes := strings.Split(route, ":")
	selected := routes[0]
	//修改route字段，向下一级递交
	AdminData.Route = strings.Join(routes[1:], ":")
	//返回下一个节点id
	return selected
}

/*-------------------------广播相关代码--------------------------*/

// BroadCast 广播消息
func BroadCast(command string) {
	var readyToBroadCast []string

	node.NodeInfo.LowerNode.Lock()
	for nodeid, _ := range node.NodeInfo.LowerNode.Payload {
		if nodeid == utils.AdminId {
			continue
		}
		readyToBroadCast = append(readyToBroadCast, nodeid)
	}
	node.NodeInfo.LowerNode.Unlock()

	for _, nodeid := range readyToBroadCast {
		mess, _ := utils.ConstructPayload(nodeid, "", "COMMAND", command, " ", " ", 0, AgentStatus.Nodeid, AgentStatus.AESKey, false)
		passToLowerData := utils.NewPassToLowerNodeData()
		passToLowerData.Data = mess
		passToLowerData.Route = nodeid
		AgentStuff.ProxyChan.ProxyChanToLowerNode <- passToLowerData
	}
}

/*-------------------------监听相关代码--------------------------*/

// TestListen 尝试监听
func TestListen(port string) error {
	var CAN_NOT_LISTEN = errors.New("cannot listen")

	listenAddr := fmt.Sprintf("0.0.0.0:%s", port)
	//admin下发listen命令时，尝试监听，不成功则返回错误
	testListener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return CAN_NOT_LISTEN
	}
	//成功则关闭此listener
	testListener.Close()
	//返回nil，启动真正的listen程序
	return nil
}

/*-------------------------程序控制相关代码--------------------------*/

// WaitForExit 捕捉程序退出信号
func WaitForExit() {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, os.Kill, syscall.SIGHUP)
	<-signalChan
	os.Exit(0)
}
