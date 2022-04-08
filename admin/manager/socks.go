/*
 * @Author: ph4ntom
 * @Date: 2021-04-02 15:43:04
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-04-02 17:29:12
 */
package manager

import (
	"fmt"
	"net"
)

const (
	S_NEWSOCKS = iota
	S_ADDTCPSOCKET
	S_GETNEWSEQ
	S_GETTCPDATACHAN
	S_GETUDPDATACHAN
	S_GETTCPDATACHAN_WITHOUTUUID
	S_GETUDPDATACHAN_WITHOUTUUID
	S_CLOSETCP
	S_GETUDPSTARTINFO
	S_UPDATEUDP
	S_GETSOCKSINFO
	S_CLOSESOCKS
	S_FORCESHUTDOWN
)

type socksManager struct {
	socksSeq    uint64
	socksSeqMap map[uint64]string // map[seq]uuid  just for accelerate the speed of searching detail only by seq
	socksMap    map[string]*socks // map[uuid]socks's detail

	SocksMessChan chan interface{}
	SocksReady    chan bool

	TaskChan   chan *SocksTask
	ResultChan chan *socksResult
	Done       chan bool
}

type SocksTask struct {
	Mode int
	UUID string // node uuid
	Seq  uint64 // seq

	SocksPort        string
	SocksUsername    string
	SocksPassword    string
	SocksTCPListener net.Listener
	SocksTCPSocket   net.Conn
	SocksUDPListener *net.UDPConn
}

type socksResult struct {
	OK   bool
	UUID string

	SocksSeq    uint64
	TCPAddr     string
	SocksInfo   string
	TCPDataChan chan []byte
	UDPDataChan chan []byte
}

type socks struct {
	port     string
	username string
	password string
	listener net.Listener

	socksStatusMap map[uint64]*socksStatus
}

type socksStatus struct {
	isUDP bool
	tcp   *tcpSocks
	udp   *udpSocks
}

type tcpSocks struct {
	dataChan chan []byte
	conn     net.Conn
}

type udpSocks struct {
	dataChan chan []byte
	listener *net.UDPConn
}

func newSocksManager() *socksManager {
	manager := new(socksManager)

	manager.socksMap = make(map[string]*socks)
	manager.socksSeqMap = make(map[uint64]string)
	manager.SocksMessChan = make(chan interface{}, 5)
	manager.SocksReady = make(chan bool)

	manager.TaskChan = make(chan *SocksTask)
	manager.ResultChan = make(chan *socksResult)
	manager.Done = make(chan bool)

	return manager
}

func (manager *socksManager) run() {
	for {
		task := <-manager.TaskChan

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
		case S_GETSOCKSINFO:
			manager.getSocksInfo(task)
		case S_CLOSESOCKS:
			manager.closeSocks(task)
		case S_FORCESHUTDOWN:
			manager.forceShutdown(task)
		}
	}
}

func (manager *socksManager) newSocks(task *SocksTask) {
	if _, ok := manager.socksMap[task.UUID]; !ok {
		manager.socksMap[task.UUID] = new(socks)
		manager.socksMap[task.UUID].port = task.SocksPort
		manager.socksMap[task.UUID].username = task.SocksUsername
		manager.socksMap[task.UUID].password = task.SocksPassword
		manager.socksMap[task.UUID].socksStatusMap = make(map[uint64]*socksStatus)
		manager.socksMap[task.UUID].listener = task.SocksTCPListener
		manager.ResultChan <- &socksResult{OK: true}
	} else {
		manager.ResultChan <- &socksResult{OK: false}
	}
}

func (manager *socksManager) addSocksTCPSocket(task *SocksTask) {
	if _, ok := manager.socksMap[task.UUID]; ok {
		manager.socksMap[task.UUID].socksStatusMap[task.Seq] = new(socksStatus)
		manager.socksMap[task.UUID].socksStatusMap[task.Seq].tcp = new(tcpSocks) // no need to check if socksStatusMap[task.Seq] exist,because it must exist
		manager.socksMap[task.UUID].socksStatusMap[task.Seq].tcp.dataChan = make(chan []byte, 5)
		manager.socksMap[task.UUID].socksStatusMap[task.Seq].tcp.conn = task.SocksTCPSocket
		manager.ResultChan <- &socksResult{OK: true}
	} else {
		manager.ResultChan <- &socksResult{OK: false}
	}
}

func (manager *socksManager) getSocksSeq(task *SocksTask) {
	// Use seqmap to record the UUIDNum <-> Seq relationship to make search quicker
	manager.socksSeqMap[manager.socksSeq] = task.UUID
	manager.ResultChan <- &socksResult{SocksSeq: manager.socksSeq}
	manager.socksSeq++
}

func (manager *socksManager) getTCPDataChan(task *SocksTask) {
	if _, ok := manager.socksMap[task.UUID]; ok {
		manager.ResultChan <- &socksResult{
			OK:          true,
			TCPDataChan: manager.socksMap[task.UUID].socksStatusMap[task.Seq].tcp.dataChan,
		}
	} else {
		manager.ResultChan <- &socksResult{OK: false}
	}
}

func (manager *socksManager) getUDPDataChan(task *SocksTask) {
	if _, ok := manager.socksMap[task.UUID]; ok {
		if _, ok := manager.socksMap[task.UUID].socksStatusMap[task.Seq]; ok {
			manager.ResultChan <- &socksResult{
				OK:          true,
				UDPDataChan: manager.socksMap[task.UUID].socksStatusMap[task.Seq].udp.dataChan,
			}
		} else {
			manager.ResultChan <- &socksResult{OK: false}
		}
	} else {
		manager.ResultChan <- &socksResult{OK: false}
	}
}

func (manager *socksManager) getTCPDataChanWithoutUUID(task *SocksTask) {
	if _, ok := manager.socksSeqMap[task.Seq]; !ok {
		manager.ResultChan <- &socksResult{OK: false}
		return
	}

	uuid := manager.socksSeqMap[task.Seq]
	// if "manager.socksSeqMap[task.Seq]" exist, "manager.socksMap[uuid]" must exist too
	if _, ok := manager.socksMap[uuid].socksStatusMap[task.Seq]; ok {
		manager.ResultChan <- &socksResult{
			OK:          true,
			TCPDataChan: manager.socksMap[uuid].socksStatusMap[task.Seq].tcp.dataChan,
		}
	} else {
		manager.ResultChan <- &socksResult{OK: false}
	}
}

func (manager *socksManager) getUDPDataChanWithoutUUID(task *SocksTask) {
	if _, ok := manager.socksSeqMap[task.Seq]; !ok {
		manager.ResultChan <- &socksResult{OK: false}
		return
	}

	uuid := manager.socksSeqMap[task.Seq]
	// manager.socksMap[uuid] must exist if manager.socksSeqMap[task.Seq] exist
	if _, ok := manager.socksMap[uuid].socksStatusMap[task.Seq]; ok {
		manager.ResultChan <- &socksResult{
			OK:          true,
			UDPDataChan: manager.socksMap[uuid].socksStatusMap[task.Seq].udp.dataChan,
		}
	} else {
		manager.ResultChan <- &socksResult{OK: false}
	}
}

// close TCP include close UDP,cuz UDP's control channel is TCP,if TCP broken,UDP is also forced to be shutted down
func (manager *socksManager) closeTCP(task *SocksTask) {
	if _, ok := manager.socksSeqMap[task.Seq]; !ok {
		return
	}

	uuid := manager.socksSeqMap[task.Seq]

	// bugfix: In order to avoid data loss,so not close conn&listener here.Thx to @lz520520
	close(manager.socksMap[uuid].socksStatusMap[task.Seq].tcp.dataChan)

	if manager.socksMap[uuid].socksStatusMap[task.Seq].isUDP {
		close(manager.socksMap[uuid].socksStatusMap[task.Seq].udp.dataChan)
	}

	delete(manager.socksMap[uuid].socksStatusMap, task.Seq)
}

func (manager *socksManager) getUDPStartInfo(task *SocksTask) {
	if _, ok := manager.socksSeqMap[task.Seq]; !ok {
		manager.ResultChan <- &socksResult{OK: false}
		return
	}

	uuid := manager.socksSeqMap[task.Seq]

	if _, ok := manager.socksMap[uuid].socksStatusMap[task.Seq]; ok {
		manager.ResultChan <- &socksResult{
			OK:      true,
			TCPAddr: manager.socksMap[uuid].socksStatusMap[task.Seq].tcp.conn.LocalAddr().(*net.TCPAddr).IP.String(),
			UUID:    uuid,
		}
	} else {
		manager.ResultChan <- &socksResult{OK: false}
	}
}

func (manager *socksManager) updateUDP(task *SocksTask) {
	if _, ok := manager.socksMap[task.UUID]; ok {
		if _, ok := manager.socksMap[task.UUID].socksStatusMap[task.Seq]; ok {
			manager.socksMap[task.UUID].socksStatusMap[task.Seq].isUDP = true
			manager.socksMap[task.UUID].socksStatusMap[task.Seq].udp = new(udpSocks)
			manager.socksMap[task.UUID].socksStatusMap[task.Seq].udp.dataChan = make(chan []byte, 5)
			manager.socksMap[task.UUID].socksStatusMap[task.Seq].udp.listener = task.SocksUDPListener
			manager.ResultChan <- &socksResult{OK: true}
		} else {
			manager.ResultChan <- &socksResult{OK: false}
		}
	} else {
		manager.ResultChan <- &socksResult{OK: false}
	}
}

func (manager *socksManager) getSocksInfo(task *SocksTask) {
	if _, ok := manager.socksMap[task.UUID]; ok {
		if manager.socksMap[task.UUID].username == "" && manager.socksMap[task.UUID].password == "" {
			info := fmt.Sprintf("\r\nSocks Info ---> ListenAddr: 0.0.0.0:%s    Username: <null>    Password: <null>",
				manager.socksMap[task.UUID].port,
			)
			manager.ResultChan <- &socksResult{
				OK:        true,
				SocksInfo: info,
			}
		} else {
			info := fmt.Sprintf("\r\nSocks Info ---> ListenAddr: 0.0.0.0:%s    Username: %s    Password: %s",
				manager.socksMap[task.UUID].port,
				manager.socksMap[task.UUID].username,
				manager.socksMap[task.UUID].password,
			)
			manager.ResultChan <- &socksResult{
				OK:        true,
				SocksInfo: info,
			}
		}
	} else {
		info := "\r\nSocks service isn't running!"
		manager.ResultChan <- &socksResult{
			OK:        false,
			SocksInfo: info,
		}
	}
}

func (manager *socksManager) closeSocks(task *SocksTask) {
	manager.socksMap[task.UUID].listener.Close()
	for seq, status := range manager.socksMap[task.UUID].socksStatusMap {
		// bugfix: In order to avoid data loss,so not close conn&listener here.Thx to @lz520520
		close(status.tcp.dataChan)
		if status.isUDP {
			close(status.udp.dataChan)
		}
		delete(manager.socksMap[task.UUID].socksStatusMap, seq)
	}

	for seq, uuid := range manager.socksSeqMap {
		if uuid == task.UUID {
			delete(manager.socksSeqMap, seq)
		}
	}

	delete(manager.socksMap, task.UUID) // we delete corresponding "socksMap"
	manager.ResultChan <- &socksResult{OK: true}
}

func (manager *socksManager) forceShutdown(task *SocksTask) {
	if _, ok := manager.socksMap[task.UUID]; ok {
		manager.closeSocks(task)
	} else {
		manager.ResultChan <- &socksResult{OK: true}
	}
}
