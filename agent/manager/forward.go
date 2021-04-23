/*
 * @Author: ph4ntom
 * @Date: 2021-04-02 17:03:45
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-04-02 17:33:47
 */
package manager

import "net"

const (
	F_NEWFORWARD = iota
	F_GETDATACHAN
	F_UPDATEFORWARD
	F_CLOSETCP
	F_FORCESHUTDOWN
)

type forwardManager struct {
	forwardStatusMap map[uint64]*forwardStatus
	ForwardMessChan  chan interface{}

	TaskChan   chan *ForwardTask
	ResultChan chan *forwardResult
}

type ForwardTask struct {
	Mode int
	Seq  uint64

	ForwardSocket net.Conn
}

type forwardResult struct {
	OK bool

	DataChan chan []byte
}

type forwardStatus struct {
	dataChan chan []byte
	conn     net.Conn
}

func newForwardManager() *forwardManager {
	manager := new(forwardManager)

	manager.forwardStatusMap = make(map[uint64]*forwardStatus)
	manager.ForwardMessChan = make(chan interface{}, 5)

	manager.ResultChan = make(chan *forwardResult)
	manager.TaskChan = make(chan *ForwardTask)

	return manager
}

func (manager *forwardManager) run() {
	for {
		task := <-manager.TaskChan

		switch task.Mode {
		case F_NEWFORWARD:
			manager.newForward(task)
		case F_GETDATACHAN:
			manager.getDataChan(task)
		case F_UPDATEFORWARD:
			manager.updateForward(task)
		case F_CLOSETCP:
			manager.closeTCP(task)
		case F_FORCESHUTDOWN:
			manager.forceShutdown()
		}
	}
}

func (manager *forwardManager) newForward(task *ForwardTask) {
	manager.forwardStatusMap[task.Seq] = new(forwardStatus)
	manager.forwardStatusMap[task.Seq].dataChan = make(chan []byte, 5)
	manager.ResultChan <- &forwardResult{OK: true}
}

func (manager *forwardManager) updateForward(task *ForwardTask) {
	if _, ok := manager.forwardStatusMap[task.Seq]; ok {
		manager.forwardStatusMap[task.Seq].conn = task.ForwardSocket
		manager.ResultChan <- &forwardResult{OK: true}
	} else {
		manager.ResultChan <- &forwardResult{OK: false}
	}
}

func (manager *forwardManager) getDataChan(task *ForwardTask) {
	if _, ok := manager.forwardStatusMap[task.Seq]; ok {
		manager.ResultChan <- &forwardResult{
			OK:       true,
			DataChan: manager.forwardStatusMap[task.Seq].dataChan,
		}
	} else {
		manager.ResultChan <- &forwardResult{OK: false}
	}
}

func (manager *forwardManager) closeTCP(task *ForwardTask) {
	if manager.forwardStatusMap[task.Seq].conn != nil {
		manager.forwardStatusMap[task.Seq].conn.Close()
	}

	close(manager.forwardStatusMap[task.Seq].dataChan)

	delete(manager.forwardStatusMap, task.Seq)
}

func (manager *forwardManager) forceShutdown() {
	for seq, status := range manager.forwardStatusMap {
		if status.conn != nil {
			status.conn.Close()
		}

		close(status.dataChan)

		delete(manager.forwardStatusMap, seq)
	}

	manager.ResultChan <- &forwardResult{OK: true}
}
