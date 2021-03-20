/*
 * @Author: ph4ntom
 * @Date: 2021-03-11 19:10:16
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-03-20 16:29:40
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
	currentIDNum int
	TaskChan     chan *TopoTask
	ResultChan   chan *TopoResult
}

type Node struct {
	uuid            string
	parentUUID      string
	childrenUUID    []string
	currentUser     string
	currentHostname string
	currentIP       string
	memo            string
}

type TopoTask struct {
	Mode     int
	UUID     string
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
	topology.currentIDNum = 0
	topology.TaskChan = make(chan *TopoTask)
	topology.ResultChan = make(chan *TopoResult)
	return topology
}

func NewNode(uuid string, ip string) *Node {
	node := new(Node)
	node.uuid = uuid
	node.currentIP = ip
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

func (topology *Topology) id2IDNum(uuid string) (idNum int) {
	for i := 0; i < len(topology.nodes); i++ {
		if topology.nodes[i].uuid == uuid {
			idNum = i
			return
		}
	}
	return
}

func (topology *Topology) idNum2ID(idNum int) string {
	return topology.nodes[idNum].uuid
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
		task.Target.parentUUID = protocol.ADMIN_UUID
	} else {
		task.Target.parentUUID = task.UUID
		parentIDNum := topology.id2IDNum(task.UUID)
		topology.nodes[parentIDNum].childrenUUID = append(topology.nodes[parentIDNum].childrenUUID, task.Target.uuid)
	}

	topology.nodes[topology.currentIDNum] = task.Target
	topology.currentIDNum++
	topology.ResultChan <- &TopoResult{} // Just tell upstream: work done!
}

func (topology *Topology) calculate() {
	newRouteInfo := make(map[int]string) // Create brand new routeInfo

	for currentID := range topology.nodes {
		var tempRoute []string
		tempID := currentID

		if topology.nodes[currentID].parentUUID == protocol.ADMIN_UUID {
			newRouteInfo[currentID] = ""
			continue
		}

		for {
			if topology.nodes[tempID].parentUUID != protocol.ADMIN_UUID {
				tempRoute = append(tempRoute, topology.nodes[tempID].parentUUID)
				for i := 0; i < len(topology.nodes); i++ {
					if topology.nodes[i].uuid == topology.nodes[tempID].parentUUID {
						tempID = i
						break
					}
				}
			} else {
				utils.StringSliceReverse(tempRoute)
				finalRoute := strings.Join(tempRoute, ":")
				newRouteInfo[currentID] = finalRoute
				break
			}
		}
	}

	topology.ResultChan <- &TopoResult{RouteInfo: newRouteInfo}
}

func (topology *Topology) updateDetail(task *TopoTask) {
	idNum := topology.id2IDNum(task.UUID)
	topology.nodes[idNum].currentUser = task.UserName
	topology.nodes[idNum].currentHostname = task.HostName
}

func (topology *Topology) showDetail() {
	for idNum, node := range topology.nodes {
		fmt.Printf("\nNode[%s] -> IP: %s  Hostname: %s  User: %s\nMemo: %s\n",
			utils.Int2Str(idNum),
			node.currentIP,
			node.currentHostname,
			node.currentUser,
			node.memo,
		)
	}
	topology.ResultChan <- &TopoResult{} // Just tell upstream: work done!
}

func (topology *Topology) showTree() {
	for idNum, node := range topology.nodes {
		fmt.Printf("\nNode[%s]'s children ->\n", utils.Int2Str(idNum))
		for _, child := range node.childrenUUID {
			fmt.Printf("Node[%s]\n", utils.Int2Str(topology.id2IDNum(child)))
		}
	}
	topology.ResultChan <- &TopoResult{} // Just tell upstream: work done!
}

func (topology *Topology) updateMemo(task *TopoTask) {
	idNum := topology.id2IDNum(task.UUID)
	topology.nodes[idNum].memo = task.Memo
}
