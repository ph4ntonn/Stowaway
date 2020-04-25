package agent

import (
	"fmt"
	"net"
	"os"

	"Stowaway/node"
	"Stowaway/share"
	"Stowaway/utils"
)

var (
	ProxyChan        *utils.ProxyChan
	SocksInfo        *utils.SocksSetting
	AgentStatus      *utils.AgentStatus
	FileDataMap      *utils.IntStrMap
	SocksDataChanMap *utils.Uint32ChanStrMap
)
var ConnToAdmin net.Conn

func NewAgent(c *utils.AgentOptions) {
	AgentStatus = utils.NewAgentStatus()
	SocksInfo = utils.NewSocksSetting()
	ProxyChan = utils.NewProxyChan()
	SocksDataChanMap = utils.NewUint32ChanStrMap()
	FileDataMap = utils.NewIntStrMap()

	AgentStatus.AESKey = []byte(c.Secret)
	listenPort := c.Listen
	reconn := c.Reconnect
	passive := c.Reverse
	monitor := c.Monitor
	isStartNode := c.IsStartNode
	reusehost := c.ReuseHost
	reuseport := c.ReusePort
	rhostreuse := c.RhostReuse

	if isStartNode && passive == false && reusehost == "" && reuseport == "" {
		go WaitForExit(AgentStatus.Nodeid)
		StartNodeInit(monitor, listenPort, reconn, passive)
	} else if passive == false && reusehost == "" && reuseport == "" {
		go WaitForExit(AgentStatus.Nodeid)
		SimpleNodeInit(monitor, listenPort, rhostreuse)
	} else if isStartNode && passive && reusehost == "" && reuseport == "" {
		go WaitForExit(AgentStatus.Nodeid)
		StartNodeReversemodeInit(monitor, listenPort, passive)
	} else if passive && reusehost == "" && reuseport == "" {
		go WaitForExit(AgentStatus.Nodeid)
		SimpleNodeReversemodeInit(monitor, listenPort)
	} else if reusehost != "" && reuseport != "" && isStartNode {
		go WaitForExit(AgentStatus.Nodeid)
		StartNodeReuseInit(reusehost, reuseport, listenPort, 1)
	} else if reusehost != "" && reuseport != "" {
		go WaitForExit(AgentStatus.Nodeid)
		SimpleNodeReuseInit(reusehost, reuseport, listenPort, 1)
	} else if reuseport != "" && listenPort != "" && isStartNode {
		StartNodeReuseInit(reusehost, reuseport, listenPort, 2)
	} else if reuseport != "" && listenPort != "" {
		SimpleNodeReuseInit(reusehost, reuseport, listenPort, 2)
	}
}

// 初始化代码开始

// 后续想让startnode与simplenode实现不一样的功能，故将两种node实现代码分开写
func StartNodeInit(monitor, listenPort, reConn string, passive bool) {
	var err error
	AgentStatus.Nodeid = utils.StartNodeId
	ConnToAdmin, AgentStatus.Nodeid, err = node.StartNodeConn(monitor, listenPort, AgentStatus.Nodeid, AgentStatus.AESKey)
	if err != nil {
		os.Exit(0)
	}
	go share.SendHeartBeatControl(&ConnToAdmin, AgentStatus.Nodeid, AgentStatus.AESKey)
	go HandleStartNodeConn(&ConnToAdmin, monitor, listenPort, reConn, passive, AgentStatus.Nodeid)
	go node.StartNodeListen(listenPort, AgentStatus.Nodeid, AgentStatus.AESKey)
	go PrepareForReOnlineNode()
	for {
		controlConnForLowerNode := <-node.NodeStuff.ControlConnForLowerNodeChan
		NewNodeMessage := <-node.NodeStuff.NewNodeMessageChan
		<-node.NodeStuff.IsAdmin //正常模式启动的节点被连接一定是agent来连接，所以这里不需要判断是否是admin连接
		ProxyChan.ProxyChanToUpperNode <- NewNodeMessage
		if AgentStatus.NotLastOne == false {
			ProxyChan.ProxyChanToLowerNode = make(chan *utils.PassToLowerNodeData)
			go HandleConnToLowerNode()
		}
		AgentStatus.NotLastOne = true
		lowerid := <-AgentStatus.WaitForIDAllocate
		go HandleConnFromLowerNode(controlConnForLowerNode, AgentStatus.Nodeid, lowerid)
	}
}

//普通的node节点
func SimpleNodeInit(monitor, listenPort string, rhostreuse bool) {
	var err error
	AgentStatus.Nodeid = utils.AdminId
	if !rhostreuse { //连接的节点是否是在reuseport？
		ConnToAdmin, AgentStatus.Nodeid, err = node.StartNodeConn(monitor, listenPort, AgentStatus.Nodeid, AgentStatus.AESKey)
	} else {
		ConnToAdmin, AgentStatus.Nodeid, err = node.StartNodeConnReuse(monitor, listenPort, AgentStatus.Nodeid, AgentStatus.AESKey)
	}
	if err != nil {
		os.Exit(0)
	}
	go share.SendHeartBeatControl(&ConnToAdmin, AgentStatus.Nodeid, AgentStatus.AESKey)
	go HandleSimpleNodeConn(&ConnToAdmin, AgentStatus.Nodeid)
	go node.StartNodeListen(listenPort, AgentStatus.Nodeid, AgentStatus.AESKey)
	go PrepareForReOnlineNode()
	for {
		controlConnForLowerNode := <-node.NodeStuff.ControlConnForLowerNodeChan
		NewNodeMessage := <-node.NodeStuff.NewNodeMessageChan
		<-node.NodeStuff.IsAdmin //正常模式启动的节点被连接一定是agent来连接，所以这里不需要判断是否是admin连接
		ProxyChan.ProxyChanToUpperNode <- NewNodeMessage
		if AgentStatus.NotLastOne == false {
			ProxyChan.ProxyChanToLowerNode = make(chan *utils.PassToLowerNodeData)
			go HandleConnToLowerNode()
		}
		AgentStatus.NotLastOne = true
		lowerid := <-AgentStatus.WaitForIDAllocate
		go HandleConnFromLowerNode(controlConnForLowerNode, AgentStatus.Nodeid, lowerid)
	}
}

//reverse mode下的startnode节点
func StartNodeReversemodeInit(monitor, listenPort string, passive bool) {
	AgentStatus.Nodeid = utils.StartNodeId
	ConnToAdmin, AgentStatus.Nodeid = node.AcceptConnFromUpperNode(listenPort, AgentStatus.Nodeid, AgentStatus.AESKey)
	go share.SendHeartBeatControl(&ConnToAdmin, AgentStatus.Nodeid, AgentStatus.AESKey)
	go HandleStartNodeConn(&ConnToAdmin, monitor, listenPort, "", passive, AgentStatus.Nodeid)
	go node.StartNodeListen(listenPort, AgentStatus.Nodeid, AgentStatus.AESKey)
	go PrepareForReOnlineNode()
	for {
		controlConnForLowerNode := <-node.NodeStuff.ControlConnForLowerNodeChan
		NewNodeMessage := <-node.NodeStuff.NewNodeMessageChan
		isAdmin := <-node.NodeStuff.IsAdmin
		if isAdmin {
			ConnToAdmin = controlConnForLowerNode
			AgentStatus.ReConnCome <- true
		} else {
			ProxyChan.ProxyChanToUpperNode <- NewNodeMessage
			if AgentStatus.NotLastOne == false {
				ProxyChan.ProxyChanToLowerNode = make(chan *utils.PassToLowerNodeData)
				go HandleConnToLowerNode()
			}
			AgentStatus.NotLastOne = true
			lowerid := <-AgentStatus.WaitForIDAllocate
			go HandleConnFromLowerNode(controlConnForLowerNode, AgentStatus.Nodeid, lowerid)
		}
	}
}

//reverse mode下的普通节点
func SimpleNodeReversemodeInit(monitor, listenPort string) {
	AgentStatus.Nodeid = utils.AdminId
	ConnToAdmin, AgentStatus.Nodeid = node.AcceptConnFromUpperNode(listenPort, AgentStatus.Nodeid, AgentStatus.AESKey)
	go share.SendHeartBeatControl(&ConnToAdmin, AgentStatus.Nodeid, AgentStatus.AESKey)
	go HandleSimpleNodeConn(&ConnToAdmin, AgentStatus.Nodeid)
	go node.StartNodeListen(listenPort, AgentStatus.Nodeid, AgentStatus.AESKey)
	go PrepareForReOnlineNode()
	for {
		controlConnForLowerNode := <-node.NodeStuff.ControlConnForLowerNodeChan
		NewNodeMessage := <-node.NodeStuff.NewNodeMessageChan
		<-node.NodeStuff.IsAdmin //被动模式启动的节点被连接一定是agent来连接，所以这里不需要判断是否是admin连接
		ProxyChan.ProxyChanToUpperNode <- NewNodeMessage
		if AgentStatus.NotLastOne == false {
			ProxyChan.ProxyChanToLowerNode = make(chan *utils.PassToLowerNodeData)
			go HandleConnToLowerNode()
		}
		AgentStatus.NotLastOne = true
		lowerid := <-AgentStatus.WaitForIDAllocate
		go HandleConnFromLowerNode(controlConnForLowerNode, AgentStatus.Nodeid, lowerid)
	}
}

//reuseport下的startnode节点
func StartNodeReuseInit(reusehost, reuseport, localport string, method int) {
	AgentStatus.Nodeid = utils.StartNodeId
	if method == 1 {
		ConnToAdmin, AgentStatus.Nodeid = node.AcceptConnFromUpperNodeReuse(reusehost, reuseport, AgentStatus.Nodeid, AgentStatus.AESKey)
	} else {
		err := node.SetPortReuseRules(localport, reuseport)
		if err != nil {
			fmt.Println("[*]Cannot set the iptable rules!")
			os.Exit(0)
		}
		ConnToAdmin, AgentStatus.Nodeid = node.AcceptConnFromUpperNodeIPTableReuse(reuseport, localport, AgentStatus.Nodeid, AgentStatus.AESKey)
	}
	go share.SendHeartBeatControl(&ConnToAdmin, AgentStatus.Nodeid, AgentStatus.AESKey)
	go HandleStartNodeConn(&ConnToAdmin, "", "", "", true, AgentStatus.Nodeid)
	if method == 1 {
		go node.StartNodeListenReuse(reusehost, reuseport, AgentStatus.Nodeid, AgentStatus.AESKey)
	} else {
		go node.StartNodeListenIPTableReuse(reuseport, localport, AgentStatus.Nodeid, AgentStatus.AESKey)
	}
	go PrepareForReOnlineNode()
	for {
		controlConnForLowerNode := <-node.NodeStuff.ControlConnForLowerNodeChan
		NewNodeMessage := <-node.NodeStuff.NewNodeMessageChan
		isAdmin := <-node.NodeStuff.IsAdmin
		if isAdmin {
			ConnToAdmin = controlConnForLowerNode
			AgentStatus.ReConnCome <- true
		} else {
			ProxyChan.ProxyChanToUpperNode <- NewNodeMessage
			if AgentStatus.NotLastOne == false {
				ProxyChan.ProxyChanToLowerNode = make(chan *utils.PassToLowerNodeData)
				go HandleConnToLowerNode()
			}
			AgentStatus.NotLastOne = true
			lowerid := <-AgentStatus.WaitForIDAllocate
			go HandleConnFromLowerNode(controlConnForLowerNode, AgentStatus.Nodeid, lowerid)
		}
	}
}

//reuseport下的普通节点
func SimpleNodeReuseInit(reusehost, reuseport, localport string, method int) {
	AgentStatus.Nodeid = utils.AdminId
	if method == 1 {
		ConnToAdmin, AgentStatus.Nodeid = node.AcceptConnFromUpperNodeReuse(reusehost, reuseport, AgentStatus.Nodeid, AgentStatus.AESKey)
	} else {
		err := node.SetPortReuseRules(localport, reuseport)
		if err != nil {
			fmt.Println("[*]Cannot set the iptable rules!")
			os.Exit(0)
		}
		ConnToAdmin, AgentStatus.Nodeid = node.AcceptConnFromUpperNodeIPTableReuse(reuseport, localport, AgentStatus.Nodeid, AgentStatus.AESKey)
	}
	go share.SendHeartBeatControl(&ConnToAdmin, AgentStatus.Nodeid, AgentStatus.AESKey)
	go HandleSimpleNodeConn(&ConnToAdmin, AgentStatus.Nodeid)
	if method == 1 {
		go node.StartNodeListenReuse(reusehost, reuseport, AgentStatus.Nodeid, AgentStatus.AESKey)
	} else {
		go node.StartNodeListenIPTableReuse(reuseport, localport, AgentStatus.Nodeid, AgentStatus.AESKey)
	}
	go PrepareForReOnlineNode()
	for {
		controlConnForLowerNode := <-node.NodeStuff.ControlConnForLowerNodeChan
		NewNodeMessage := <-node.NodeStuff.NewNodeMessageChan
		<-node.NodeStuff.IsAdmin
		ProxyChan.ProxyChanToUpperNode <- NewNodeMessage
		if AgentStatus.NotLastOne == false {
			ProxyChan.ProxyChanToLowerNode = make(chan *utils.PassToLowerNodeData)
			go HandleConnToLowerNode()
		}
		AgentStatus.NotLastOne = true
		lowerid := <-AgentStatus.WaitForIDAllocate
		go HandleConnFromLowerNode(controlConnForLowerNode, AgentStatus.Nodeid, lowerid)
	}
}

//初始化代码结束
