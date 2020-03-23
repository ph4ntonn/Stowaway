package node

import (
	"Stowaway/common"
	"fmt"
	"log"
	"net"
	"os"
)

var (
	ControlConnForLowerNodeChan = make(chan net.Conn, 1) //下级节点控制信道
	NewNodeMessageChan          = make(chan []byte, 1)   //新节点加入消息
	ConnectStatusChan           = make(chan []byte, 1)
	AdminOrAgent                = make(chan string, 1)
)

//初始化一个节点连接操作
func StartNodeConn(monitor string, listenPort string, nodeID uint32, key []byte) (net.Conn, uint32, error) {
	controlConnToUpperNode, err := net.Dial("tcp", monitor)
	if err != nil {
		log.Println("[*]Connection refused!")
		return controlConnToUpperNode, 11235, err
	}
	respcommand, err := common.ConstructPayload(nodeID, "COMMAND", "INIT", " ", listenPort, 0, 0, key, false)
	if err != nil {
		log.Printf("[*]Error occured: %s", err)
	}
	_, err = controlConnToUpperNode.Write(respcommand)
	if err != nil {
		log.Printf("[*]Error occured: %s", err)
	}
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
func StartNodeListen(listenPort string, NodeId uint32, key []byte, reconn bool, single bool, firstNodeStatus bool) {
	var NewNodeMessage []byte

	if listenPort == "" {
		return
	}
	if single { //如果passive重连状态下只有startnode一个节点，没有后续节点的话，直接交给AcceptConnFromUpperNode函数
		for {
			controlConnToAdmin, _ := AcceptConnFromUpperNode(listenPort, NodeId, key)
			AdminOrAgent <- "admin"
			ControlConnForLowerNodeChan <- controlConnToAdmin
		}
	}

	//如果passive重连状态下startnode后有节点连接，先执行后续节点的初始化操作，再交给AcceptConnFromUpperNode函数
	listenAddr := fmt.Sprintf("0.0.0.0:%s", listenPort)
	WaitingForLowerNode, err := net.Listen("tcp", listenAddr)

	if err != nil {
		log.Printf("[*]Cannot listen on port %s", listenPort)
		os.Exit(1)
	}

	for !firstNodeStatus {
		ConnToLowerNode, err := WaitingForLowerNode.Accept() //判断一下是否是合法连接
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
			respNodeID := NodeId + 1
			respCommand, _ := common.ConstructPayload(respNodeID, "COMMAND", "ID", " ", " ", 0, 0, key, false)
			_, err := ConnToLowerNode.Write(respCommand)
			NewNodeMessage, _ = common.ConstructPayload(0, "COMMAND", "NEW", " ", ConnToLowerNode.RemoteAddr().String(), 0, NodeId, key, false)
			if err != nil {
				log.Println("[*]", err)
				return
			}
		} else {
			respCommand, _ := common.ConstructPayload(command.NodeId, "COMMAND", "ID", " ", " ", 0, 0, key, false)
			_, err := ConnToLowerNode.Write(respCommand)
			if err != nil {
				log.Println("[*]", err)
				return
			}
		}
		AdminOrAgent <- "agent"
		ControlConnForLowerNodeChan <- ConnToLowerNode
		NewNodeMessageChan <- NewNodeMessage
		break
	}
	WaitingForLowerNode.Close()
	if reconn {
		for {
			controlConnToAdmin, _ := AcceptConnFromUpperNode(listenPort, NodeId, key)
			AdminOrAgent <- "admin"
			ControlConnForLowerNodeChan <- controlConnToAdmin
		}
	}
}

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
			respNodeID := nodeid + 1
			respCommand, _ := common.ConstructPayload(respNodeID, "COMMAND", "ID", " ", " ", 0, 0, key, false)
			_, err := controlConnToNextNode.Write(respCommand)
			if err != nil {
				log.Println("[*]", err)
				return false
			}
			NewNodeMessage, _ = common.ConstructPayload(0, "COMMAND", "NEW", " ", controlConnToNextNode.RemoteAddr().String(), 0, nodeid, key, false)
			AdminOrAgent <- "agent"
			ControlConnForLowerNodeChan <- controlConnToNextNode
			NewNodeMessageChan <- NewNodeMessage
			return true
		}
	}
}

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

		respcommand, _ := common.ConstructPayload(nodeid, "COMMAND", "INIT", " ", listenPort, 0, 0, key, false)
		Comingconn.Write(respcommand)
		command, _ := common.ExtractPayload(Comingconn, key, 0, true)
		if command.Command == "ID" {
			nodeid = command.NodeId
			WaitingForConn.Close()
			return Comingconn, nodeid
		}
	}

}
