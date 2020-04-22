package node

import (
	"Stowaway/config"
	"Stowaway/utils"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"time"
)

//reuse模式下的共用代码

/*-------------------------端口复用模式下节点主动连接功能代码--------------------------*/
//初始化时的连接
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
}

//connect命令时的连接
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
}

/*-------------------------端口复用模式下判断流量、转发流量功能代码--------------------------*/
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
	message := make([]byte, 8)
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

//不是来自Stowaway的连接，进行代理
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
