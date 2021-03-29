/*
 * @Author: ph4ntom
 * @Date: 2021-03-23 19:01:26
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-03-26 19:25:09
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
)

type Manager struct {
	//Fiel
	File *share.MyFile
	//Socks5
	socks             map[uint64]*socksStatus
	Socks5TCPDataChan chan *protocol.SocksTCPData
	socksTaskChan     chan *ManagerTask
	SocksResultChan   chan *ManagerResult
	SocksReadyChan    chan bool
	//share
	TaskChan chan *ManagerTask
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
	manager.Socks5TCPDataChan = make(chan *protocol.SocksTCPData, 5)
	manager.SocksResultChan = make(chan *ManagerResult)
	manager.SocksReadyChan = make(chan bool)

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
		}
	}
}

func (manager *Manager) getTCPDataChan(task *ManagerTask) {
	if _, ok := manager.socks[task.SocksSequence]; ok {
		manager.SocksResultChan <- &ManagerResult{
			SocksSeqExist: true,
			DataChan:      manager.socks[task.SocksSequence].tcp.DataChan,
		}
	} else {
		manager.socks[task.SocksSequence] = new(socksStatus)
		manager.socks[task.SocksSequence].tcp = new(tcpSocks)
		manager.socks[task.SocksSequence].tcp.DataChan = make(chan []byte, 5) // register it!
		manager.SocksResultChan <- &ManagerResult{
			SocksSeqExist: false,
			DataChan:      manager.socks[task.SocksSequence].tcp.DataChan,
		} // tell upstream result
	}
}

func (manager *Manager) updateTCP(task *ManagerTask) {
	manager.socks[task.SocksSequence].IsUDP = false
	manager.socks[task.SocksSequence].tcp.Conn = task.SocksSocket
	manager.SocksReadyChan <- true // tell upstream work done
}

func (manager *Manager) updateUDP(task *ManagerTask) {
	manager.socks[task.SocksSequence].IsUDP = true
	manager.socks[task.SocksSequence].udp.Listener = task.SocksListener
	manager.socks[task.SocksSequence].udp.SourceAddr = task.SocksSourceAddr
	manager.SocksReadyChan <- true
}
