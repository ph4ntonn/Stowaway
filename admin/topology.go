package admin

import (
	"errors"
	"fmt"
	"net"
	"strings"

	"Stowaway/utils"

	"github.com/gofrs/uuid"
)

var Topology *utils.SafeNodeMap
var Route *utils.SafeRouteMap

func init() {
	Topology = utils.NewSafeNodeMap()
	Route = utils.NewSafeRouteMap()
}

/*-------------------------节点拓扑相关代码--------------------------*/

// AddNodeToTopology 将节点加入拓扑
func AddNodeToTopology(nodeid string, uppernodeid string) {
	Topology.Lock()
	defer Topology.Unlock()

	if _, ok := Topology.AllNode[nodeid]; ok {
		Topology.AllNode[nodeid].Uppernode = uppernodeid
	} else {
		tempnode := utils.NewNode()
		Topology.AllNode[nodeid] = tempnode
		Topology.AllNode[nodeid].Uppernode = uppernodeid
	}
	if uppernodeid != utils.AdminId {
		Topology.AllNode[uppernodeid].Lowernode = append(Topology.AllNode[uppernodeid].Lowernode, nodeid)
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

// DelNodeFromTopology 将节点从拓扑中删除
func DelNodeFromTopology(nodeid string) {
	var readyToDel []string

	Topology.Lock()
	defer Topology.Unlock()

	if _, ok := Topology.AllNode[nodeid]; ok {
		uppernode := Topology.AllNode[nodeid].Uppernode
		if _, ok := Topology.AllNode[uppernode]; ok {
			index := utils.FindSpecFromSlice(nodeid, Topology.AllNode[uppernode].Lowernode)
			Topology.AllNode[uppernode].Lowernode = append(Topology.AllNode[uppernode].Lowernode[:index], Topology.AllNode[uppernode].Lowernode[index+1:]...)
		}

		Del(nodeid, readyToDel)

		readyToDel = append(readyToDel, nodeid)
		for _, value := range readyToDel {
			delete(Topology.AllNode, value)
			delete(AdminStuff.NodeStatus.NodeIP, value)
			delete(AdminStuff.NodeStatus.Nodenote, value)
		}
		readyToDel = make([]string, 0)
	}
}

// Del 收集需要删除的节点
func Del(nodeid string, readyToDel []string) {
	for _, value := range Topology.AllNode[nodeid].Lowernode {
		readyToDel = append(readyToDel, value)
		Del(value, readyToDel)
	}
}

// FindAll 找到所有的子节点
func FindAll(nodeid string) []string {
	var readyToDel []string

	Find(&readyToDel, nodeid)

	readyToDel = append(readyToDel, nodeid)

	return readyToDel
}

// Find 收集所有的子节点
func Find(readyToDel *[]string, nodeid string) {
	Topology.Lock()
	for _, value := range Topology.AllNode[nodeid].Lowernode {
		*readyToDel = append(*readyToDel, value)
		Find(readyToDel, value)
	}
	Topology.Unlock()
}

/*-------------------------路由相关代码--------------------------*/

// CalRoute 计算路由表
func CalRoute() {
	Topology.Lock()
	defer Topology.Unlock()

	for key, _ := range Topology.AllNode {
		var temp []string = []string{}
		count := key

		if key == utils.AdminId {
			continue
		}

		for {
			if Topology.AllNode[count].Uppernode != utils.AdminId && Topology.AllNode[count].Uppernode != utils.StartNodeId {
				count = Topology.AllNode[count].Uppernode
				temp = append(temp, count)
			} else {
				utils.StringReverse(temp)
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

		for Nodeid, _ := range AdminStuff.NodeStatus.NodeIP {
			nodes = append(nodes, Nodeid)
		}
		for _, value := range nodes {
			fmt.Printf("Node[%s]: IP:%s  Hostname:%s  Username:%s\nNote:%s\n\n",
				fmt.Sprint(FindIntByNodeid(value)+1),
				AdminStuff.NodeStatus.NodeIP[value],
				AdminStuff.NodeStatus.NodeHostname[utils.StartNodeId],
				AdminStuff.NodeStatus.NodeUser[utils.StartNodeId],
				AdminStuff.NodeStatus.Nodenote[value],
			)
		}
	} else {
		fmt.Println("There is no agent connected!")
	}
}

// ShowTree 显示节点层级关系
func ShowTree() {
	if AdminStatus.StartNode != "0.0.0.0" {
		var nodes []string
		var nodesid []int

		Topology.Lock()
		defer Topology.Unlock()

		for key, _ := range Topology.AllNode {
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
			nodestatus := Topology.AllNode[node]

			if node == utils.StartNodeId {
				fmt.Printf("StartNode[%s]'s child nodes:\n", fmt.Sprint(value+1))
				if len(nodestatus.Lowernode) == 0 {
					fmt.Println("\tThere is no child node for this one.")
				} else {
					for _, child := range nodestatus.Lowernode {
						childid := FindIntByNodeid(child)
						fmt.Printf("\tNode [%s]\n", fmt.Sprint(childid+1))
					}
				}
			} else {
				fmt.Printf("Node[%s]'s child nodes:\n", fmt.Sprint(value+1))
				if len(nodestatus.Lowernode) == 0 {
					fmt.Println("\tThere is no child node for this one.")
				} else {
					for _, child := range nodestatus.Lowernode {
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

// AddNote 为node添加note
func AddNote(startNodeConn net.Conn, data []string, nodeID string) bool {
	var info string

	for _, i := range data[1:len(data)] {
		info = info + " " + i
	}

	if _, ok := AdminStuff.NodeStatus.Nodenote[nodeID]; ok {
		AdminStuff.NodeStatus.Nodenote[nodeID] = info
		//发送备忘至节点储存，防止admin下线后丢失备忘
		SendPayloadViaRoute(startNodeConn, nodeID, "COMMAND", "YOURINFO", " ", info, 0, utils.AdminId, AdminStatus.AESKey, false)
		return true
	}
	return false
}

// DelNote 为node删除note
func DelNote(startNodeConn net.Conn, nodeID string) bool {
	if _, ok := AdminStuff.NodeStatus.Nodenote[nodeID]; ok {
		AdminStuff.NodeStatus.Nodenote[nodeID] = ""
		//将节点储存的备忘同时清空
		SendPayloadViaRoute(startNodeConn, nodeID, "COMMAND", "YOURINFO", " ", "", 0, utils.AdminId, AdminStatus.AESKey, false)
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
