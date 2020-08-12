package admin

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"

	"Stowaway/utils"

	"github.com/gofrs/uuid"
)

type Topology struct {
	sync.RWMutex
	AllNode map[string]*Node
}

type Node struct {
	Uppernode string
	Lowernode []string
}

func NewTopology() *Topology {
	nsnm := new(Topology)
	nsnm.AllNode = make(map[string]*Node)
	return nsnm
}

func NewNode() *Node {
	nn := new(Node)
	nn.Lowernode = make([]string, 0)
	return nn
}

/*-------------------------节点拓扑相关代码--------------------------*/

// AddNode 将节点加入拓扑
func (topology *Topology) AddNode(nodeid, upperNodeId string) {
	topology.Lock()
	defer topology.Unlock()

	if _, ok := topology.AllNode[nodeid]; ok {
		topology.AllNode[nodeid].Uppernode = upperNodeId
	} else {
		topology.AllNode[nodeid] = NewNode()
		topology.AllNode[nodeid].Uppernode = upperNodeId
	}
	if upperNodeId != utils.AdminId {
		topology.AllNode[upperNodeId].Lowernode = append(topology.AllNode[upperNodeId].Lowernode, nodeid)
	}
}

// DelNode 将节点从拓扑中删除
func (topology *Topology) DelNode(nodeid string) {
	var readyToDel []string

	topology.Lock()
	defer topology.Unlock()

	if _, ok := topology.AllNode[nodeid]; ok {
		upperNode := topology.AllNode[nodeid].Uppernode
		if _, ok := topology.AllNode[upperNode]; ok {
			index := utils.FindSpecFromSlice(nodeid, topology.AllNode[upperNode].Lowernode)
			topology.AllNode[upperNode].Lowernode = append(topology.AllNode[upperNode].Lowernode[:index], topology.AllNode[upperNode].Lowernode[index+1:]...)
		}

		topology.Find(&readyToDel, nodeid)

		readyToDel = append(readyToDel, nodeid)
		for _, value := range readyToDel {
			delete(topology.AllNode, value)
			delete(AdminStuff.NodeStatus.NodeIP, value)
			delete(AdminStuff.NodeStatus.Nodenote, value)
		}
		readyToDel = make([]string, 0)
	}
}

// FindAll 找到所有的子节点
func (topology *Topology) FindAll(nodeid string) []string {
	var readyToDel []string

	topology.Lock()
	defer topology.Unlock()

	topology.Find(&readyToDel, nodeid)

	readyToDel = append(readyToDel, nodeid)

	return readyToDel
}

// Find 收集所有的子节点
func (topology *Topology) Find(readyToDel *[]string, nodeid string) {
	for _, value := range topology.AllNode[nodeid].Lowernode {
		*readyToDel = append(*readyToDel, value)
		topology.Find(readyToDel, value)
	}
}

// ReconnAddCurrentClient 重连时对添加clientid的操作
func ReconnAddCurrentClient(id string) {
	for _, value := range AdminStatus.CurrentClient {
		if value == id {
			return
		}
	}
	AdminStatus.CurrentClient = append(AdminStatus.CurrentClient, id)
}

// AddToChain 将节点加入拓扑
func AddToChain() {
	for {
		newNode := <-AdminStatus.NodesReadyToadd
		for key, value := range newNode {
			AdminStuff.NodeStatus.NodeIP[key] = value
		}
	}
}

/*-------------------------路由相关代码--------------------------*/

// CalRoute 计算路由表
func (topology *Topology) CalRoute() {
	topology.Lock()
	defer topology.Unlock()

	for key := range topology.AllNode {
		var temp []string = []string{}
		count := key

		if key == utils.AdminId {
			continue
		}

		for {
			if topology.AllNode[count].Uppernode != utils.AdminId && topology.AllNode[count].Uppernode != utils.StartNodeId {
				count = topology.AllNode[count].Uppernode
				temp = append(temp, count)
			} else {
				utils.StringSliceReverse(temp)
				route := strings.Join(temp, ":")
				Route.Lock()
				Route.Route[key] = route
				Route.Unlock()
				break
			}
		}
	}
}

/*-------------------------节点拓扑信息相关代码--------------------------*/

// ShowTree 显示节点层级关系
func (topology *Topology) ShowTree() {
	if AdminStatus.StartNode != "0.0.0.0" {
		var nodes []string
		var nodesid []int

		topology.Lock()
		defer topology.Unlock()

		for key := range topology.AllNode {
			nodes = append(nodes, key)
		}

		for _, value := range nodes {
			id := FindIntByNodeid(value)
			nodesid = append(nodesid, id)
		}
		//排序，防止map顺序出错
		utils.CheckRange(nodesid)

		for _, value := range nodesid {
			node := AdminStatus.CurrentClient[value]
			nodeStatus := topology.AllNode[node]

			if node == utils.StartNodeId {
				fmt.Printf("StartNode[%s]'s child nodes:\n", fmt.Sprint(value+1))
				if len(nodeStatus.Lowernode) == 0 {
					fmt.Println("\tThere is no child node for this one.")
				} else {
					for _, child := range nodeStatus.Lowernode {
						childid := FindIntByNodeid(child)
						fmt.Printf("\tNode [%s]\n", fmt.Sprint(childid+1))
					}
				}
			} else {
				fmt.Printf("Node[%s]'s child nodes:\n", fmt.Sprint(value+1))
				if len(nodeStatus.Lowernode) == 0 {
					fmt.Println("\tThere is no child node for this one.")
				} else {
					for _, child := range nodeStatus.Lowernode {
						childid := FindIntByNodeid(child)
						fmt.Printf("\tNode [%s]\n", fmt.Sprint(childid+1))
					}
				}
			}
		}
	} else {
		fmt.Println("There is no agent connected!")
	}
}

// ShowDetail 显示节点拓扑详细信息
func ShowDetail() {
	if AdminStatus.StartNode != "0.0.0.0" {
		var nodes []string

		fmt.Printf("StartNode[1]: IP:%s  Hostname:%s  Username:%s\nNote:%s\n\n",
			AdminStatus.StartNode,
			AdminStuff.NodeStatus.NodeHostname[utils.StartNodeId],
			AdminStuff.NodeStatus.NodeUser[utils.StartNodeId],
			AdminStuff.NodeStatus.Nodenote[utils.StartNodeId],
		)

		for Nodeid := range AdminStuff.NodeStatus.NodeIP {
			nodes = append(nodes, Nodeid)
		}

		for _, id := range nodes {
			fmt.Printf("Node[%s]: IP:%s  Hostname:%s  Username:%s\nNote:%s\n\n",
				fmt.Sprint(FindIntByNodeid(id)+1),
				AdminStuff.NodeStatus.NodeIP[id],
				AdminStuff.NodeStatus.NodeHostname[id],
				AdminStuff.NodeStatus.NodeUser[id],
				AdminStuff.NodeStatus.Nodenote[id],
			)
		}
	} else {
		fmt.Println("There is no agent connected!")
	}
}

// AddNote 为node添加note
func AddNote(startNodeConn net.Conn, data []string, nodeid string) bool {
	var info string

	for _, i := range data[1:len(data)] {
		info = info + " " + i
	}

	if _, ok := AdminStuff.NodeStatus.Nodenote[nodeid]; ok {
		AdminStuff.NodeStatus.Nodenote[nodeid] = info
		//发送备忘至节点储存，防止admin下线后丢失备忘
		SendPayloadViaRoute(startNodeConn, nodeid, "COMMAND", "YOURINFO", " ", info, 0, utils.AdminId, AdminStatus.AESKey, false)
		return true
	}
	return false
}

// DelNote 为node删除note
func DelNote(startNodeConn net.Conn, nodeid string) bool {
	if _, ok := AdminStuff.NodeStatus.Nodenote[nodeid]; ok {
		AdminStuff.NodeStatus.Nodenote[nodeid] = ""
		//将节点储存的备忘同时清空
		SendPayloadViaRoute(startNodeConn, nodeid, "COMMAND", "YOURINFO", " ", "", 0, utils.AdminId, AdminStatus.AESKey, false)
		return true
	}
	return false
}

/*-------------------------nodeid生成、搜索相关代码--------------------------*/

// GenerateNodeID 生成一个nodeid
func GenerateNodeID() string {
	u2, _ := uuid.NewV4()
	uu := strings.Replace(u2.String(), "-", "", -1)
	uuid := uu[11:21] //取10位，尽量减少包头长度
	AdminStatus.CurrentClient = append(AdminStatus.CurrentClient, uuid)
	return uuid
}

// FindNumByNodeid 将字符串型的nodeid转为对应的int
func FindNumByNodeid(id string) (string, error) {
	var NO_NODE = errors.New("This node isn't exist")

	if id == "" {
		return "", NO_NODE
	}

	nodeid := int(utils.StrUint32(id))
	currentid := nodeid - 1

	if len(AdminStatus.CurrentClient) < nodeid {
		return "", NO_NODE
	}

	return AdminStatus.CurrentClient[currentid], nil
}

// FindIntByNodeid 用int找到对应的nodeid
func FindIntByNodeid(id string) int {
	for key, value := range AdminStatus.CurrentClient {
		if value == id {
			return key
		}
	}
	return 0
}
