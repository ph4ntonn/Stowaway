/*
 * @Author: ph4ntom
 * @Date: 2021-04-02 16:01:58
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-04-02 18:46:53
 */
package manager

import "net"

const (
	F_GETNEWSEQ = iota
	F_NEWFORWARD
	F_ADDCONN
)

type forwardManager struct {
	forwardSeq    uint64
	forwardSeqMap map[uint64]int
	forwardMap    map[int]map[string]*forward
	ForwardReady  chan bool

	TaskChan   chan *ForwardTask
	ResultChan chan *forwardResult
	Done       chan bool
}

type ForwardTask struct {
	Mode    int
	UUIDNum int    // node uuidNum
	Seq     uint64 // seq

	Port     string
	Listener net.Listener
	Conn     net.Conn
}

type forwardResult struct {
	OK      bool
	UUIDNum int

	ForwardSeq uint64
}

type forward struct {
	listener net.Listener

	forwardStatusMap map[uint64]*forwardStatus
}

type forwardStatus struct {
	dataChan chan []byte
	conn     net.Conn
}

func newForwardManager() *forwardManager {
	manager := new(forwardManager)

	manager.forwardSeqMap = make(map[uint64]int)
	manager.ForwardReady = make(chan bool, 1)
	manager.forwardMap = make(map[int]map[string]*forward)

	manager.TaskChan = make(chan *ForwardTask)
	manager.Done = make(chan bool)
	manager.ResultChan = make(chan *forwardResult)

	return manager
}

func (manager *forwardManager) run() {
	for {
		task := <-manager.TaskChan

		switch task.Mode {
		case F_NEWFORWARD:
			manager.newForward(task)
		case F_GETNEWSEQ:
			manager.getNewSeq(task)
		case F_ADDCONN:
			manager.addConn(task)
		}
	}
}

func (manager *forwardManager) newForward(task *ForwardTask) {
	if _, ok := manager.forwardMap[task.UUIDNum]; !ok {
		manager.forwardMap = make(map[int]map[string]*forward)
		manager.forwardMap[task.UUIDNum] = make(map[string]*forward)
	}
	// task.Port must exist
	manager.forwardMap[task.UUIDNum][task.Port] = new(forward)
	manager.forwardMap[task.UUIDNum][task.Port].listener = task.Listener
	manager.forwardMap[task.UUIDNum][task.Port].forwardStatusMap = make(map[uint64]*forwardStatus)

	manager.ResultChan <- &forwardResult{OK: true}
}

func (manager *forwardManager) getNewSeq(task *ForwardTask) {
	manager.forwardSeqMap[manager.forwardSeq] = task.UUIDNum
	manager.ResultChan <- &forwardResult{ForwardSeq: manager.forwardSeq}
	manager.forwardSeq++
}

func (manager *forwardManager) addConn(task *ForwardTask) {
	if _, ok := manager.forwardMap[task.UUIDNum][task.Port]; ok {
		manager.forwardMap[task.UUIDNum][task.Port].forwardStatusMap[task.Seq] = new(forwardStatus)
		manager.forwardMap[task.UUIDNum][task.Port].forwardStatusMap[task.Seq].conn = task.Conn
		manager.forwardMap[task.UUIDNum][task.Port].forwardStatusMap[task.Seq].dataChan = make(chan []byte)
		manager.ResultChan <- &forwardResult{OK: true}
	} else {
		manager.ResultChan <- &forwardResult{OK: false}
	}
}
