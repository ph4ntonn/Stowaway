package node

import (
	"Stowaway/common"
	"Stowaway/config"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"

	reuseport "github.com/libp2p/go-reuseport"
)

//以下代码和init.go中大体相似，只是为了将改动剥离，所以单列出来

func StartNodeConnReuse(monitor string, listenPort string, nodeID string, key []byte) (net.Conn, string, error) {
	for {
		controlConnToUpperNode, err := net.Dial("tcp", monitor)
		if err != nil {
			log.Println("[*]Connection refused!")
			return controlConnToUpperNode, "", err
		}

		err = IfValid(controlConnToUpperNode)
		if err != nil {
			controlConnToUpperNode.Close()
			continue
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
}

//初始化节点监听操作
func StartNodeListenReuse(rehost, report string, NodeId string, key []byte) {
	var NewNodeMessage []byte

	if report == "" { //如果没有port，直接退出
		return
	}

	listenAddr := fmt.Sprintf("%s:%s", rehost, report)
	WaitingForLowerNode, err := reuseport.Listen("tcp", listenAddr)

	if err != nil {
		log.Printf("[*]Cannot listen on port %s", report)
		os.Exit(0)
	}

	for {
		ConnToLowerNode, err := WaitingForLowerNode.Accept()
		if err != nil {
			log.Println("[*]", err)
			return
		}

		err = CheckValid(ConnToLowerNode, true, report)
		if err != nil {
			continue
		}

		for i := 0; i < 2; i++ {
			command, _ := common.ExtractPayload(ConnToLowerNode, key, common.AdminId, true)
			switch command.Command {
			case "STOWAWAYADMIN":
				respcommand, _ := common.ConstructPayload(NodeId, "", "COMMAND", "INIT", " ", report, 0, common.AdminId, key, false)
				ConnToLowerNode.Write(respcommand)
			case "ID":
				NodeStuff.ControlConnForLowerNodeChan <- ConnToLowerNode
				NodeStuff.NewNodeMessageChan <- NewNodeMessage
				NodeStuff.IsAdmin <- true
			case "REONLINESUC":
				NodeStuff.Adminconn <- ConnToLowerNode
			case "STOWAWAYAGENT":
				if !NodeStuff.Offline {
					NewNodeMessage, _ = common.ConstructPayload(NodeId, "", "COMMAND", "CONFIRM", " ", " ", 0, NodeId, key, false)
					ConnToLowerNode.Write(NewNodeMessage)
				} else {
					respcommand, _ := common.ConstructPayload(NodeId, "", "COMMAND", "REONLINE", " ", report, 0, NodeId, key, false)
					ConnToLowerNode.Write(respcommand)
				}
			case "INIT":
				//告知admin新节点消息
				NewNodeMessage, _ = common.ConstructPayload(common.AdminId, "", "COMMAND", "NEW", " ", ConnToLowerNode.RemoteAddr().String(), 0, NodeId, key, false)
				NodeInfo.LowerNode.Payload[common.AdminId] = ConnToLowerNode //将这个socket用0号位暂存，等待admin分配完id后再将其放入对应的位置
				NodeStuff.ControlConnForLowerNodeChan <- ConnToLowerNode
				NodeStuff.NewNodeMessageChan <- NewNodeMessage //被连接后不终止监听，继续等待可能的后续节点连接，以此组成树状结构
				NodeStuff.IsAdmin <- false
			}
		}
	}
}

//connect命令代码
func ConnectNextNodeReuse(target string, nodeid string, key []byte) bool {
	for {
		controlConnToNextNode, err := net.Dial("tcp", target)

		if err != nil {
			return false
		}

		err = IfValid(controlConnToNextNode)
		if err != nil {
			controlConnToNextNode.Close()
			continue
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
				NewNodeMessage, _ := common.ConstructPayload(common.AdminId, "", "COMMAND", "NEW", " ", controlConnToNextNode.RemoteAddr().String(), 0, nodeid, key, false)
				NodeInfo.LowerNode.Payload[common.AdminId] = controlConnToNextNode
				NodeStuff.ControlConnForLowerNodeChan <- controlConnToNextNode
				NodeStuff.NewNodeMessageChan <- NewNodeMessage
				NodeStuff.IsAdmin <- false
				return true
			case "REONLINE":
				//普通节点重连
				NodeStuff.ReOnlineId <- command.CurrentId
				NodeStuff.ReOnlineConn <- controlConnToNextNode
				<-NodeStuff.PrepareForReOnlineNodeReady
				NewNodeMessage, _ := common.ConstructPayload(nodeid, "", "COMMAND", "REONLINESUC", " ", " ", 0, nodeid, key, false)
				controlConnToNextNode.Write(NewNodeMessage)
				return true
			}
		}
	}
}

//被动模式下startnode接收admin重连 && 普通节点被动启动等待上级节点主动连接
func AcceptConnFromUpperNodeReuse(rehost, report string, nodeid string, key []byte) (net.Conn, string) {
	listenAddr := fmt.Sprintf("%s:%s", rehost, report)
	WaitingForConn, err := reuseport.Listen("tcp", listenAddr)

	if err != nil {
		log.Printf("[*]Cannot reuse port %s", report)
		os.Exit(0)
	}
	for {
		Comingconn, err := WaitingForConn.Accept()
		if err != nil {
			log.Println("[*]", err)
			continue
		}

		err = CheckValid(Comingconn, true, report)
		if err != nil {
			continue
		}

		common.ExtractPayload(Comingconn, key, common.AdminId, true)

		respcommand, _ := common.ConstructPayload(nodeid, "", "COMMAND", "INIT", " ", report, 0, common.AdminId, key, false)
		Comingconn.Write(respcommand)

		command, _ := common.ExtractPayload(Comingconn, key, common.AdminId, true) //等待分配id
		if command.Command == "ID" {
			nodeid = command.NodeId
			WaitingForConn.Close()
			return Comingconn, nodeid
		}

	}

}

/*-------------------------Reuseport主要功能代码--------------------------*/
//发送特征字段
func IfValid(conn net.Conn) error {
	var NOT_VALID = errors.New("Not valid")
	conn.Write([]byte(config.VALIDMESSAGE))
	returnMess := make([]byte, 13)
	io.ReadFull(conn, returnMess)
	if string(returnMess) != config.READYMESSAGE {
		return NOT_VALID
	} else {
		return nil
	}
}

//检查特征字符串
func CheckValid(conn net.Conn, reuse bool, report string) error {
	var NOT_VALID = errors.New("Not valid")
	defer conn.SetReadDeadline(time.Time{})
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	message := make([]byte, 12)
	count, err := io.ReadFull(conn, message)

	if timeouterr, ok := err.(net.Error); ok && timeouterr.Timeout() {
		if reuse {
			go ProxyStream(conn, message[:count], report)
		}
		return NOT_VALID
	}

	if string(message) == config.VALIDMESSAGE {
		conn.Write([]byte(config.READYMESSAGE))
		return nil
	} else {
		if reuse {
			go ProxyStream(conn, message, report)
		}
		return NOT_VALID
	}
}

//不是stowaway的连接，进行代理
func ProxyStream(conn net.Conn, message []byte, report string) {
	reuseAddr := fmt.Sprintf("127.0.0.1:%s", report)
	reuseConn, err := net.Dial("tcp", reuseAddr)
	if err != nil {
		fmt.Println(err)
		return
	}
	reuseConn.Write(message)
	go CopyTraffic(conn, reuseConn)
	CopyTraffic(reuseConn, conn)
}

//将流量代理至正确的port
func CopyTraffic(input, output net.Conn) {
	defer input.Close()

	buf := make([]byte, 10240)
	for {
		count, err := input.Read(buf)
		if err != nil {
			if err == io.EOF && count > 0 {
				output.Write(buf[:count])
			}
			break
		}
		if count > 0 {
			output.Write(buf[:count])
		}
	}
	return
}
