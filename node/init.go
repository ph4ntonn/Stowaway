package node

import (
	"Stowaway/common"
	"fmt"
	"log"
	"net"
	"os"
)

var (
	NodeInfo                    *common.NodeInfo
	ControlConnForLowerNodeChan chan net.Conn //下级节点控制信道
	Adminconn                   chan net.Conn
	ReOnlineConn                chan net.Conn
	NewNodeMessageChan          chan []byte //新节点加入消息
	IsAdmin                     chan bool   //分辨连接是属于admin还是agent
	PrepareForReOnlineNodeReady chan bool
	ReOnlineId                  chan string
	Offline                     bool //判断当前状态是否是掉线状态
)

func init() {
	ControlConnForLowerNodeChan = make(chan net.Conn, 1)
	Adminconn = make(chan net.Conn, 1)
	ReOnlineConn = make(chan net.Conn, 1)
	NewNodeMessageChan = make(chan []byte, 1)
	IsAdmin = make(chan bool, 1)
	PrepareForReOnlineNodeReady = make(chan bool, 1)
	ReOnlineId = make(chan string, 1)
	Offline = false
	NodeInfo = common.NewNodeInfo()
}

//初始化一个节点连接操作
func StartNodeConn(monitor string, listenPort string, nodeID string, key []byte) (net.Conn, string, error) {
	controlConnToUpperNode, err := net.Dial("tcp", monitor)
	if err != nil {
		log.Println("[*]Connection refused!")
		return controlConnToUpperNode, "", err
	}
	helloMess, _ := common.ConstructPayload(nodeID, "", "COMMAND", "STOWAWAYAGENT", " ", " ", 0, common.AdminId, key, false)
	controlConnToUpperNode.Write(helloMess)

	common.ExtractPayload(controlConnToUpperNode, key, common.AdminId, true)

	respcommand, _ := common.ConstructPayload(nodeID, "", "COMMAND", "INIT", " ", listenPort, 0, common.AdminId, key, false) //主动向上级节点发送初始信息
	_, err = controlConnToUpperNode.Write(respcommand)
	if err != nil {
		log.Printf("[*]Error occured: %s", err)
		return controlConnToUpperNode, "", err
	}
	//等待admin为其分配一个id号
	for {
		command, _ := common.ExtractPayload(controlConnToUpperNode, key, common.AdminId, true)
		switch command.Command {
		case "ID":
			nodeID = command.NodeId
			return controlConnToUpperNode, nodeID, nil
		}
	}
}

//初始化节点监听操作
func StartNodeListen(listenPort string, NodeId string, key []byte) {
	var NewNodeMessage []byte

	if listenPort == "" { //如果没有port，直接退出
		return
	}

	listenAddr := fmt.Sprintf("0.0.0.0:%s", listenPort)
	WaitingForLowerNode, err := net.Listen("tcp", listenAddr)

	if err != nil {
		log.Printf("[*]Cannot listen on port %s", listenPort)
		os.Exit(0)
	}

	for {
		ConnToLowerNode, err := WaitingForLowerNode.Accept()
		if err != nil {
			log.Println("[*]", err)
			return
		}
		for i := 0; i < 2; i++ {
			command, _ := common.ExtractPayload(ConnToLowerNode, key, common.AdminId, true)
			switch command.Command {
			case "STOWAWAYADMIN":
				respcommand, _ := common.ConstructPayload(NodeId, "", "COMMAND", "INIT", " ", listenPort, 0, common.AdminId, key, false)
				ConnToLowerNode.Write(respcommand)
			case "ID":
				ControlConnForLowerNodeChan <- ConnToLowerNode
				NewNodeMessageChan <- NewNodeMessage
				IsAdmin <- true
			case "REONLINESUC":
				Adminconn <- ConnToLowerNode
			case "STOWAWAYAGENT":
				if !Offline {
					NewNodeMessage, _ = common.ConstructPayload(NodeId, "", "COMMAND", "CONFIRM", " ", " ", 0, NodeId, key, false)
					ConnToLowerNode.Write(NewNodeMessage)
				} else {
					respcommand, _ := common.ConstructPayload(NodeId, "", "COMMAND", "REONLINE", " ", listenPort, 0, NodeId, key, false)
					ConnToLowerNode.Write(respcommand)
				}
			case "INIT":
				//告知admin新节点消息
				NewNodeMessage, _ = common.ConstructPayload(common.AdminId, "", "COMMAND", "NEW", " ", ConnToLowerNode.RemoteAddr().String(), 0, NodeId, key, false)
				NodeInfo.LowerNode.Payload[common.AdminId] = ConnToLowerNode //将这个socket用0号位暂存，等待admin分配完id后再将其放入对应的位置
				ControlConnForLowerNodeChan <- ConnToLowerNode
				NewNodeMessageChan <- NewNodeMessage //被连接后不终止监听，继续等待可能的后续节点连接，以此组成树状结构
				IsAdmin <- false
			}
		}
	}
}

//connect命令代码
func ConnectNextNode(target string, nodeid string, key []byte) bool {
	var NewNodeMessage []byte

	controlConnToNextNode, err := net.Dial("tcp", target)

	if err != nil {
		return false
	}

	helloMess, _ := common.ConstructPayload(nodeid, "", "COMMAND", "STOWAWAYAGENT", " ", " ", 0, common.AdminId, key, false)
	controlConnToNextNode.Write(helloMess)
	for {
		command, err := common.ExtractPayload(controlConnToNextNode, key, common.AdminId, true)
		if err != nil {
			log.Println("[*]", err)
			return false
		}
		switch command.Command {
		case "INIT":
			//类似与上面
			NewNodeMessage, _ = common.ConstructPayload(common.AdminId, "", "COMMAND", "NEW", " ", controlConnToNextNode.RemoteAddr().String(), 0, nodeid, key, false)
			NodeInfo.LowerNode.Payload[common.AdminId] = controlConnToNextNode
			ControlConnForLowerNodeChan <- controlConnToNextNode
			NewNodeMessageChan <- NewNodeMessage
			IsAdmin <- false
			return true
		case "REONLINE":
			//普通节点重连
			ReOnlineId <- command.CurrentId
			ReOnlineConn <- controlConnToNextNode
			<-PrepareForReOnlineNodeReady
			NewNodeMessage, _ = common.ConstructPayload(nodeid, "", "COMMAND", "REONLINESUC", " ", " ", 0, nodeid, key, false)
			controlConnToNextNode.Write(NewNodeMessage)
			return true
		}
	}
}

//被动模式下startnode接收admin重连 && 普通节点被动启动等待上级节点主动连接
func AcceptConnFromUpperNode(listenPort string, nodeid string, key []byte) (net.Conn, string) {
	listenAddr := fmt.Sprintf("0.0.0.0:%s", listenPort)
	WaitingForConn, err := net.Listen("tcp", listenAddr)

	if err != nil {
		log.Printf("[*]Cannot listen on port %s", listenPort)
		os.Exit(0)
	}
	for {
		Comingconn, err := WaitingForConn.Accept()
		if err != nil {
			log.Println("[*]", err)
			continue
		}
		common.ExtractPayload(Comingconn, key, common.AdminId, true)
		respcommand, _ := common.ConstructPayload(nodeid, "", "COMMAND", "INIT", " ", listenPort, 0, common.AdminId, key, false)
		Comingconn.Write(respcommand)
		command, _ := common.ExtractPayload(Comingconn, key, common.AdminId, true) //等待分配id
		if command.Command == "ID" {
			nodeid = command.NodeId
			WaitingForConn.Close()
			return Comingconn, nodeid
		}

	}

}
