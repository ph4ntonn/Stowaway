package node

import (
	"Stowaway/common"
	"fmt"
	"log"
	"net"
	"os"

	reuseport "github.com/libp2p/go-reuseport"
)

/*-------------------------SO_REUSEPORT,SO_REUSEADDR复用模式功能代码--------------------------*/
//以下代码和init.go中大体相似，只是为了将改动剥离，所以单列出来
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
