/*
 * @Author: ph4ntom
 * @Date: 2021-03-11 19:10:16
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-04-03 13:26:10
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
	GETUUID
	CHECKNODE
	CALCULATE
	GETROUTE
	// User-friendly
	UPDATEDETAIL
	SHOWDETAIL
	SHOWTREE
	UPDATEMEMO
)

// IDNum is only for user-friendly,uuid is used internally
type Topology struct {
	nodes        map[int]*node // we use uuidNum as the map's key,that's the only special excpection
	currentIDNum int
	route        map[string]string // map[uuid]route
	TaskChan     chan *TopoTask
	ResultChan   chan *topoResult
}

type node struct {
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
	UUIDNum  int
	Target   *node
	HostName string
	UserName string
	Memo     string
	IsFirst  bool
}

type topoResult struct {
	IsExist bool
	UUID    string
	Route   string
}

func NewTopology() *Topology {
	topology := new(Topology)
	topology.nodes = make(map[int]*node)
	topology.route = make(map[string]string)
	topology.currentIDNum = 0
	topology.TaskChan = make(chan *TopoTask)
	topology.ResultChan = make(chan *topoResult)
	return topology
}

func NewNode(uuid string, ip string) *node {
	node := new(node)
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
		case GETUUID:
			topology.getUUID(task)
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
		case GETROUTE:
			topology.getRoute(task)
		}
	}
}

func (topology *Topology) id2IDNum(uuid string) (uuidNum int) {
	for i := 0; i < len(topology.nodes); i++ {
		if topology.nodes[i].uuid == uuid {
			uuidNum = i
			return
		}
	}
	return
}

func (topology *Topology) idNum2ID(uuidNum int) string {
	return topology.nodes[uuidNum].uuid
}

func (topology *Topology) getUUID(task *TopoTask) {
	topology.ResultChan <- &topoResult{UUID: topology.idNum2ID(task.UUIDNum)}
}

func (topology *Topology) checkNode(task *TopoTask) {
	if _, ok := topology.nodes[task.UUIDNum]; ok {
		topology.ResultChan <- &topoResult{IsExist: true}
	} else {
		topology.ResultChan <- &topoResult{IsExist: false}
	}
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

	topology.ResultChan <- &topoResult{} // Just tell upstream: work done!
}

func (topology *Topology) calculate() {
	newRouteInfo := make(map[string]string) // Create brand new routeInfo

	for currentIDNum := range topology.nodes {
		var tempRoute []string
		currentID := topology.nodes[currentIDNum].uuid
		tempIDNum := currentIDNum

		if topology.nodes[currentIDNum].parentUUID == protocol.ADMIN_UUID {
			newRouteInfo[currentID] = ""
			continue
		}

		for {
			if topology.nodes[tempIDNum].parentUUID != protocol.ADMIN_UUID {
				tempRoute = append(tempRoute, topology.nodes[tempIDNum].parentUUID)
				for i := 0; i < len(topology.nodes); i++ {
					if topology.nodes[i].uuid == topology.nodes[tempIDNum].parentUUID {
						tempIDNum = i
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

	topology.route = newRouteInfo

	topology.ResultChan <- &topoResult{} // Just tell upstream: work done!
}

func (topology *Topology) getRoute(task *TopoTask) {
	topology.ResultChan <- &topoResult{Route: topology.route[task.UUID]}
}

func (topology *Topology) updateDetail(task *TopoTask) {
	uuidNum := topology.id2IDNum(task.UUID)
	topology.nodes[uuidNum].currentUser = task.UserName
	topology.nodes[uuidNum].currentHostname = task.HostName
}

func (topology *Topology) showDetail() {
	for uuidNum, node := range topology.nodes {
		fmt.Printf("\nNode[%s] -> IP: %s  Hostname: %s  User: %s\nMemo: %s\n",
			utils.Int2Str(uuidNum),
			node.currentIP,
			node.currentHostname,
			node.currentUser,
			node.memo,
		)
	}

	topology.ResultChan <- &topoResult{} // Just tell upstream: work done!
}

func (topology *Topology) showTree() {
	for uuidNum, node := range topology.nodes {
		fmt.Printf("\nNode[%s]'s children ->\n", utils.Int2Str(uuidNum))
		for _, child := range node.childrenUUID {
			fmt.Printf("Node[%s]\n", utils.Int2Str(topology.id2IDNum(child)))
		}
	}

	topology.ResultChan <- &topoResult{} // Just tell upstream: work done!
}

func (topology *Topology) updateMemo(task *TopoTask) {
	uuidNum := topology.id2IDNum(task.UUID)
	topology.nodes[uuidNum].memo = task.Memo
}
