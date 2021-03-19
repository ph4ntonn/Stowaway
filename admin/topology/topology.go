/*
 * @Author: ph4ntom
 * @Date: 2021-03-11 19:10:16
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-03-19 19:50:23
 */
package topology

import (
	"Stowaway/protocol"
	"Stowaway/utils"
	"fmt"
	"strings"
)

const (
	// Topology
	ADDNODE = iota
	GETNODEID
	CHECKNODE
	CALCULATE
	// User-friendly
	UPDATEDETAIL
	SHOWDETAIL
	SHOWTREE
	UPDATEMEMO
)

type Topology struct {
	nodes        map[int]*Node
	routes       map[int]string
	currentIDNum int
	TaskChan     chan *TopoTask
	ResultChan   chan *TopoResult
}

type Node struct {
	ID              string
	ParentID        string
	ChildrenID      []string
	CurrentUser     string
	CurrentHostname string
	CurrentIP       string
	Memo            string
}

type TopoTask struct {
	Mode     int
	ID       string
	IDNum    int
	Target   *Node
	HostName string
	UserName string
	Memo     string
	IsFirst  bool
}

type TopoResult struct {
	IsExist   bool
	NodeID    string
	RouteInfo map[int]string
}

func NewTopology() *Topology {
	topology := new(Topology)
	topology.nodes = make(map[int]*Node)
	topology.routes = make(map[int]string)
	topology.currentIDNum = 0
	topology.TaskChan = make(chan *TopoTask)
	topology.ResultChan = make(chan *TopoResult)
	return topology
}

func NewNode(id string, ip string) *Node {
	node := new(Node)
	node.ID = id
	node.CurrentIP = ip
	return node
}

func (topology *Topology) Run() {
	for {
		task := <-topology.TaskChan
		switch task.Mode {
		case ADDNODE:
			topology.addNode(task)
		case GETNODEID:
			topology.getNodeID(task)
		case CHECKNODE:
			topology.checkNode(task)
		case UPDATEDETAIL:
			topology.updateDetail(task)
		case SHOWDETAIL:
			topology.showDetail()
		case SHOWTREE:
			topology.showTree()
		case UPDATEMEMO:
			topology.updateMemo(task)
		case CALCULATE:
			topology.calculate()
		}
	}
}

func (topology *Topology) id2IDNum(id string) (idNum int) {
	for i := 0; i < len(topology.nodes); i++ {
		if topology.nodes[i].ID == id {
			idNum = i
			return
		}
	}
	return
}

func (topology *Topology) idNum2ID(idNum int) (id string) {
	return topology.nodes[idNum].ID
}

func (topology *Topology) getNodeID(task *TopoTask) {
	result := &TopoResult{
		NodeID: topology.idNum2ID(task.IDNum),
	}
	topology.ResultChan <- result
}

func (topology *Topology) checkNode(task *TopoTask) {
	result := new(TopoResult)
	_, ok := topology.nodes[task.IDNum]
	if ok {
		result.IsExist = true
	}
	topology.ResultChan <- result
}

func (topology *Topology) addNode(task *TopoTask) {
	if task.IsFirst {
		task.Target.ParentID = protocol.ADMIN_UUID
	} else {
		task.Target.ParentID = task.ID
		parentIDNum := topology.id2IDNum(task.ID)
		topology.nodes[parentIDNum].ChildrenID = append(topology.nodes[parentIDNum].ChildrenID, task.Target.ID)
	}

	topology.nodes[topology.currentIDNum] = task.Target
	topology.currentIDNum++
	topology.ResultChan <- &TopoResult{} // Just tell upstream: work done!
}

func (topology *Topology) calculate() {
	for currentID := range topology.nodes {
		var tempRoute []string
		tempID := currentID

		if topology.nodes[currentID].ParentID == protocol.ADMIN_UUID {
			topology.routes[currentID] = ""
			continue
		}

		for {
			if topology.nodes[tempID].ParentID != protocol.ADMIN_UUID {
				tempRoute = append(tempRoute, topology.nodes[tempID].ParentID)
				for i := 0; i < len(topology.nodes); i++ {
					if topology.nodes[i].ID == topology.nodes[tempID].ParentID {
						tempID = i
						break
					}
				}
			} else {
				utils.StringSliceReverse(tempRoute)
				finalRoute := strings.Join(tempRoute, ":")
				topology.routes[currentID] = finalRoute
				break
			}
		}
	}

	newRouteInfo := make(map[int]string) // Create brand new routeInfo
	for idNum, oldRoute := range topology.routes {
		newRouteInfo[idNum] = oldRoute
	}

	topology.ResultChan <- &TopoResult{RouteInfo: newRouteInfo}
}

func (topology *Topology) updateDetail(task *TopoTask) {
	idNum := topology.id2IDNum(task.ID)
	topology.nodes[idNum].CurrentUser = task.UserName
	topology.nodes[idNum].CurrentHostname = task.HostName
}

func (topology *Topology) showDetail() {
	for idNum, node := range topology.nodes {
		fmt.Printf("\nNode[%s] -> IP: %s  Hostname: %s  User: %s\nMemo: %s\n",
			utils.Int2Str(idNum),
			node.CurrentIP,
			node.CurrentHostname,
			node.CurrentUser,
			node.Memo,
		)
	}
	topology.ResultChan <- &TopoResult{} // Just tell upstream: work done!
}

func (topology *Topology) showTree() {
	for idNum, node := range topology.nodes {
		fmt.Printf("\nNode[%s]'s children ->\n", utils.Int2Str(idNum))
		for _, child := range node.ChildrenID {
			fmt.Printf("Node[%s]\n", utils.Int2Str(topology.id2IDNum(child)))
		}
	}
	topology.ResultChan <- &TopoResult{} // Just tell upstream: work done!
}

func (topology *Topology) updateMemo(task *TopoTask) {
	idNum := topology.id2IDNum(task.ID)
	topology.nodes[idNum].Memo = task.Memo
}
