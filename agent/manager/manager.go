/*
 * @Author: ph4ntom
 * @Date: 2021-03-23 19:01:26
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-03-26 14:16:43
 */
package manager

import (
	"Stowaway/share"
	"net"
)

const (
	SOCKS = iota
	S_CHECKIFSEQEXIST
	S_UPDATETCP
	S_UPDATEUDP
)

type Manager struct {
	File *share.MyFile

	socks map[uint64]*SocksStatus

	TaskChan chan *ManagerTask
	// socks
	socksTaskChan   chan *ManagerTask
	SocksResultChan chan *ManagerResult
	SocksReadyChan  chan bool
}

type ManagerTask struct {
	Category int
	Mode     int
	//socks
	SocksSequence   uint64
	SocksSocket     net.Conn
	SocksListener   *net.UDPConn
	SocksSourceAddr string
}

type ManagerResult struct {
	OK bool
	//socks
	SocksSeqExist bool
	DataChan      chan []byte
	SocksID       uint64
}

type SocksStatus struct {
	DataChan   chan []byte
	IsUDP      bool
	Conn       net.Conn
	Listener   *net.UDPConn
	SourceAddr string
}

func NewManager(file *share.MyFile) *Manager {
	manager := new(Manager)
	manager.File = file
	manager.socks = make(map[uint64]*SocksStatus)
	manager.TaskChan = make(chan *ManagerTask)
	manager.socksTaskChan = make(chan *ManagerTask)
	manager.SocksResultChan = make(chan *ManagerResult)
	manager.SocksReadyChan = make(chan bool)
	return manager
}

func (manager *Manager) Run() {
	go manager.socksRun()

	for {
		task := <-manager.TaskChan
		switch task.Category {
		case SOCKS:
			manager.socksTaskChan <- task
		}
	}
}

func (manager *Manager) socksRun() {
	for {
		task := <-manager.socksTaskChan
		switch task.Mode {
		case S_CHECKIFSEQEXIST:
			manager.checkIfSeqExist(task)
		case S_UPDATETCP:
			manager.updateTCP(task)
		case S_UPDATEUDP:
			manager.updateUDP(task)
		}
	}
}

func (manager *Manager) checkIfSeqExist(task *ManagerTask) {
	switch task.Category {
	case SOCKS:
		if _, ok := manager.socks[task.SocksSequence]; ok {
			manager.SocksResultChan <- &ManagerResult{SocksSeqExist: true, DataChan: manager.socks[task.SocksSequence].DataChan}
		} else {
			manager.socks[task.SocksSequence].DataChan = make(chan []byte, 50)                                                    // register it!
			manager.SocksResultChan <- &ManagerResult{SocksSeqExist: false, DataChan: manager.socks[task.SocksSequence].DataChan} // tell upstream result
		}
	}
}

func (manager *Manager) updateTCP(task *ManagerTask) {
	manager.socks[task.SocksSequence].IsUDP = false
	manager.socks[task.SocksSequence].Conn = task.SocksSocket
	manager.SocksReadyChan <- true // tell upstream work done
}

func (manager *Manager) updateUDP(task *ManagerTask) {
	manager.socks[task.SocksSequence].IsUDP = true
	manager.socks[task.SocksSequence].Listener = task.SocksListener
	manager.socks[task.SocksSequence].SourceAddr = task.SocksSourceAddr
	manager.SocksReadyChan <- true
}
