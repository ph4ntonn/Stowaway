/*
 * @Author: ph4ntom
 * @Date: 2021-04-02 16:01:58
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-04-02 17:29:34
 */
package manager

import (
	"net"
)

const (
	F_GETNEWSEQ = iota
	F_NEWFORWARD
)

type forwardManager struct {
	forwardSeq    uint64
	forwardSeqMap map[uint64]int
	forwardMap    map[int]*forward
	ForwardReady  chan bool

	TaskChan   chan *ForwardTask
	ResultChan chan *forwardResult
	Done       chan bool
}

type ForwardTask struct {
	Mode    int
	UUIDNum int    // node uuidNum
	Seq     uint64 // seq
}

type forwardResult struct {
	OK      bool
	UUIDNum int

	ForwardSeq uint64
}

type forward struct {
	dataChan chan []byte
	conn     net.Conn
}

func newForwardManager() *forwardManager {
	manager := new(forwardManager)

	manager.forwardSeqMap = make(map[uint64]int)
	manager.ForwardReady = make(chan bool, 1)
	manager.forwardMap = make(map[int]*forward)

	manager.TaskChan = make(chan *ForwardTask)
	manager.Done = make(chan bool)
	manager.ResultChan = make(chan *forwardResult)

	return manager
}

func (manager *forwardManager) run() {
	for {
		task := <-manager.TaskChan

		switch task.Mode {
		case F_GETNEWSEQ:
			manager.getForwardSeq(task)
		}
	}
}

func (manager *forwardManager) getForwardSeq(task *ForwardTask) {
	manager.forwardSeqMap[manager.forwardSeq] = task.UUIDNum
	manager.ResultChan <- &forwardResult{ForwardSeq: manager.forwardSeq}
	manager.forwardSeq++
}
