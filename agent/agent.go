package agent

import (
	"log"
	"net"

	"Stowaway/node"
	"Stowaway/share"
	"Stowaway/utils"
)

var AgentStuff *utils.AgentStuff
var AgentStatus *utils.AgentStatus
var ConnToAdmin net.Conn

func init() {
	AgentStatus = utils.NewAgentStatus()
	AgentStuff = utils.NewAgentStuff()
}

// NewAgent 启动agent
func NewAgent(c *utils.AgentOptions) {
	//解析参数
	AgentStatus.AESKey = []byte(c.Secret)
	listenPort := c.Listen
	reconn := c.Reconnect
	passive := c.Reverse
	monitor := c.Connect
	isStartNode := c.IsStartNode
	reuseHost := c.ReuseHost
	reusePort := c.ReusePort
	rhostReuse := c.RhostReuse
	proxy := c.Proxy
	proxyU := c.ProxyU
	proxyP := c.ProxyP
	//设置通信字符串
	node.SetValidtMessage(AgentStatus.AESKey)
	node.SetForwardMessage(AgentStatus.AESKey)
	//根据选择确定启动方式
	if isStartNode && passive == false && reuseHost == "" && reusePort == "" {
		go WaitForExit()
		StartNodeInit(monitor, listenPort, reconn,proxy, proxyU, proxyP, passive)
	} else if passive == false && reuseHost == "" && reusePort == "" {
		go WaitForExit()
		SimpleNodeInit(monitor, listenPort, proxy, proxyU, proxyP, rhostReuse)
	} else if isStartNode && passive && reuseHost == "" && reusePort == "" {
		go WaitForExit()
		StartNodeReversemodeInit(monitor, listenPort, proxy, proxyU, proxyP,passive)
	} else if passive && reuseHost == "" && reusePort == "" {
		go WaitForExit()
		SimpleNodeReversemodeInit(monitor, listenPort)
	} else if reuseHost != "" && reusePort != "" && isStartNode {
		go WaitForExit()
		StartNodeReuseInit(reuseHost, reusePort, listenPort, proxy, proxyU, proxyP, 1)
	} else if reuseHost != "" && reusePort != "" {
		go WaitForExit()
		SimpleNodeReuseInit(reuseHost, reusePort, listenPort, 1)
	} else if reusePort != "" && listenPort != "" && isStartNode {
		StartNodeReuseInit(reuseHost, reusePort, listenPort, proxy, proxyU, proxyP,2)
	} else if reusePort != "" && listenPort != "" {
		SimpleNodeReuseInit(reuseHost, reusePort, listenPort, 2)
	}
}

// 初始化代码开始

// StartNodeInit 后续想让startnode与simplenode实现不一样的功能，故将两种node实现代码分开写
func StartNodeInit(monitor, listenPort, reConn, proxy, proxyU, proxyP string, passive bool) {
	var err error
	AgentStatus.Nodeid = utils.StartNodeId

	ConnToAdmin, AgentStatus.Nodeid, err = node.StartNodeConn(monitor, listenPort, AgentStatus.Nodeid, proxy, proxyU, proxyP,AgentStatus.AESKey)
	if err != nil {
		log.Fatalf("[*]Error occured: %s\n", err)
	}

	go SendInfo(AgentStatus.Nodeid) //发送自身信息
	go share.SendHeartBeatControl(&ConnToAdmin, AgentStatus.Nodeid, AgentStatus.AESKey)
	go HandleStartNodeConn(&ConnToAdmin, monitor, listenPort, reConn, passive, AgentStatus.Nodeid, proxy, proxyU, proxyP)
	go node.StartNodeListen(listenPort, AgentStatus.Nodeid, AgentStatus.AESKey)
	go PrepareForReOnlineNode()

	for {
		payloadBuffChan := make(chan *utils.Payload, 10)
		controlConnForLowerNode := <-node.NodeStuff.ControlConnForLowerNodeChan
		newNodeMessage := <-node.NodeStuff.NewNodeMessageChan
		<-node.NodeStuff.IsAdmin //正常模式启动的节点被连接一定是agent来连接，所以这里不需要判断是否是admin连接
		AgentStuff.ProxyChan.ProxyChanToUpperNode <- newNodeMessage
		if AgentStatus.NotLastOne == false {
			AgentStuff.ProxyChan.ProxyChanToLowerNode = make(chan *utils.PassToLowerNodeData)
			go HandleDataToLowerNode()
		}
		AgentStatus.NotLastOne = true
		lowerid := <-AgentStatus.WaitForIDAllocate
		go HandleLowerNodeConn(controlConnForLowerNode, payloadBuffChan, AgentStatus.Nodeid, lowerid)
		go HandleDataFromLowerNode(controlConnForLowerNode, payloadBuffChan, AgentStatus.Nodeid, lowerid)
	}
}

// SimpleNodeInit 普通的node节点
func SimpleNodeInit(monitor, listenPort, proxy, proxyU, proxyP string, rhostReuse bool) {
	var err error
	AgentStatus.Nodeid = utils.AdminId

	if !rhostReuse { //连接的节点是否是在reuseport？
		ConnToAdmin, AgentStatus.Nodeid, err = node.StartNodeConn(monitor, listenPort, AgentStatus.Nodeid, proxy, proxyU, proxyP, AgentStatus.AESKey)
	} else {
		ConnToAdmin, AgentStatus.Nodeid, err = node.StartNodeConnReuse(monitor, listenPort, AgentStatus.Nodeid, proxy, proxyU, proxyP,AgentStatus.AESKey)
	}
	if err != nil {
		log.Fatalf("[*]Error occured: %s\n", err)
	}
	//与上级连接建立成功后的代码
	go SendInfo(AgentStatus.Nodeid)
	go share.SendHeartBeatControl(&ConnToAdmin, AgentStatus.Nodeid, AgentStatus.AESKey)
	go HandleSimpleNodeConn(&ConnToAdmin, AgentStatus.Nodeid)
	go node.StartNodeListen(listenPort, AgentStatus.Nodeid, AgentStatus.AESKey)
	go PrepareForReOnlineNode()
	//等待下级节点的连接
	for {
		payloadBuffChan := make(chan *utils.Payload, 10)
		controlConnForLowerNode := <-node.NodeStuff.ControlConnForLowerNodeChan
		newNodeMessage := <-node.NodeStuff.NewNodeMessageChan
		<-node.NodeStuff.IsAdmin //正常模式启动的节点被连接一定是agent来连接，所以这里不需要判断是否是admin连接
		AgentStuff.ProxyChan.ProxyChanToUpperNode <- newNodeMessage
		if AgentStatus.NotLastOne == false {
			AgentStuff.ProxyChan.ProxyChanToLowerNode = make(chan *utils.PassToLowerNodeData)
			go HandleDataToLowerNode()
		}
		AgentStatus.NotLastOne = true
		lowerid := <-AgentStatus.WaitForIDAllocate
		go HandleLowerNodeConn(controlConnForLowerNode, payloadBuffChan, AgentStatus.Nodeid, lowerid)
		go HandleDataFromLowerNode(controlConnForLowerNode, payloadBuffChan, AgentStatus.Nodeid, lowerid)
	}
}

// StartNodeReversemodeInit reverse mode下的startnode节点
func StartNodeReversemodeInit(monitor, listenPort, proxy, proxyU, proxyP string, passive bool) {
	AgentStatus.Nodeid = utils.StartNodeId

	ConnToAdmin, AgentStatus.Nodeid = node.AcceptConnFromUpperNode(listenPort, AgentStatus.Nodeid, AgentStatus.AESKey)

	go SendInfo(AgentStatus.Nodeid)
	go share.SendHeartBeatControl(&ConnToAdmin, AgentStatus.Nodeid, AgentStatus.AESKey)
	go HandleStartNodeConn(&ConnToAdmin, monitor, listenPort, "", passive, AgentStatus.Nodeid, proxy, proxyU, proxyP)
	go node.StartNodeListen(listenPort, AgentStatus.Nodeid, AgentStatus.AESKey)
	go PrepareForReOnlineNode()

	for {
		payloadBuffChan := make(chan *utils.Payload, 10)
		controlConnForLowerNode := <-node.NodeStuff.ControlConnForLowerNodeChan
		newNodeMessage := <-node.NodeStuff.NewNodeMessageChan
		isAdmin := <-node.NodeStuff.IsAdmin
		if isAdmin {
			ConnToAdmin = controlConnForLowerNode
			AgentStatus.ReConnCome <- true
		} else {
			AgentStuff.ProxyChan.ProxyChanToUpperNode <- newNodeMessage
			if AgentStatus.NotLastOne == false {
				AgentStuff.ProxyChan.ProxyChanToLowerNode = make(chan *utils.PassToLowerNodeData)
				go HandleDataToLowerNode()
			}
			AgentStatus.NotLastOne = true
			lowerid := <-AgentStatus.WaitForIDAllocate
			go HandleLowerNodeConn(controlConnForLowerNode, payloadBuffChan, AgentStatus.Nodeid, lowerid)
			go HandleDataFromLowerNode(controlConnForLowerNode, payloadBuffChan, AgentStatus.Nodeid, lowerid)
		}
	}
}

// SimpleNodeReversemodeInit reverse mode下的普通节点
func SimpleNodeReversemodeInit(monitor, listenPort string) {
	AgentStatus.Nodeid = utils.AdminId

	ConnToAdmin, AgentStatus.Nodeid = node.AcceptConnFromUpperNode(listenPort, AgentStatus.Nodeid, AgentStatus.AESKey)

	go SendInfo(AgentStatus.Nodeid)
	go share.SendHeartBeatControl(&ConnToAdmin, AgentStatus.Nodeid, AgentStatus.AESKey)
	go HandleSimpleNodeConn(&ConnToAdmin, AgentStatus.Nodeid)
	go node.StartNodeListen(listenPort, AgentStatus.Nodeid, AgentStatus.AESKey)
	go PrepareForReOnlineNode()

	for {
		payloadBuffChan := make(chan *utils.Payload, 10)
		controlConnForLowerNode := <-node.NodeStuff.ControlConnForLowerNodeChan
		newNodeMessage := <-node.NodeStuff.NewNodeMessageChan
		<-node.NodeStuff.IsAdmin //被动模式启动的节点被连接一定是agent来连接，所以这里不需要判断是否是admin连接
		AgentStuff.ProxyChan.ProxyChanToUpperNode <- newNodeMessage
		if AgentStatus.NotLastOne == false {
			AgentStuff.ProxyChan.ProxyChanToLowerNode = make(chan *utils.PassToLowerNodeData)
			go HandleDataToLowerNode()
		}
		AgentStatus.NotLastOne = true
		lowerid := <-AgentStatus.WaitForIDAllocate
		go HandleLowerNodeConn(controlConnForLowerNode, payloadBuffChan, AgentStatus.Nodeid, lowerid)
		go HandleDataFromLowerNode(controlConnForLowerNode, payloadBuffChan, AgentStatus.Nodeid, lowerid)
	}
}

// StartNodeReuseInit reuseport下的startnode节点
func StartNodeReuseInit(reuseHost, reusePort, localPort, proxy, proxyU, proxyP string, method int) {
	AgentStatus.Nodeid = utils.StartNodeId

	if method == 1 {
		ConnToAdmin, AgentStatus.Nodeid = node.AcceptConnFromUpperNodeReuse(reuseHost, reusePort, AgentStatus.Nodeid, AgentStatus.AESKey)
	} else {
		err := node.SetPortReuseRules(localPort, reusePort)
		if err != nil {
			log.Fatal("[*]Cannot set the iptable rules!")
		}
		ConnToAdmin, AgentStatus.Nodeid = node.AcceptConnFromUpperNodeIPTableReuse(reusePort, localPort, AgentStatus.Nodeid, AgentStatus.AESKey)
	}

	go SendInfo(AgentStatus.Nodeid)
	go share.SendHeartBeatControl(&ConnToAdmin, AgentStatus.Nodeid, AgentStatus.AESKey)
	go HandleStartNodeConn(&ConnToAdmin, "", "", "", true, AgentStatus.Nodeid, proxy, proxyU, proxyP)

	if method == 1 {
		go node.StartNodeListenReuse(reuseHost, reusePort, AgentStatus.Nodeid, AgentStatus.AESKey)
	} else {
		go node.StartNodeListenIPTableReuse(reusePort, localPort, AgentStatus.Nodeid, AgentStatus.AESKey)
	}

	go PrepareForReOnlineNode()

	for {
		payloadBuffChan := make(chan *utils.Payload, 10)
		controlConnForLowerNode := <-node.NodeStuff.ControlConnForLowerNodeChan
		newNodeMessage := <-node.NodeStuff.NewNodeMessageChan
		isAdmin := <-node.NodeStuff.IsAdmin
		if isAdmin {
			ConnToAdmin = controlConnForLowerNode
			AgentStatus.ReConnCome <- true
		} else {
			AgentStuff.ProxyChan.ProxyChanToUpperNode <- newNodeMessage
			if AgentStatus.NotLastOne == false {
				AgentStuff.ProxyChan.ProxyChanToLowerNode = make(chan *utils.PassToLowerNodeData)
				go HandleDataToLowerNode()
			}
			AgentStatus.NotLastOne = true
			lowerid := <-AgentStatus.WaitForIDAllocate
			go HandleLowerNodeConn(controlConnForLowerNode, payloadBuffChan, AgentStatus.Nodeid, lowerid)
			go HandleDataFromLowerNode(controlConnForLowerNode, payloadBuffChan, AgentStatus.Nodeid, lowerid)
		}
	}
}

// SimpleNodeReuseInit reuseport下的普通节点
func SimpleNodeReuseInit(reuseHost, reusePort, localPort string, method int) {
	AgentStatus.Nodeid = utils.AdminId

	if method == 1 {
		ConnToAdmin, AgentStatus.Nodeid = node.AcceptConnFromUpperNodeReuse(reuseHost, reusePort, AgentStatus.Nodeid, AgentStatus.AESKey)
	} else {
		err := node.SetPortReuseRules(localPort, reusePort)
		if err != nil {
			log.Fatal("[*]Cannot set the iptable rules!")
		}
		ConnToAdmin, AgentStatus.Nodeid = node.AcceptConnFromUpperNodeIPTableReuse(reusePort, localPort, AgentStatus.Nodeid, AgentStatus.AESKey)
	}

	go SendInfo(AgentStatus.Nodeid)
	go share.SendHeartBeatControl(&ConnToAdmin, AgentStatus.Nodeid, AgentStatus.AESKey)
	go HandleSimpleNodeConn(&ConnToAdmin, AgentStatus.Nodeid)

	if method == 1 {
		go node.StartNodeListenReuse(reuseHost, reusePort, AgentStatus.Nodeid, AgentStatus.AESKey)
	} else {
		go node.StartNodeListenIPTableReuse(reusePort, localPort, AgentStatus.Nodeid, AgentStatus.AESKey)
	}

	go PrepareForReOnlineNode()

	for {
		payloadBuffChan := make(chan *utils.Payload, 10)
		controlConnForLowerNode := <-node.NodeStuff.ControlConnForLowerNodeChan
		newNodeMessage := <-node.NodeStuff.NewNodeMessageChan
		<-node.NodeStuff.IsAdmin
		AgentStuff.ProxyChan.ProxyChanToUpperNode <- newNodeMessage
		if AgentStatus.NotLastOne == false {
			AgentStuff.ProxyChan.ProxyChanToLowerNode = make(chan *utils.PassToLowerNodeData)
			go HandleDataToLowerNode()
		}
		AgentStatus.NotLastOne = true
		lowerid := <-AgentStatus.WaitForIDAllocate
		go HandleLowerNodeConn(controlConnForLowerNode, payloadBuffChan, AgentStatus.Nodeid, lowerid)
		go HandleDataFromLowerNode(controlConnForLowerNode, payloadBuffChan, AgentStatus.Nodeid, lowerid)
	}
}

//初始化代码结束
