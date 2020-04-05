package admin

import (
	"Stowaway/common"
	"fmt"
	"strings"
	"sync"
)

type Node struct {
	Uppernode uint32
	Lowernode []uint32
}

type SafeNodeMap struct {
	sync.RWMutex
	AllNode map[uint32]*Node
}

type SafeRouteMap struct {
	sync.RWMutex
	Route map[uint32]string
}

var Nooode *SafeNodeMap
var Route *SafeRouteMap
var readyToDel []uint32

func init() {
	Nooode = NewSafeNodeMap()
	Route = NewSafeRouteMap()
}

func NewNode() *Node {
	nn := new(Node)
	nn.Lowernode = make([]uint32, 0)
	return nn
}

func NewSafeNodeMap() *SafeNodeMap {
	nsnm := new(SafeNodeMap)
	nsnm.AllNode = make(map[uint32]*Node)
	return nsnm
}

func NewSafeRouteMap() *SafeRouteMap {
	nsrm := new(SafeRouteMap)
	nsrm.Route = make(map[uint32]string)
	return nsrm
}

/*-------------------------节点拓扑相关代码--------------------------*/
//将节点加入拓扑
func AddNodeToTopology(nodeid uint32, uppernodeid uint32) {
	Nooode.Lock()
	if _, ok := Nooode.AllNode[nodeid]; ok {
		Nooode.AllNode[nodeid].Uppernode = uppernodeid
	} else {
		tempnode := NewNode()
		Nooode.AllNode[nodeid] = tempnode
		Nooode.AllNode[nodeid].Uppernode = uppernodeid
	}
	if uppernodeid != 0 {
		Nooode.AllNode[uppernodeid].Lowernode = append(Nooode.AllNode[uppernodeid].Lowernode, nodeid)
	}
	Nooode.Unlock()
}

//将节点从拓扑中删除
func DelNodeFromTopology(nodeid uint32) {
	Nooode.Lock()
	if _, ok := Nooode.AllNode[nodeid]; ok {
		uppernode := Nooode.AllNode[nodeid].Uppernode
		if _, ok := Nooode.AllNode[uppernode]; ok {
			index := common.FindSpecFromSlice(nodeid, Nooode.AllNode[uppernode].Lowernode)
			Nooode.AllNode[uppernode].Lowernode = append(Nooode.AllNode[uppernode].Lowernode[:index], Nooode.AllNode[uppernode].Lowernode[index+1:]...)
		}
		Del(nodeid)
		readyToDel = append(readyToDel, nodeid)
		for _, value := range readyToDel {
			delete(Nooode.AllNode, value)
			delete(NodeStatus.Nodes, value)
			delete(NodeStatus.Nodenote, value)
		}
		readyToDel = make([]uint32, 0)
	}
	Nooode.Unlock()
}

//收集需要删除的节点
func Del(nodeid uint32) {
	for _, value := range Nooode.AllNode[nodeid].Lowernode {
		readyToDel = append(readyToDel, value)
		Del(value)
	}
}

//计算路由表
func CalRoute() {
	Nooode.Lock()
	for key, _ := range Nooode.AllNode {
		var temp []string = []string{}
		count := key

		if key == 0 {
			continue
		}

		for {
			if Nooode.AllNode[count].Uppernode != 0 && Nooode.AllNode[count].Uppernode != 1 {
				count = Nooode.AllNode[count].Uppernode
				node := common.Uint32Str(count)
				temp = append(temp, node)
			} else {
				common.StringReverse(temp)
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

// 显示节点拓扑详细信息
func ShowDetail() {
	if AdminStuff.StartNode != "0.0.0.0" {
		var nodes []uint32
		fmt.Printf("StartNode:[1] %s   note:%s\n", AdminStuff.StartNode, NodeStatus.Nodenote[1])
		for Nodeid, _ := range NodeStatus.Nodes {
			nodes = append(nodes, Nodeid)
		}
		CheckRange(nodes)
		for _, value := range nodes {
			fmt.Printf("Node[%s]: %s  note:%s\n", fmt.Sprint(value), NodeStatus.Nodes[value], NodeStatus.Nodenote[value])
		}
	} else {
		fmt.Println("There is no agent connected!")
	}
}

//显示节点层级关系
func ShowTree() {
	if AdminStuff.StartNode != "0.0.0.0" {
		var nodes []uint32
		Nooode.Lock()
		for key, _ := range Nooode.AllNode {
			nodes = append(nodes, key)
		}
		CheckRange(nodes)
		for _, value := range nodes {
			if value == 1 {
				fmt.Printf("StartNode[%s]'s child nodes:\n", fmt.Sprint(value))
				if len(Nooode.AllNode[value].Lowernode) == 0 {
					fmt.Println("\tThere is no child node for this one.")
				} else {
					for _, child := range Nooode.AllNode[value].Lowernode {
						fmt.Printf("\tNode [%s]\n", fmt.Sprint(child))
					}
				}
			} else {
				fmt.Printf("Node[%s]'s child nodes:\n", fmt.Sprint(value))
				if len(Nooode.AllNode[value].Lowernode) == 0 {
					fmt.Println("\tThere is no child node for this one.")
				} else {
					for _, child := range Nooode.AllNode[value].Lowernode {
						fmt.Printf("\tNode [%s]\n", fmt.Sprint(child))
					}
				}
			}
		}
		Nooode.Unlock()
	} else {
		fmt.Println("There is no agent connected!")
	}
}

//排序 || 防止map元素内存错位带来的展示效果错误
func CheckRange(nodes []uint32) {
	for m := len(nodes) - 1; m > 0; m-- {
		var flag bool = false
		for n := 0; n < m; n++ {
			if nodes[n] > nodes[n+1] {
				temp := nodes[n]
				nodes[n] = nodes[n+1]
				nodes[n+1] = temp
				flag = true
			}
		}
		if !flag {
			break
		}
	}
}

// 将节点加入拓扑
func AddToChain() {
	for {
		newNode := <-AdminStatus.NodesReadyToadd
		for key, value := range newNode {
			NodeStatus.Nodes[key] = value
		}
	}
}

//为node添加note
func AddNote(data []string, nodeid uint32) bool {
	info := ""
	data = data[1:len(data)]
	for _, i := range data {
		info = info + " " + i
	}
	if _, ok := NodeStatus.Nodenote[nodeid]; ok {
		NodeStatus.Nodenote[nodeid] = info
		return true
	}
	return false
}

//为node删除note
func DelNote(nodeid uint32) bool {
	if _, ok := NodeStatus.Nodenote[nodeid]; ok {
		NodeStatus.Nodenote[nodeid] = ""
		return true
	}
	return false
}

//admin端重连后，查找最大的nodeid值，以便之后的分配
func FindMax() {
	var tempAllNode []uint32
	for key, _ := range Nooode.AllNode {
		tempAllNode = append(tempAllNode, key)
	}
	CheckRange(tempAllNode)
	NodeIdAllocate = tempAllNode[len(tempAllNode)-1]
}
