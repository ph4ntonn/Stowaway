/*
 * @Author: ph4ntom
 * @Date: 2021-03-23 19:01:26
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-03-29 17:48:22
 */
package manager

import (
	"Stowaway/protocol"
	"Stowaway/share"
	"net"
)

const (
	SOCKS = iota
	S_UPDATETCP
	S_UPDATEUDP
	S_GETTCPDATACHAN
	S_CLOSETCP
)

type Manager struct {
	//Fiel
	File *share.MyFile
	//Socks5
	socks            map[uint64]*socksStatus
	socksTaskChan    chan *ManagerTask
	SocksTCPDataChan chan *protocol.SocksTCPData
	SocksResultChan  chan *ManagerResult
	//share
	TaskChan chan *ManagerTask
}

type ManagerTask struct {
	Category int
	Mode     int
	Seq      uint64
	//socks
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

type socksStatus struct {
	IsUDP bool
	tcp   *tcpSocks
	udp   *udpSocks
}

type tcpSocks struct {
	DataChan chan []byte
	Conn     net.Conn
}

type udpSocks struct {
	DataChan   chan []byte
	Listener   *net.UDPConn
	SourceAddr string
}

func NewManager(file *share.MyFile) *Manager {
	manager := new(Manager)
	manager.File = file

	manager.socks = make(map[uint64]*socksStatus)
	manager.socksTaskChan = make(chan *ManagerTask)
	manager.SocksTCPDataChan = make(chan *protocol.SocksTCPData, 5)
	manager.SocksResultChan = make(chan *ManagerResult)

	manager.TaskChan = make(chan *ManagerTask)
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
		case S_GETTCPDATACHAN:
			manager.getTCPDataChan(task)
		case S_UPDATETCP:
			manager.updateTCP(task)
		case S_UPDATEUDP:
			manager.updateUDP(task)
		case S_CLOSETCP:
			manager.closeTCP(task)
		}
	}
}

func (manager *Manager) getTCPDataChan(task *ManagerTask) {
	if _, ok := manager.socks[task.Seq]; ok {
		manager.SocksResultChan <- &ManagerResult{
			SocksSeqExist: true,
			DataChan:      manager.socks[task.Seq].tcp.DataChan,
		}
	} else {
		manager.socks[task.Seq] = new(socksStatus)
		manager.socks[task.Seq].tcp = new(tcpSocks)
		manager.socks[task.Seq].tcp.DataChan = make(chan []byte, 5) // register it!
		manager.SocksResultChan <- &ManagerResult{
			SocksSeqExist: false,
			DataChan:      manager.socks[task.Seq].tcp.DataChan,
		} // tell upstream result
	}
}

func (manager *Manager) updateTCP(task *ManagerTask) {
	if _, ok := manager.socks[task.Seq]; ok {
		manager.socks[task.Seq].IsUDP = false
		manager.socks[task.Seq].tcp.Conn = task.SocksSocket
	} else {
		task.SocksSocket.Close() // avoid the scenario that admin conn ask to fin before "socks.buildConn()" call "updateTCP()"
	}

	manager.SocksResultChan <- &ManagerResult{} // tell upstream work done
}

func (manager *Manager) updateUDP(task *ManagerTask) {
	manager.socks[task.Seq].IsUDP = true
	manager.socks[task.Seq].udp.Listener = task.SocksListener
	manager.socks[task.Seq].udp.SourceAddr = task.SocksSourceAddr
	manager.SocksResultChan <- &ManagerResult{} // tell upstream work done
}

func (manager *Manager) closeTCP(task *ManagerTask) {
	if manager.socks[task.Seq].tcp.Conn != nil {
		manager.socks[task.Seq].tcp.Conn.Close() // avoid the scenario that admin conn ask to fin before "socks.buildConn()" call "updateTCP()"
	}

	close(manager.socks[task.Seq].tcp.DataChan)

	if manager.socks[task.Seq].IsUDP {
		manager.socks[task.Seq].udp.Listener.Close()
		close(manager.socks[task.Seq].udp.DataChan)
	}

	delete(manager.socks, task.Seq) // upstream not waiting
}
