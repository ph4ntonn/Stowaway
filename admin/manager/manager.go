/*
 * @Author: ph4ntom
 * @Date: 2021-03-23 19:01:26
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-04-01 15:20:49
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
	S_GETUDPDATACHAN
	S_GETTCPDATACHAN_WITHOUTUUID
	S_GETUDPDATACHAN_WITHOUTUUID
	S_CLOSETCP
	S_GETUDPSTARTINFO
	S_UPDATEUDP
)

type Manager struct {
	// File
	File *share.MyFile
	//Socks
	socks5Seq        uint64
	socks5SeqMap     map[uint64]int
	socks5           map[int]*socks
	SocksTCPDataChan chan interface{} // accept both data and fin mess
	SocksUDPDataChan chan *protocol.SocksUDPData
	// share
	TaskChan   chan *ManagerTask
	ResultChan chan *ManagerResult
	Done       chan bool // try to avoid this situation: A routine ask to get chan -> after that a TCPFIN message come and closeTCP() is called to close chan -> routine doesn't know chan is closed,so continue to input message into it -> panic
}

type ManagerTask struct {
	Category int
	Mode     int
	UUIDNum  int    // node uuidNum
	Seq      uint64 // seq
	//socks
	SocksPort          string
	SocksUsername      string
	SocksPassword      string
	SocksTCPListener   net.Listener
	SocksTCPSocket     net.Conn
	SocksUDPListener   *net.UDPConn
	SocksUDPListenAddr string
}

type ManagerResult struct {
	OK      bool
	UUIDNum int
	//socks
	SocksID     uint64
	TCPDataChan chan []byte
	UDPDataChan chan []byte
	TCPAddr     string
}

type socks struct {
	Port     string
	Username string
	Password string
	Listener net.Listener

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
	DataChan   chan []byte
	ListenAddr string
	Listener   *net.UDPConn
}

func NewManager(file *share.MyFile) *Manager {
	manager := new(Manager)
	manager.File = file
	manager.socks5 = make(map[int]*socks)
	manager.socks5SeqMap = make(map[uint64]int)
	manager.SocksTCPDataChan = make(chan interface{}, 5)
	manager.SocksUDPDataChan = make(chan *protocol.SocksUDPData, 5)
	manager.TaskChan = make(chan *ManagerTask)
	manager.Done = make(chan bool)
	manager.ResultChan = make(chan *ManagerResult)
	return manager
}

func (manager *Manager) Run() {
	for {
		task := <-manager.TaskChan
		switch task.Category {
		case SOCKS:
			switch task.Mode {
			case S_NEWSOCKS:
				manager.newSocks(task)
			case S_ADDTCPSOCKET:
				manager.addSocksTCPSocket(task)
			case S_GETNEWSEQ:
				manager.getSocksSeq(task)
			case S_GETTCPDATACHAN:
				manager.getTCPDataChan(task)
			case S_GETUDPDATACHAN:
				manager.getUDPDataChan(task)
				<-manager.Done
			case S_GETTCPDATACHAN_WITHOUTUUID:
				manager.getTCPDataChanWithoutUUID(task)
				<-manager.Done
			case S_GETUDPDATACHAN_WITHOUTUUID:
				manager.getUDPDataChanWithoutUUID(task)
				<-manager.Done
			case S_CLOSETCP:
				manager.closeTCP(task)
			case S_GETUDPSTARTINFO:
				manager.getUDPStartInfo(task)
			case S_UPDATEUDP:
				manager.updateUDP(task)
			}
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
		manager.socks5[task.UUIDNum].Listener = task.SocksTCPListener
		manager.ResultChan <- &ManagerResult{OK: true}
	} else {
		manager.ResultChan <- &ManagerResult{OK: false}
	}
}

func (manager *Manager) addSocksTCPSocket(task *ManagerTask) {
	manager.socks5[task.UUIDNum].SocksStatus[task.Seq] = new(socksStatus)
	manager.socks5[task.UUIDNum].SocksStatus[task.Seq].tcp = new(tcpSocks) // no need to check if SocksStatus[task.Seq] exist,because it must exist
	manager.socks5[task.UUIDNum].SocksStatus[task.Seq].tcp.DataChan = make(chan []byte)
	manager.socks5[task.UUIDNum].SocksStatus[task.Seq].tcp.Conn = task.SocksTCPSocket
	manager.ResultChan <- &ManagerResult{}
}

func (manager *Manager) getSocksSeq(task *ManagerTask) {
	manager.socks5SeqMap[manager.socks5Seq] = task.UUIDNum
	manager.ResultChan <- &ManagerResult{SocksID: manager.socks5Seq}
	manager.socks5Seq++
}

func (manager *Manager) getTCPDataChan(task *ManagerTask) {
	manager.ResultChan <- &ManagerResult{TCPDataChan: manager.socks5[task.UUIDNum].SocksStatus[task.Seq].tcp.DataChan} // no need to check if SocksStatus[task.Seq] exist,because it must exist
}

func (manager *Manager) getUDPDataChan(task *ManagerTask) {
	if _, ok := manager.socks5[task.UUIDNum]; ok {
		if _, ok := manager.socks5[task.UUIDNum].SocksStatus[task.Seq]; ok {
			manager.ResultChan <- &ManagerResult{
				OK:          true,
				UDPDataChan: manager.socks5[task.UUIDNum].SocksStatus[task.Seq].udp.DataChan,
			}
		} else {
			manager.ResultChan <- &ManagerResult{OK: false}
		}
	} else {
		manager.ResultChan <- &ManagerResult{OK: false}
	}
}

func (manager *Manager) getTCPDataChanWithoutUUID(task *ManagerTask) {
	uuidNum := manager.socks5SeqMap[task.Seq] // no need to check if SocksStatus[task.Seq] exist,because it must exist
	if _, ok := manager.socks5[uuidNum]; ok { // Must check if element is really exist!
		if _, ok := manager.socks5[uuidNum].SocksStatus[task.Seq]; ok {
			manager.ResultChan <- &ManagerResult{
				OK:          true,
				TCPDataChan: manager.socks5[uuidNum].SocksStatus[task.Seq].tcp.DataChan,
			}
		} else {
			manager.ResultChan <- &ManagerResult{OK: false}
		}
	} else {
		manager.ResultChan <- &ManagerResult{OK: false}
	}
}

func (manager *Manager) getUDPDataChanWithoutUUID(task *ManagerTask) {
	uuidNum := manager.socks5SeqMap[task.Seq]
	if _, ok := manager.socks5[uuidNum]; ok {
		if _, ok := manager.socks5[uuidNum].SocksStatus[task.Seq]; ok {
			manager.ResultChan <- &ManagerResult{
				OK:          true,
				UDPDataChan: manager.socks5[uuidNum].SocksStatus[task.Seq].udp.DataChan,
			}
		} else {
			manager.ResultChan <- &ManagerResult{OK: false}
		}
	} else {
		manager.ResultChan <- &ManagerResult{OK: false}
	}
}

// close TCP include close UDP,cuz UDP's control channel is TCP,if TCP broken,UDP is also forced to be shutted down
func (manager *Manager) closeTCP(task *ManagerTask) {
	uuidNum := manager.socks5SeqMap[task.Seq]
	if _, ok := manager.socks5[uuidNum]; ok { // check if node is still online
		manager.socks5[uuidNum].SocksStatus[task.Seq].tcp.Conn.Close() // SocksStatus[task.Seq] must exist, no need to check(error)
		close(manager.socks5[uuidNum].SocksStatus[task.Seq].tcp.DataChan)

		if manager.socks5[uuidNum].SocksStatus[task.Seq].IsUDP {
			manager.socks5[uuidNum].SocksStatus[task.Seq].udp.Listener.Close()
			close(manager.socks5[uuidNum].SocksStatus[task.Seq].udp.DataChan)
		}

		delete(manager.socks5[uuidNum].SocksStatus, task.Seq)
	}
}

func (manager *Manager) getUDPStartInfo(task *ManagerTask) {
	uuidNum := manager.socks5SeqMap[task.Seq]
	if _, ok := manager.socks5[uuidNum]; ok { // check if node is still online
		if _, ok := manager.socks5[uuidNum].SocksStatus[task.Seq]; ok {
			manager.ResultChan <- &ManagerResult{
				OK:      true,
				TCPAddr: manager.socks5[uuidNum].SocksStatus[task.Seq].tcp.Conn.LocalAddr().(*net.TCPAddr).IP.String(),
				UUIDNum: uuidNum,
			}
		} else {
			manager.ResultChan <- &ManagerResult{
				OK:      false,
				UUIDNum: uuidNum,
			}
		}
	} else {
		manager.ResultChan <- &ManagerResult{
			OK:      false,
			UUIDNum: uuidNum,
		}
	}
}

func (manager *Manager) updateUDP(task *ManagerTask) {
	if _, ok := manager.socks5[task.UUIDNum]; ok {
		if _, ok := manager.socks5[task.UUIDNum].SocksStatus[task.Seq]; ok {
			manager.socks5[task.UUIDNum].SocksStatus[task.Seq].udp = new(udpSocks)
			manager.socks5[task.UUIDNum].SocksStatus[task.Seq].udp.DataChan = make(chan []byte)
			manager.socks5[task.UUIDNum].SocksStatus[task.Seq].udp.ListenAddr = task.SocksUDPListenAddr
			manager.socks5[task.UUIDNum].SocksStatus[task.Seq].udp.Listener = task.SocksUDPListener
			manager.ResultChan <- &ManagerResult{OK: true}
		} else {
			manager.ResultChan <- &ManagerResult{OK: false}
		}
	} else {
		manager.ResultChan <- &ManagerResult{OK: false}
	}
}
