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
)

type forwardManager struct {
	forwardStatusMap map[uint64]*forwardStatus
	ForwardDataChan  chan interface{}

	TaskChan   chan *ForwardTask
	ResultChan chan *forwardResult
	Done       chan bool
}

type ForwardTask struct {
	Category int
	Mode     int
	Seq      uint64

	ForwardSocket net.Conn
}

type forwardResult struct {
	OK bool
}

type forwardStatus struct {
	dataChan chan []byte
	conn     net.Conn
}

func newForwardManager() *forwardManager {
	manager := new(forwardManager)

	manager.forwardStatusMap = make(map[uint64]*forwardStatus)
	manager.ForwardDataChan = make(chan interface{}, 5)

	manager.ResultChan = make(chan *forwardResult)
	manager.TaskChan = make(chan *ForwardTask)
	manager.Done = make(chan bool)

	return manager
}

func (manager *forwardManager) run() {
	for {
		task := <-manager.TaskChan

		switch task.Mode {
		case F_NEWFORWARD:
			manager.newForward(task)
		}
	}
}

func (manager *forwardManager) newForward(task *ForwardTask) {
	manager.forwardStatusMap[task.Seq] = new(forwardStatus)
	manager.forwardStatusMap[task.Seq].dataChan = make(chan []byte, 5)
	manager.forwardStatusMap[task.Seq].conn = task.ForwardSocket
	manager.ResultChan <- &forwardResult{OK: true}
}
