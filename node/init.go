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
	NewNodeMessageChan          chan []byte   //新节点加入消息
)

func init() {
	ControlConnForLowerNodeChan = make(chan net.Conn, 1)
	NewNodeMessageChan = make(chan []byte, 1)
	NodeInfo = common.NewNodeInfo()
}

//初始化一个节点连接操作
func StartNodeConn(monitor string, listenPort string, nodeID uint32, key []byte) (net.Conn, uint32, error) {
	controlConnToUpperNode, err := net.Dial("tcp", monitor)
	if err != nil {
		log.Println("[*]Connection refused!")
		return controlConnToUpperNode, 11235, err
	}
	respcommand, _ := common.ConstructPayload(nodeID, "", "COMMAND", "INIT", " ", listenPort, 0, 0, key, false) //主动向上级节点发送初始信息
	_, err = controlConnToUpperNode.Write(respcommand)
	if err != nil {
		log.Printf("[*]Error occured: %s", err)
		return controlConnToUpperNode, 11235, err
	}
	//等待admin为其分配一个id号
	for {
		command, _ := common.ExtractPayload(controlConnToUpperNode, key, 0, true)
		switch command.Command {
		case "ID":
			nodeID = command.NodeId
			return controlConnToUpperNode, nodeID, nil
		}
	}
}

//初始化节点监听操作
func StartNodeListen(listenPort string, NodeId uint32, key []byte) {
	var NewNodeMessage []byte

	if listenPort == "" { //如果没有port，直接退出
		return
	}

	listenAddr := fmt.Sprintf("0.0.0.0:%s", listenPort)
	WaitingForLowerNode, err := net.Listen("tcp", listenAddr)

	if err != nil {
		log.Printf("[*]Cannot listen on port %s", listenPort)
		os.Exit(1)
	}

	for {
		ConnToLowerNode, err := WaitingForLowerNode.Accept()
		if err != nil {
			log.Println("[*]", err)
			return
		}
		command, err := common.ExtractPayload(ConnToLowerNode, key, 0, true)
		if err != nil {
			log.Println("[*]", err)
			return
		}
		if command.Command == "INIT" {
			//告知admin新节点消息
			NewNodeMessage, _ = common.ConstructPayload(0, "", "COMMAND", "NEW", " ", ConnToLowerNode.RemoteAddr().String(), 0, NodeId, key, false)
		} else {
			continue
		}
		NodeInfo.LowerNode.Payload[0] = ConnToLowerNode //将这个socket用0号位暂存，等待admin分配完id后再将其放入对应的位置
		ControlConnForLowerNodeChan <- ConnToLowerNode
		NewNodeMessageChan <- NewNodeMessage //被连接后不终止监听，继续等待可能的后续节点连接，以此组成树状结构
	}
}

//connect命令代码
func ConnectNextNode(target string, nodeid uint32, key []byte) bool {
	var NewNodeMessage []byte

	controlConnToNextNode, err := net.Dial("tcp", target)

	if err != nil {
		return false
	}

	for {
		command, err := common.ExtractPayload(controlConnToNextNode, key, 0, true)
		if err != nil {
			log.Println("[*]", err)
			return false
		}
		switch command.Command {
		case "INIT":
			//类似与上面
			NewNodeMessage, _ = common.ConstructPayload(0, "", "COMMAND", "NEW", " ", controlConnToNextNode.RemoteAddr().String(), 0, nodeid, key, false)
			NodeInfo.LowerNode.Payload[0] = controlConnToNextNode
			ControlConnForLowerNodeChan <- controlConnToNextNode
			NewNodeMessageChan <- NewNodeMessage
			return true
		}
	}
}

//被动模式下startnode接收admin重连 && 普通节点被动启动等待上级节点主动连接
func AcceptConnFromUpperNode(listenPort string, nodeid uint32, key []byte) (net.Conn, uint32) {
	listenAddr := fmt.Sprintf("0.0.0.0:%s", listenPort)
	WaitingForConn, err := net.Listen("tcp", listenAddr)

	if err != nil {
		log.Printf("[*]Cannot listen on port %s", listenPort)
		os.Exit(1)
	}
	for {
		Comingconn, err := WaitingForConn.Accept()
		if err != nil {
			log.Println("[*]", err)
			continue
		}

		respcommand, _ := common.ConstructPayload(nodeid, "", "COMMAND", "INIT", " ", listenPort, 0, 0, key, false)
		Comingconn.Write(respcommand)
		command, _ := common.ExtractPayload(Comingconn, key, 0, true) //等待分配id
		if command.Command == "ID" {
			nodeid = command.NodeId
			WaitingForConn.Close()
			return Comingconn, nodeid
		}
	}

}
