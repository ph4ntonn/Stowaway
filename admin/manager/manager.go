/*
 * @Author: ph4ntom
 * @Date: 2021-03-23 19:01:26
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-03-26 19:08:20
 */
package manager

import (
	"Stowaway/protocol"
	"Stowaway/share"
	"net"
)

const (
	SOCKS = iota
	S_NEWSOCKS
	S_ADDTCPSOCKET
	S_GETNEWSEQ
	S_GETTCPDATACHAN
	S_GETTCPDATACHAN_WITHOUTUUID
)

type Manager struct {
	// File
	File *share.MyFile
	//Socks
	socks5Seq         uint64
	socks5SeqMap      map[uint64]int
	socks5            map[int]*socks
	socksTaskChan     chan *ManagerTask
	Socks5TCPDataChan chan *protocol.SocksTCPData
	SocksResultChan   chan *ManagerResult
	// share
	TaskChan chan *ManagerTask
}

type ManagerTask struct {
	Category int
	Mode     int
	UUIDNum  int    // node idnum
	Seq      uint64 // seq
	//socks
	SocksPort      string
	SocksUsername  string
	SocksPassword  string
	SocksTCPSocket net.Conn
}

type ManagerResult struct {
	OK bool
	//socks
	SocksID     uint64
	TCPDataChan chan []byte
	UDPDataChan chan []byte
}

type socks struct {
	Port     string
	Username string
	Password string

	SocksStatus map[uint64]*socksStatus
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
	DataChan chan []byte
}

func NewManager(file *share.MyFile) *Manager {
	manager := new(Manager)
	manager.File = file
	manager.socks5 = make(map[int]*socks)
	manager.socks5SeqMap = make(map[uint64]int)
	manager.Socks5TCPDataChan = make(chan *protocol.SocksTCPData, 5)
	manager.TaskChan = make(chan *ManagerTask)
	manager.socksTaskChan = make(chan *ManagerTask)
	manager.SocksResultChan = make(chan *ManagerResult)
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
		case S_NEWSOCKS:
			manager.newSocks(task)
		case S_ADDTCPSOCKET:
			manager.addSocksTCPSocket(task)
		case S_GETNEWSEQ:
			manager.getSocksSeq(task)
		case S_GETTCPDATACHAN:
			manager.getTCPDataChan(task)
		case S_GETTCPDATACHAN_WITHOUTUUID:
			manager.getTCPDataChanWithoutUUID(task)
		}
	}
}

func (manager *Manager) ifSocksExist(uuidNum int) bool {
	if _, ok := manager.socks5[uuidNum]; ok { // check if element exist
		return true
	}
	return false
}

func (manager *Manager) newSocks(task *ManagerTask) {
	if !manager.ifSocksExist(task.UUIDNum) {
		manager.socks5[task.UUIDNum] = new(socks)
		manager.socks5[task.UUIDNum].Port = task.SocksPort
		manager.socks5[task.UUIDNum].Username = task.SocksUsername
		manager.socks5[task.UUIDNum].Password = task.SocksPassword
		manager.socks5[task.UUIDNum].SocksStatus = make(map[uint64]*socksStatus)
		manager.SocksResultChan <- &ManagerResult{OK: true}
	} else {
		manager.SocksResultChan <- &ManagerResult{OK: false}
	}
}

func (manager *Manager) addSocksTCPSocket(task *ManagerTask) {
	manager.socks5[task.UUIDNum].SocksStatus[task.Seq] = new(socksStatus)
	manager.socks5[task.UUIDNum].SocksStatus[task.Seq].tcp = new(tcpSocks) // no need to check if SocksStatus[task.Seq] exist,because it must exist
	manager.socks5[task.UUIDNum].SocksStatus[task.Seq].tcp.DataChan = make(chan []byte)
	manager.socks5[task.UUIDNum].SocksStatus[task.Seq].tcp.Conn = task.SocksTCPSocket
	manager.SocksResultChan <- &ManagerResult{}
}

func (manager *Manager) getSocksSeq(task *ManagerTask) {
	manager.socks5SeqMap[manager.socks5Seq] = task.UUIDNum
	manager.SocksResultChan <- &ManagerResult{SocksID: manager.socks5Seq}
	manager.socks5Seq++
}

func (manager *Manager) getTCPDataChan(task *ManagerTask) {
	manager.SocksResultChan <- &ManagerResult{TCPDataChan: manager.socks5[task.UUIDNum].SocksStatus[task.Seq].tcp.DataChan} // no need to check if SocksStatus[task.Seq] exist,because it must exist
}

func (manager *Manager) getTCPDataChanWithoutUUID(task *ManagerTask) {
	idNum := manager.socks5SeqMap[task.Seq] // no need to check if SocksStatus[task.Seq] exist,because it must exist
	if _, ok := manager.socks5[idNum]; ok { // Must check if element is really exist!
		if _, ok := manager.socks5[idNum].SocksStatus[task.Seq]; ok {
			manager.SocksResultChan <- &ManagerResult{
				OK:          true,
				TCPDataChan: manager.socks5[idNum].SocksStatus[task.Seq].tcp.DataChan,
			}
		} else {
			manager.SocksResultChan <- &ManagerResult{OK: false}
		}
	} else {
		manager.SocksResultChan <- &ManagerResult{OK: false}
	}
}
