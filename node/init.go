package node

import (
	"Stowaway/utils"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"
)

var (
	NodeInfo  *utils.NodeInfo
	NodeStuff *utils.NodeStuff
)

func init() {
	NodeStuff = utils.NewNodeStuff()
	NodeInfo = utils.NewNodeInfo()
}

/*-------------------------一般模式下初始化节点代码--------------------------*/
//初始化一个节点连接操作
func StartNodeConn(monitor string, listenPort string, nodeID string, key []byte) (net.Conn, string, error) {
	controlConnToUpperNode, err := net.Dial("tcp", monitor)
	if err != nil {
		log.Println("[*]Connection refused!")
		return controlConnToUpperNode, "", err
	}

	err = SendSecret(controlConnToUpperNode, key)
	if err != nil {
		log.Println("[*]Connection refused!")
		return controlConnToUpperNode, "", err
	}

	helloMess, _ := utils.ConstructPayload(nodeID, "", "COMMAND", "STOWAWAYAGENT", " ", " ", 0, utils.AdminId, key, false)
	controlConnToUpperNode.Write(helloMess)

	utils.ExtractPayload(controlConnToUpperNode, key, utils.AdminId, true)

	respcommand, _ := utils.ConstructPayload(nodeID, "", "COMMAND", "INIT", " ", listenPort, 0, utils.AdminId, key, false) //主动向上级节点发送初始信息
	_, err = controlConnToUpperNode.Write(respcommand)
	if err != nil {
		log.Printf("[*]Error occured: %s", err)
		return controlConnToUpperNode, "", err
	}
	//等待admin为其分配一个id号
	for {
		command, _ := utils.ExtractPayload(controlConnToUpperNode, key, utils.AdminId, true)
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

		err = CheckSecret(ConnToLowerNode, key)
		if err != nil {
			continue
		}

		for i := 0; i < 2; i++ {
			command, _ := utils.ExtractPayload(ConnToLowerNode, key, utils.AdminId, true)
			switch command.Command {
			case "STOWAWAYADMIN":
				respcommand, _ := utils.ConstructPayload(NodeId, "", "COMMAND", "INIT", " ", listenPort, 0, utils.AdminId, key, false)
				ConnToLowerNode.Write(respcommand)
			case "ID":
				NodeStuff.ControlConnForLowerNodeChan <- ConnToLowerNode
				NodeStuff.NewNodeMessageChan <- NewNodeMessage
				NodeStuff.IsAdmin <- true
			case "REONLINESUC":
				NodeStuff.Adminconn <- ConnToLowerNode
			case "STOWAWAYAGENT":
				if !NodeStuff.Offline {
					NewNodeMessage, _ = utils.ConstructPayload(NodeId, "", "COMMAND", "CONFIRM", " ", " ", 0, NodeId, key, false)
					ConnToLowerNode.Write(NewNodeMessage)
				} else {
					respcommand, _ := utils.ConstructPayload(NodeId, "", "COMMAND", "REONLINE", " ", listenPort, 0, NodeId, key, false)
					ConnToLowerNode.Write(respcommand)
				}
			case "INIT":
				//告知admin新节点消息
				NewNodeMessage, _ = utils.ConstructPayload(utils.AdminId, "", "COMMAND", "NEW", " ", ConnToLowerNode.RemoteAddr().String(), 0, NodeId, key, false)
				NodeInfo.LowerNode.Payload[utils.AdminId] = ConnToLowerNode //将这个socket用0号位暂存，等待admin分配完id后再将其放入对应的位置
				NodeStuff.ControlConnForLowerNodeChan <- ConnToLowerNode
				NodeStuff.NewNodeMessageChan <- NewNodeMessage //被连接后不终止监听，继续等待可能的后续节点连接，以此组成树状结构
				NodeStuff.IsAdmin <- false
			}
		}
	}
}

/*-------------------------节点主动connect模式代码--------------------------*/
//connect命令代码
func ConnectNextNode(target string, nodeid string, key []byte) bool {
	controlConnToNextNode, err := net.Dial("tcp", target)

	if err != nil {
		return false
	}

	err = SendSecret(controlConnToNextNode, key)
	if err != nil {
		log.Println("[*]", err)
		return false
	}

	helloMess, _ := utils.ConstructPayload(nodeid, "", "COMMAND", "STOWAWAYAGENT", " ", " ", 0, utils.AdminId, key, false)
	controlConnToNextNode.Write(helloMess)

	for {
		command, err := utils.ExtractPayload(controlConnToNextNode, key, utils.AdminId, true)
		if err != nil {
			log.Println("[*]", err)
			return false
		}

		switch command.Command {
		case "INIT":
			//类似与上面
			NewNodeMessage, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "NEW", " ", controlConnToNextNode.RemoteAddr().String(), 0, nodeid, key, false)
			NodeInfo.LowerNode.Payload[utils.AdminId] = controlConnToNextNode
			NodeStuff.ControlConnForLowerNodeChan <- controlConnToNextNode
			NodeStuff.NewNodeMessageChan <- NewNodeMessage
			NodeStuff.IsAdmin <- false
			return true
		case "REONLINE":
			//普通节点重连
			NodeStuff.ReOnlineId <- command.CurrentId
			NodeStuff.ReOnlineConn <- controlConnToNextNode
			<-NodeStuff.PrepareForReOnlineNodeReady
			NewNodeMessage, _ := utils.ConstructPayload(nodeid, "", "COMMAND", "REONLINESUC", " ", " ", 0, nodeid, key, false)
			controlConnToNextNode.Write(NewNodeMessage)
			return true
		}
	}
}

/*-------------------------节点被动模式下功能代码--------------------------*/
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

		err = CheckSecret(Comingconn, key)
		if err != nil {
			continue
		}

		utils.ExtractPayload(Comingconn, key, utils.AdminId, true)

		respcommand, _ := utils.ConstructPayload(nodeid, "", "COMMAND", "INIT", " ", listenPort, 0, utils.AdminId, key, false)
		Comingconn.Write(respcommand)

		command, _ := utils.ExtractPayload(Comingconn, key, utils.AdminId, true) //等待分配id
		if command.Command == "ID" {
			nodeid = command.NodeId
			WaitingForConn.Close()
			return Comingconn, nodeid
		}

	}

}

/*-------------------------一般模式及被动模式下(除了reuse模式)节点互相校验代码--------------------------*/
//发送secret值
func SendSecret(conn net.Conn, key []byte) error {
	var NOT_VALID = errors.New("not valid")
	defer conn.SetReadDeadline(time.Time{})
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	secret := utils.GetStringMd5(string(key))
	conn.Write([]byte(secret[:16]))

	buffer := make([]byte, 16)
	count, err := io.ReadFull(conn, buffer)

	if timeouterr, ok := err.(net.Error); ok && timeouterr.Timeout() {
		conn.Close()
		return NOT_VALID
	}

	if err != nil {
		conn.Close()
		return NOT_VALID
	}

	if string(buffer[:count]) == secret[:16] {
		return nil
	}
	conn.Close()
	return NOT_VALID
}

//检查secret值，在连接建立前测试合法性
func CheckSecret(conn net.Conn, key []byte) error {
	var NOT_VALID = errors.New("not valid")
	defer conn.SetReadDeadline(time.Time{})
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	secret := utils.GetStringMd5(string(key))

	buffer := make([]byte, 16)
	count, err := io.ReadFull(conn, buffer)

	if timeouterr, ok := err.(net.Error); ok && timeouterr.Timeout() {
		conn.Close()
		return NOT_VALID
	}

	if err != nil {
		conn.Close()
		return NOT_VALID
	}

	if string(buffer[:count]) == secret[:16] {
		conn.Write([]byte(secret[:16]))
		return nil
	}
	conn.Close()
	return NOT_VALID
}
