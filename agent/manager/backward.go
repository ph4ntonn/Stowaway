package manager

import (
	"net"
)

const (
	B_NEWBACKWARD = iota
	B_GETSEQCHAN
	B_ADDCONN
	B_GETDATACHAN
	B_GETDATACHAN_WITHOUTUUID
	B_CLOSETCP
)

type backwardManager struct {
	backwardSeqMap   map[uint64]string
	backwardMap      map[string]*backward
	BackwardMessChan chan interface{}

	TaskChan   chan *BackwardTask
	ResultChan chan *backwardResult
	SeqReady   chan bool
}

type BackwardTask struct {
	Mode int
	Seq  uint64

	Listener       net.Listener
	RPort          string
	BackwardSocket net.Conn
}

type backwardResult struct {
	OK bool

	SeqChan  chan uint64
	DataChan chan []byte
}

type backward struct {
	listener net.Listener
	seqChan  chan uint64

	backwardStatusMap map[uint64]*backwardStatus
}

type backwardStatus struct {
	dataChan chan []byte
	conn     net.Conn
}

func newBackwardManager() *backwardManager {
	manager := new(backwardManager)

	manager.backwardSeqMap = make(map[uint64]string)
	manager.backwardMap = make(map[string]*backward)
	manager.BackwardMessChan = make(chan interface{}, 5)

	manager.ResultChan = make(chan *backwardResult)
	manager.TaskChan = make(chan *BackwardTask)
	manager.SeqReady = make(chan bool)

	return manager
}

func (manager *backwardManager) run() {
	for {
		task := <-manager.TaskChan

		switch task.Mode {
		case B_NEWBACKWARD:
			manager.newBackward(task)
		case B_GETSEQCHAN:
			manager.getSeqChan(task)
		case B_ADDCONN:
			manager.addConn(task)
		case B_GETDATACHAN:
			manager.getDataChan(task)
		case B_GETDATACHAN_WITHOUTUUID:
			manager.getDatachanWithoutUUID(task)
		case B_CLOSETCP:
			manager.closeTCP(task)
		}
	}
}

func (manager *backwardManager) newBackward(task *BackwardTask) {
	manager.backwardMap[task.RPort] = new(backward)
	manager.backwardMap[task.RPort].listener = task.Listener
	manager.backwardMap[task.RPort].backwardStatusMap = make(map[uint64]*backwardStatus)
	manager.backwardMap[task.RPort].seqChan = make(chan uint64)
	manager.ResultChan <- &backwardResult{OK: true}
}

func (manager *backwardManager) getSeqChan(task *BackwardTask) {
	if _, ok := manager.backwardMap[task.RPort]; ok {
		manager.ResultChan <- &backwardResult{
			OK:      true,
			SeqChan: manager.backwardMap[task.RPort].seqChan,
		}
	} else {
		manager.ResultChan <- &backwardResult{OK: false}
	}
}

func (manager *backwardManager) addConn(task *BackwardTask) {
	if _, ok := manager.backwardMap[task.RPort]; ok {
		manager.backwardSeqMap[task.Seq] = task.RPort
		manager.backwardMap[task.RPort].backwardStatusMap[task.Seq] = new(backwardStatus)
		manager.backwardMap[task.RPort].backwardStatusMap[task.Seq].conn = task.BackwardSocket
		manager.backwardMap[task.RPort].backwardStatusMap[task.Seq].dataChan = make(chan []byte, 5)
		manager.ResultChan <- &backwardResult{OK: true}
	} else {
		manager.ResultChan <- &backwardResult{OK: false}
	}
}

func (manager *backwardManager) getDataChan(task *BackwardTask) {
	if _, ok := manager.backwardMap[task.RPort]; ok {
		manager.ResultChan <- &backwardResult{
			OK:       true,
			DataChan: manager.backwardMap[task.RPort].backwardStatusMap[task.Seq].dataChan,
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

	rPort := manager.backwardSeqMap[task.Seq]

	if _, ok := manager.backwardMap[rPort]; ok {
		manager.ResultChan <- &backwardResult{
			OK:       true,
			DataChan: manager.backwardMap[rPort].backwardStatusMap[task.Seq].dataChan,
		}
	} else {
		manager.ResultChan <- &backwardResult{OK: false}
	}
}

func (manager *backwardManager) closeTCP(task *BackwardTask) {
	if _, ok := manager.backwardSeqMap[task.Seq]; !ok {
		return
	}

	rPort := manager.backwardSeqMap[task.Seq]

	manager.backwardMap[rPort].backwardStatusMap[task.Seq].conn.Close()

	close(manager.backwardMap[rPort].backwardStatusMap[task.Seq].dataChan)

	delete(manager.backwardMap[rPort].backwardStatusMap, task.Seq)
}
