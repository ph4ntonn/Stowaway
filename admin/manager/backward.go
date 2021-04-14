package manager

import (
	"net"
)

const (
	B_NEWBACKWARD = iota
	B_GETNEWSEQ
	B_ADDCONN
	B_UPDATEBACKWARD
	B_GETDATACHAN
	B_GETDATACHAN_WITHOUTUUID
	B_CLOSETCP
)

type backwardManager struct {
	backwardSeq    uint64
	backwardSeqMap map[uint64]*bwSeqRelationship   // map[seq](port+uuid) just for accelerate the speed of searching detail only by seq
	backwardMap    map[string]map[string]*backward // map[uuid][rport]backward status

	BackwardMessChan chan interface{}
	BackwardReady    chan bool

	TaskChan   chan *BackwardTask
	ResultChan chan *backwardResult
}

type BackwardTask struct {
	Mode int
	UUID string // node uuid
	Seq  uint64 // seq

	LPort string
	RPort string
	Conn  net.Conn
}

type backwardResult struct {
	OK bool

	DataChan    chan []byte
	BackwardSeq uint64
}

type backward struct {
	localPort string

	backwardStatusMap map[uint64]*backwardStatus
}

type backwardStatus struct {
	dataChan chan []byte
	conn     net.Conn
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
		case B_UPDATEBACKWARD:
			manager.updateBackward(task)
		case B_GETDATACHAN:
			manager.getDataChan(task)
		case B_GETDATACHAN_WITHOUTUUID:
			manager.getDatachanWithoutUUID(task)
		case B_CLOSETCP:
			manager.closeTCP(task)
		}
	}
}

// register a brand new backforward
func (manager *backwardManager) newBackward(task *BackwardTask) {
	if _, ok := manager.backwardMap[task.UUID]; !ok {
		manager.backwardMap = make(map[string]map[string]*backward)
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
	manager.backwardMap[task.UUID][task.RPort].backwardStatusMap[task.Seq] = new(backwardStatus)
	manager.backwardMap[task.UUID][task.RPort].backwardStatusMap[task.Seq].dataChan = make(chan []byte)
	manager.ResultChan <- &backwardResult{OK: true}
}

func (manager *backwardManager) updateBackward(task *BackwardTask) {
	if _, ok := manager.backwardSeqMap[task.Seq]; !ok {
		manager.ResultChan <- &backwardResult{OK: false}
		return
	}

	if _, ok := manager.backwardMap[task.UUID][task.RPort].backwardStatusMap[task.Seq]; ok {
		manager.backwardMap[task.UUID][task.RPort].backwardStatusMap[task.Seq].conn = task.Conn
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

	if _, ok := manager.backwardMap[task.UUID][task.RPort]; ok {
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

	if _, ok := manager.backwardMap[uuid][rPort].backwardStatusMap[task.Seq]; ok {
		manager.ResultChan <- &backwardResult{
			OK:       true,
			DataChan: manager.backwardMap[uuid][rPort].backwardStatusMap[task.Seq].dataChan,
		}
	} else {
		manager.ResultChan <- &backwardResult{OK: false}
	}
}

func (manager *backwardManager) closeTCP(task *BackwardTask) {
	if _, ok := manager.backwardSeqMap[task.Seq]; !ok {
		return
	}

	uuid := manager.backwardSeqMap[task.Seq].uuid
	rPort := manager.backwardSeqMap[task.Seq].rPort

	if manager.backwardMap[uuid][rPort].backwardStatusMap[task.Seq].conn != nil {
		manager.backwardMap[uuid][rPort].backwardStatusMap[task.Seq].conn.Close()
	}

	close(manager.backwardMap[uuid][rPort].backwardStatusMap[task.Seq].dataChan)

	delete(manager.backwardMap[uuid][rPort].backwardStatusMap, task.Seq)
}
