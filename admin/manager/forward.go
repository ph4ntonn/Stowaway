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
	F_GETDATACHAN
	F_GETDATACHAN_WITHOUTUUID
	F_CLOSETCP
)

type forwardManager struct {
	forwardSeq      uint64
	forwardSeqMap   map[uint64]*seqRelationship
	forwardMap      map[int]map[string]*forward
	ForwardDataChan chan interface{}
	ForwardReady    chan bool

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
	DataChan   chan []byte
}

type forward struct {
	listener net.Listener

	forwardStatusMap map[uint64]*forwardStatus
}

type forwardStatus struct {
	dataChan chan []byte
	conn     net.Conn
}

type seqRelationship struct {
	uuidNum int
	port    string
}

func newForwardManager() *forwardManager {
	manager := new(forwardManager)

	manager.forwardMap = make(map[int]map[string]*forward)
	manager.forwardSeqMap = make(map[uint64]*seqRelationship)
	manager.ForwardDataChan = make(chan interface{}, 5)
	manager.ForwardReady = make(chan bool, 1)

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
		case F_GETDATACHAN:
			manager.getDatachan(task)
		case F_GETDATACHAN_WITHOUTUUID:
			manager.getDatachanWithouUUID(task)
			<-manager.Done
		case F_CLOSETCP:
			manager.closeTCP(task)
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
	manager.forwardSeqMap[manager.forwardSeq] = &seqRelationship{uuidNum: task.UUIDNum, port: task.Port}
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

func (manager *forwardManager) getDatachan(task *ForwardTask) {
	if _, ok := manager.forwardMap[task.UUIDNum][task.Port]; ok {
		manager.ResultChan <- &forwardResult{
			OK:       true,
			DataChan: manager.forwardMap[task.UUIDNum][task.Port].forwardStatusMap[task.Seq].dataChan, // no need to check forwardStatusMap[task.Seq]
		}
	} else {
		manager.ResultChan <- &forwardResult{OK: false}
	}
}

func (manager *forwardManager) getDatachanWithouUUID(task *ForwardTask) {
	if _, ok := manager.forwardSeqMap[task.Seq]; !ok {
		manager.ResultChan <- &forwardResult{OK: false}
		return
	}

	uuidNum := manager.forwardSeqMap[task.Seq].uuidNum
	port := manager.forwardSeqMap[task.Seq].port

	if _, ok := manager.forwardMap[uuidNum][port].forwardStatusMap[task.Seq]; ok {
		manager.ResultChan <- &forwardResult{
			OK:       true,
			DataChan: manager.forwardMap[uuidNum][port].forwardStatusMap[task.Seq].dataChan,
		}
	} else {
		manager.ResultChan <- &forwardResult{OK: false}
	}
}

func (manager *forwardManager) closeTCP(task *ForwardTask) {
	if _, ok := manager.forwardSeqMap[task.Seq]; !ok {
		return
	}

	uuidNum := manager.forwardSeqMap[task.Seq].uuidNum
	port := manager.forwardSeqMap[task.Seq].port

	manager.forwardMap[uuidNum][port].forwardStatusMap[task.Seq].conn.Close()
	close(manager.forwardMap[uuidNum][port].forwardStatusMap[task.Seq].dataChan)

	delete(manager.forwardMap[uuidNum][port].forwardStatusMap, task.Seq)
}
