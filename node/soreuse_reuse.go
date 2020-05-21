package node

import (
	"fmt"
	"log"
	"net"

	"Stowaway/utils"

	reuseport "github.com/libp2p/go-reuseport"
)

//以下代码和init.go中大体相似，只是为了将改动剥离，所以单列出来

/*-------------------------SO_REUSEPORT,SO_REUSEADDR复用模式功能代码--------------------------*/

// StartNodeListenReuse 初始化节点监听操作
func StartNodeListenReuse(rehost, report string, nodeid string, key []byte) {
	var newNodeMessage []byte

	if report == "" { //如果没有port，直接退出
		return
	}

	listenAddr := fmt.Sprintf("%s:%s", rehost, report)
	waitingForLowerNode, err := reuseport.Listen("tcp", listenAddr)

	if err != nil {
		log.Fatalf("[*]Cannot listen on port %s", report)
	}

	for {
		connToLowerNode, err := waitingForLowerNode.Accept()
		if err != nil {
			log.Println("[*]", err)
			return
		}

		err = CheckValid(connToLowerNode, true, report)
		if err != nil {
			continue
		}

		for i := 0; i < 2; i++ {
			command, _ := utils.ExtractPayload(connToLowerNode, key, utils.AdminId, true)
			switch command.Command {
			case "STOWAWAYADMIN":
				utils.ConstructPayloadAndSend(connToLowerNode, nodeid, "", "COMMAND", "INIT", " ", report, 0, utils.AdminId, key, false)
			case "ID":
				NodeStuff.ControlConnForLowerNodeChan <- connToLowerNode
				NodeStuff.NewNodeMessageChan <- newNodeMessage
				NodeStuff.IsAdmin <- true
			case "REONLINESUC":
				NodeStuff.Adminconn <- connToLowerNode
			case "STOWAWAYAGENT":
				if !NodeStuff.Offline {
					utils.ConstructPayloadAndSend(connToLowerNode, nodeid, "", "COMMAND", "CONFIRM", " ", " ", 0, nodeid, key, false)
				} else {
					utils.ConstructPayloadAndSend(connToLowerNode, nodeid, "", "COMMAND", "REONLINE", " ", report, 0, nodeid, key, false)
				}
			case "INIT":
				//告知admin新节点消息
				newNodeMessage, _ = utils.ConstructPayload(utils.AdminId, "", "COMMAND", "NEW", " ", connToLowerNode.RemoteAddr().String(), 0, nodeid, key, false)
				NodeInfo.LowerNode.Payload[utils.AdminId] = connToLowerNode //将这个socket用0号位暂存，等待admin分配完id后再将其放入对应的位置
				NodeStuff.ControlConnForLowerNodeChan <- connToLowerNode
				NodeStuff.NewNodeMessageChan <- newNodeMessage //被连接后不终止监听，继续等待可能的后续节点连接，以此组成树状结构
				NodeStuff.IsAdmin <- false
			}
		}
	}
}

// AcceptConnFromUpperNodeReuse 被动模式下startnode接收admin重连 && 普通节点被动启动等待上级节点主动连接
func AcceptConnFromUpperNodeReuse(rehost, report string, nodeid string, key []byte) (net.Conn, string) {
	listenAddr := fmt.Sprintf("%s:%s", rehost, report)
	waitingForConn, err := reuseport.Listen("tcp", listenAddr)

	if err != nil {
		log.Fatalf("[*]Cannot reuse port %s", report)
	}

	for {
		comingConn, err := waitingForConn.Accept()
		if err != nil {
			log.Println("[*]", err)
			continue
		}

		err = CheckValid(comingConn, true, report)
		if err != nil {
			continue
		}

		utils.ExtractPayload(comingConn, key, utils.AdminId, true)

		utils.ConstructPayloadAndSend(comingConn, nodeid, "", "COMMAND", "INIT", " ", report, 0, utils.AdminId, key, false)

		command, _ := utils.ExtractPayload(comingConn, key, utils.AdminId, true) //等待分配id
		if command.Command == "ID" {
			nodeid = command.NodeId
			waitingForConn.Close()
			return comingConn, nodeid
		}

	}

}
