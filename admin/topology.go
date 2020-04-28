package admin

import (
	"errors"
	"fmt"
	"net"
	"strings"

	"Stowaway/utils"

	"github.com/gofrs/uuid"
)

var Nooode *utils.SafeNodeMap
var Route *utils.SafeRouteMap
var readyToDel []string

func init() {
	Nooode = utils.NewSafeNodeMap()
	Route = utils.NewSafeRouteMap()
}

/*-------------------------节点拓扑相关代码--------------------------*/
//将节点加入拓扑
func AddNodeToTopology(nodeid string, uppernodeid string) {
	Nooode.Lock()
	if _, ok := Nooode.AllNode[nodeid]; ok {
		Nooode.AllNode[nodeid].Uppernode = uppernodeid
	} else {
		tempnode := utils.NewNode()
		Nooode.AllNode[nodeid] = tempnode
		Nooode.AllNode[nodeid].Uppernode = uppernodeid
	}
	if uppernodeid != utils.AdminId {
		Nooode.AllNode[uppernodeid].Lowernode = append(Nooode.AllNode[uppernodeid].Lowernode, nodeid)
	}
	Nooode.Unlock()
}

//重连时对添加clientid的操作
func ReconnAddCurrentClient(id string) {
	for _, value := range CurrentClient {
		if value == id {
			return
		}
	}
	CurrentClient = append(CurrentClient, id)
}

// 将节点加入拓扑
func AddToChain() {
	for {
		newNode := <-AdminStatus.NodesReadyToadd
		for key, value := range newNode {
			NodeStatus.NodeIP[key] = value
		}
	}
}

//将节点从拓扑中删除
func DelNodeFromTopology(nodeid string) {
	Nooode.Lock()
	if _, ok := Nooode.AllNode[nodeid]; ok {
		uppernode := Nooode.AllNode[nodeid].Uppernode
		if _, ok := Nooode.AllNode[uppernode]; ok {
			index := utils.FindSpecFromSlice(nodeid, Nooode.AllNode[uppernode].Lowernode)
			Nooode.AllNode[uppernode].Lowernode = append(Nooode.AllNode[uppernode].Lowernode[:index], Nooode.AllNode[uppernode].Lowernode[index+1:]...)
		}

		Del(nodeid)
		readyToDel = append(readyToDel, nodeid)

		for _, value := range readyToDel {
			delete(Nooode.AllNode, value)
			delete(NodeStatus.NodeIP, value)
			delete(NodeStatus.Nodenote, value)
		}
		readyToDel = make([]string, 0)
	}
	Nooode.Unlock()
}

//收集需要删除的节点
func Del(nodeid string) {
	for _, value := range Nooode.AllNode[nodeid].Lowernode {
		readyToDel = append(readyToDel, value)
		Del(value)
	}
}

//找到所有的子节点
func FindAll(nodeid string) []string {
	var readyToDel []string
	Nooode.Lock()
	Find(&readyToDel, nodeid)
	Nooode.Unlock()

	readyToDel = append(readyToDel, nodeid)
	WaitForFindAll <- true
	return readyToDel
}

//收集所有的子节点
func Find(readyToDel *[]string, nodeid string) {
	for _, value := range Nooode.AllNode[nodeid].Lowernode {
		*readyToDel = append(*readyToDel, value)
		Find(readyToDel, value)
	}
}

/*-------------------------路由相关代码--------------------------*/
//计算路由表
func CalRoute() {
	Nooode.Lock()
	for key, _ := range Nooode.AllNode {
		var temp []string = []string{}
		count := key

		if key == utils.AdminId {
			continue
		}

		for {
			if Nooode.AllNode[count].Uppernode != utils.AdminId && Nooode.AllNode[count].Uppernode != utils.StartNodeId {
				count = Nooode.AllNode[count].Uppernode
				node := count
				temp = append(temp, node)
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
	Nooode.Unlock()
}

/*-------------------------节点拓扑信息相关代码--------------------------*/
// 显示节点拓扑详细信息
func ShowDetail() {
	if AdminStuff.StartNode != "0.0.0.0" {
		var nodes []string

		fmt.Printf("StartNode[1]: IP:%s  Hostname:%s  Username:%s\nNote:%s\n\n", AdminStuff.StartNode, NodeStatus.NodeHostname[utils.StartNodeId], NodeStatus.NodeUser[utils.StartNodeId], NodeStatus.Nodenote[utils.StartNodeId])

		for Nodeid, _ := range NodeStatus.NodeIP {
			nodes = append(nodes, Nodeid)
		}
		for _, value := range nodes {
			fmt.Printf("Node[%s]: IP:%s  Hostname:%s  Username:%s\nNote:%s\n\n", fmt.Sprint(FindIntByNodeid(value)+1), NodeStatus.NodeIP[value], NodeStatus.NodeHostname[utils.StartNodeId], NodeStatus.NodeUser[utils.StartNodeId], NodeStatus.Nodenote[value])
		}
	} else {
		fmt.Println("There is no agent connected!")
	}
}

//显示节点层级关系
func ShowTree() {
	if AdminStuff.StartNode != "0.0.0.0" {
		var nodes []string
		var nodesid []int

		Nooode.Lock()
		for key, _ := range Nooode.AllNode {
			nodes = append(nodes, key)
		}
		for _, value := range nodes {
			id := FindIntByNodeid(value)
			nodesid = append(nodesid, id)
		}
		//排序，防止map顺序出错
		utils.CheckRange(nodesid)

		for _, value := range nodesid {
			node := CurrentClient[value]
			nodestatus := Nooode.AllNode[node]

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
		Nooode.Unlock()
	} else {
		fmt.Println("There is no agent connected!")
	}
}

//为node添加note
func AddNote(startNodeConn net.Conn, data []string, nodeID string) bool {
	var info string
	data = data[1:len(data)]

	for _, i := range data {
		info = info + " " + i
	}

	if _, ok := NodeStatus.Nodenote[nodeID]; ok {
		NodeStatus.Nodenote[nodeID] = info
		//发送备忘至节点储存，防止admin下线后丢失备忘
		Route.Lock()
		respComm, _ := utils.ConstructPayload(nodeID, Route.Route[nodeID], "COMMAND", "YOURINFO", " ", info, 0, utils.AdminId, AdminStatus.AESKey, false)
		Route.Unlock()
		startNodeConn.Write(respComm)
		return true
	}
	return false
}

//为node删除note
func DelNote(startNodeConn net.Conn, nodeID string) bool {
	if _, ok := NodeStatus.Nodenote[nodeID]; ok {
		NodeStatus.Nodenote[nodeID] = ""
		//将节点储存的备忘同时清空
		Route.Lock()
		respComm, _ := utils.ConstructPayload(nodeID, Route.Route[nodeID], "COMMAND", "YOURINFO", " ", "", 0, utils.AdminId, AdminStatus.AESKey, false)
		Route.Unlock()
		startNodeConn.Write(respComm)
		return true
	}
	return false
}

/*-------------------------nodeid生成、搜索相关代码--------------------------*/
//生成一个nodeid
func GenerateNodeID() string {
	u2, _ := uuid.NewV4()
	uu := strings.Replace(u2.String(), "-", "", -1)
	uuid := uu[11:21] //取10位，尽量减少包头长度
	CurrentClient = append(CurrentClient, uuid)
	return uuid
}

//将字符串型的nodeid转为对应的int
func FindNumByNodeid(id string) (string, error) {
	var NO_NODE = errors.New("This node isn't exist")

	if id == "" {
		return "", NO_NODE
	}

	nodeid := utils.StrUint32(id)
	currentid := int(nodeid) - 1

	if len(CurrentClient) < int(nodeid) {
		return "", NO_NODE
	}

	return CurrentClient[currentid], nil
}

//用int找到对应的nodeid
func FindIntByNodeid(id string) int {
	for key, value := range CurrentClient {
		if value == id {
			return key
		}
	}
	return 0
}
