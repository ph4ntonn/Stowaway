package manager

import (
	"fmt"
)

const (
	B_NEWBACKWARD = iota
	B_GETNEWSEQ
	B_ADDCONN
	B_CHECKBACKWARD
	B_GETDATACHAN
	B_GETDATACHAN_WITHOUTUUID
	B_CLOSETCP
	B_GETBACKWARDINFO
	B_GETSTOPRPORT
	B_CLOSESINGLE
	B_CLOSESINGLEALL
	B_FORCESHUTDOWN
)

type backwardManager struct {
	backwardSeq      uint64
	backwardSeqMap   map[uint64]*bwSeqRelationship   // map[seq](port+uuid) just for accelerate the speed of searching detail only by seq
	backwardMap      map[string]map[string]*backward // map[uuid][rport]backward status
	backwardReadyDel map[int]string

	BackwardMessChan chan interface{}
	BackwardReady    chan bool

	TaskChan   chan *BackwardTask
	ResultChan chan *backwardResult
}

type BackwardTask struct {
	Mode int
	UUID string // node uuid
	Seq  uint64 // seq

	LPort  string
	RPort  string
	Choice int
}

type backwardResult struct {
	OK bool

	DataChan     chan []byte
	BackwardSeq  uint64
	BackwardInfo []string
	RPort        string
}

type backward struct {
	localPort string

	backwardStatusMap map[uint64]*backwardStatus
}

type backwardStatus struct {
	dataChan chan []byte
}

type bwSeqRelationship struct {
	uuid  string
	rPort string
}

func newBackwardManager() *backwardManager {
	manager := new(backwardManager)

	manager.backwardMap = make(map[string]map[string]*backward)
	manager.backwardSeqMap = make(map[uint64]*bwSeqRelationship)
	manager.BackwardMessChan = make(chan interface{}, 5)

	manager.BackwardReady = make(chan bool)
	manager.TaskChan = make(chan *BackwardTask)
	manager.ResultChan = make(chan *backwardResult)

	return manager
}

func (manager *backwardManager) run() {
	for {
		task := <-manager.TaskChan

		switch task.Mode {
		case B_NEWBACKWARD:
			manager.newBackward(task)
		case B_GETNEWSEQ:
			manager.getNewSeq(task)
		case B_ADDCONN:
			manager.addConn(task)
		case B_CHECKBACKWARD:
			manager.checkBackward(task)
		case B_GETDATACHAN:
			manager.getDataChan(task)
		case B_GETDATACHAN_WITHOUTUUID:
			manager.getDatachanWithoutUUID(task)
		case B_CLOSETCP:
			manager.closeTCP(task)
		case B_GETBACKWARDINFO:
			manager.getBackwardInfo(task)
		case B_GETSTOPRPORT:
			manager.getStopRPort(task)
		case B_CLOSESINGLE:
			manager.closeSingle(task)
		case B_CLOSESINGLEALL:
			manager.closeSingleAll(task)
		case B_FORCESHUTDOWN:
			manager.forceShutdown(task)
		}
	}
}

// register a brand new backforward
// 2022.7.19 Fix nil pointer bug,thx to @zyylhn
func (manager *backwardManager) newBackward(task *BackwardTask) {
	if _, ok := manager.backwardMap[task.UUID]; !ok {
		manager.backwardMap[task.UUID] = make(map[string]*backward)
	}

	manager.backwardMap[task.UUID][task.RPort] = new(backward)
	manager.backwardMap[task.UUID][task.RPort].localPort = task.LPort
	manager.backwardMap[task.UUID][task.RPort].backwardStatusMap = make(map[uint64]*backwardStatus)

	manager.ResultChan <- &backwardResult{OK: true}
}

func (manager *backwardManager) getNewSeq(task *BackwardTask) {
	manager.backwardSeqMap[manager.backwardSeq] = &bwSeqRelationship{rPort: task.RPort, uuid: task.UUID}
	manager.ResultChan <- &backwardResult{BackwardSeq: manager.backwardSeq}
	manager.backwardSeq++
}

func (manager *backwardManager) addConn(task *BackwardTask) {
	if _, ok := manager.backwardSeqMap[task.Seq]; !ok {
		manager.ResultChan <- &backwardResult{OK: false}
		return
	}

	manager.backwardMap[task.UUID][task.RPort].backwardStatusMap[task.Seq] = new(backwardStatus)
	manager.backwardMap[task.UUID][task.RPort].backwardStatusMap[task.Seq].dataChan = make(chan []byte, 5)
	manager.ResultChan <- &backwardResult{OK: true}
}

func (manager *backwardManager) checkBackward(task *BackwardTask) {
	if _, ok := manager.backwardSeqMap[task.Seq]; !ok {
		manager.ResultChan <- &backwardResult{OK: false}
		return
	}

	if _, ok := manager.backwardMap[task.UUID][task.RPort].backwardStatusMap[task.Seq]; ok {
		manager.ResultChan <- &backwardResult{OK: true}
	} else {
		manager.ResultChan <- &backwardResult{OK: false}
	}

}

func (manager *backwardManager) getDataChan(task *BackwardTask) {
	if _, ok := manager.backwardSeqMap[task.Seq]; !ok {
		manager.ResultChan <- &backwardResult{OK: false}
		return
	}

	if _, ok := manager.backwardMap[task.UUID][task.RPort].backwardStatusMap[task.Seq]; ok {
		manager.ResultChan <- &backwardResult{
			OK:       true,
			DataChan: manager.backwardMap[task.UUID][task.RPort].backwardStatusMap[task.Seq].dataChan,
		}
	} else {
		manager.ResultChan <- &backwardResult{OK: false}
	}

}

func (manager *backwardManager) getDatachanWithoutUUID(task *BackwardTask) {
	if _, ok := manager.backwardSeqMap[task.Seq]; !ok {
		manager.ResultChan <- &backwardResult{OK: false}
		return
	}

	uuid := manager.backwardSeqMap[task.Seq].uuid
	rPort := manager.backwardSeqMap[task.Seq].rPort

	manager.ResultChan <- &backwardResult{
		OK:       true,
		DataChan: manager.backwardMap[uuid][rPort].backwardStatusMap[task.Seq].dataChan,
	}
}

func (manager *backwardManager) closeTCP(task *BackwardTask) {
	if _, ok := manager.backwardSeqMap[task.Seq]; !ok {
		return
	}

	uuid := manager.backwardSeqMap[task.Seq].uuid
	rPort := manager.backwardSeqMap[task.Seq].rPort

	close(manager.backwardMap[uuid][rPort].backwardStatusMap[task.Seq].dataChan)

	delete(manager.backwardMap[uuid][rPort].backwardStatusMap, task.Seq)
}

func (manager *backwardManager) getBackwardInfo(task *BackwardTask) {
	manager.backwardReadyDel = make(map[int]string)

	var backwardInfo []string
	infoNum := 1

	if _, ok := manager.backwardMap[task.UUID]; ok {
		backwardInfo = append(backwardInfo, "\r\n[0] All")
		for port, info := range manager.backwardMap[task.UUID] {
			manager.backwardReadyDel[infoNum] = port
			detail := fmt.Sprintf("\r\n[%d] Remote Port : %s , Local Port : %s , Current Active Connnections : %d", infoNum, port, info.localPort, len(info.backwardStatusMap))
			backwardInfo = append(backwardInfo, detail)
			infoNum++
		}
		manager.ResultChan <- &backwardResult{
			OK:           true,
			BackwardInfo: backwardInfo,
		}
	} else {
		backwardInfo = append(backwardInfo, "\r\nBackward service isn't running!")
		manager.ResultChan <- &backwardResult{
			OK:           false,
			BackwardInfo: backwardInfo,
		}
	}
}

func (manager *backwardManager) getStopRPort(task *BackwardTask) {
	manager.ResultChan <- &backwardResult{RPort: manager.backwardReadyDel[task.Choice]}
}

func (manager *backwardManager) closeSingle(task *BackwardTask) {
	rPort := task.RPort

	delete(manager.backwardMap[task.UUID], rPort)

	for seq, relationship := range manager.backwardSeqMap {
		if relationship.uuid == task.UUID && relationship.rPort == rPort {
			delete(manager.backwardSeqMap, seq)
		}
	}

	if len(manager.backwardMap[task.UUID]) == 0 {
		delete(manager.backwardMap, task.UUID)
	}

	manager.ResultChan <- &backwardResult{OK: true}
}

func (manager *backwardManager) closeSingleAll(task *BackwardTask) {
	for rPort := range manager.backwardMap[task.UUID] {
		delete(manager.backwardMap[task.UUID], rPort)
	}

	for seq, relationship := range manager.backwardSeqMap {
		if relationship.uuid == task.UUID {
			delete(manager.backwardSeqMap, seq)
		}
	}

	delete(manager.backwardMap, task.UUID)

	manager.ResultChan <- &backwardResult{OK: true}
}

func (manager *backwardManager) forceShutdown(task *BackwardTask) {
	if _, ok := manager.backwardMap[task.UUID]; ok {
		for rPort := range manager.backwardMap[task.UUID] {
			for seq, status := range manager.backwardMap[task.UUID][rPort].backwardStatusMap {
				close(status.dataChan)
				delete(manager.backwardMap[task.UUID][rPort].backwardStatusMap, seq)
			}
			delete(manager.backwardMap[task.UUID], rPort)
		}

		for seq, relationship := range manager.backwardSeqMap {
			if relationship.uuid == task.UUID {
				delete(manager.backwardSeqMap, seq)
			}
		}

		delete(manager.backwardMap, task.UUID)
	}

	manager.ResultChan <- &backwardResult{OK: true}
}
