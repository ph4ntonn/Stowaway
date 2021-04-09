/*
 * @Author: ph4ntom
 * @Date: 2021-04-02 16:53:15
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-04-02 17:33:10
 */
package manager

import (
	"net"
)

const (
	S_UPDATETCP = iota
	S_UPDATEUDP
	S_UPDATEUDPHEADER
	S_GETTCPDATACHAN
	S_GETUDPCHANS
	S_GETUDPHEADER
	S_CLOSETCP
	S_CHECKSOCKSREADY
)

type socksManager struct {
	socksStatusMap map[uint64]*socksStatus
	SocksMessChan  chan interface{}

	TaskChan   chan *SocksTask
	ResultChan chan *socksResult
	Done       chan bool
}

type SocksTask struct {
	Mode int
	Seq  uint64

	SocksSocket     net.Conn
	SocksListener   *net.UDPConn
	SocksHeaderAddr string
	SocksHeader     []byte
}

type socksResult struct {
	OK bool

	SocksSeqExist  bool
	DataChan       chan []byte
	ReadyChan      chan string
	SocksID        uint64
	SocksUDPHeader []byte
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
	dataChan    chan []byte
	readyChan   chan string
	headerPairs map[string][]byte
	listener    *net.UDPConn
}

func newSocksManager() *socksManager {
	manager := new(socksManager)

	manager.socksStatusMap = make(map[uint64]*socksStatus)
	manager.SocksMessChan = make(chan interface{}, 5)

	manager.ResultChan = make(chan *socksResult)
	manager.TaskChan = make(chan *SocksTask)
	manager.Done = make(chan bool)

	return manager
}

func (manager *socksManager) run() {
	for {
		task := <-manager.TaskChan

		switch task.Mode {
		case S_GETTCPDATACHAN:
			manager.getTCPDataChan(task)
			<-manager.Done
		case S_GETUDPCHANS:
			manager.getUDPChans(task)
			<-manager.Done
		case S_GETUDPHEADER:
			manager.getUDPHeader(task)
		case S_UPDATETCP:
			manager.updateTCP(task)
		case S_UPDATEUDP:
			manager.updateUDP(task)
		case S_UPDATEUDPHEADER:
			manager.updateUDPHeader(task)
		case S_CLOSETCP:
			manager.closeTCP(task)
		case S_CHECKSOCKSREADY:
			manager.checkSocksReady()
		}
	}
}

func (manager *socksManager) getTCPDataChan(task *SocksTask) {
	if _, ok := manager.socksStatusMap[task.Seq]; ok {
		manager.ResultChan <- &socksResult{
			SocksSeqExist: true,
			DataChan:      manager.socksStatusMap[task.Seq].tcp.dataChan,
		}
	} else {
		manager.socksStatusMap[task.Seq] = new(socksStatus)
		manager.socksStatusMap[task.Seq].tcp = new(tcpSocks)
		manager.socksStatusMap[task.Seq].tcp.dataChan = make(chan []byte, 5) // register it!
		manager.ResultChan <- &socksResult{
			SocksSeqExist: false,
			DataChan:      manager.socksStatusMap[task.Seq].tcp.dataChan,
		} // tell upstream result
	}
}

func (manager *socksManager) getUDPChans(task *SocksTask) {
	if _, ok := manager.socksStatusMap[task.Seq]; ok {
		manager.ResultChan <- &socksResult{
			OK:        true,
			DataChan:  manager.socksStatusMap[task.Seq].udp.dataChan,
			ReadyChan: manager.socksStatusMap[task.Seq].udp.readyChan,
		}
	} else {
		manager.ResultChan <- &socksResult{OK: false}
	}
}

func (manager *socksManager) updateTCP(task *SocksTask) {
	if _, ok := manager.socksStatusMap[task.Seq]; ok {
		manager.socksStatusMap[task.Seq].tcp.conn = task.SocksSocket
		manager.ResultChan <- &socksResult{OK: true}
	} else {
		manager.ResultChan <- &socksResult{OK: false} // avoid the scenario that admin conn ask to fin before "socks.buildConn()" call "updateTCP()"
	}
}

func (manager *socksManager) updateUDP(task *SocksTask) {
	if _, ok := manager.socksStatusMap[task.Seq]; ok {
		manager.socksStatusMap[task.Seq].isUDP = true
		manager.socksStatusMap[task.Seq].udp = new(udpSocks)
		manager.socksStatusMap[task.Seq].udp.dataChan = make(chan []byte)
		manager.socksStatusMap[task.Seq].udp.readyChan = make(chan string)
		manager.socksStatusMap[task.Seq].udp.headerPairs = make(map[string][]byte)
		manager.socksStatusMap[task.Seq].udp.listener = task.SocksListener
		manager.ResultChan <- &socksResult{OK: true} // tell upstream work done
	} else {
		manager.ResultChan <- &socksResult{OK: false}
	}
}

func (manager *socksManager) updateUDPHeader(task *SocksTask) {
	if _, ok := manager.socksStatusMap[task.Seq]; ok {
		manager.socksStatusMap[task.Seq].udp.headerPairs[task.SocksHeaderAddr] = task.SocksHeader
	}
	manager.ResultChan <- &socksResult{}
}

func (manager *socksManager) getUDPHeader(task *SocksTask) {
	if _, ok := manager.socksStatusMap[task.Seq]; ok {
		if _, ok := manager.socksStatusMap[task.Seq].udp.headerPairs[task.SocksHeaderAddr]; ok {
			manager.ResultChan <- &socksResult{
				OK:             true,
				SocksUDPHeader: manager.socksStatusMap[task.Seq].udp.headerPairs[task.SocksHeaderAddr],
			}
		} else {
			manager.ResultChan <- &socksResult{OK: false}
		}
	} else {
		manager.ResultChan <- &socksResult{OK: false}
	}
}

func (manager *socksManager) closeTCP(task *SocksTask) {
	if manager.socksStatusMap[task.Seq].tcp.conn != nil {
		manager.socksStatusMap[task.Seq].tcp.conn.Close() // avoid the scenario that admin conn ask to fin before "socks.buildConn()" call "updateTCP()"
	}

	close(manager.socksStatusMap[task.Seq].tcp.dataChan)

	if manager.socksStatusMap[task.Seq].isUDP {
		manager.socksStatusMap[task.Seq].udp.listener.Close()
		close(manager.socksStatusMap[task.Seq].udp.dataChan)
		close(manager.socksStatusMap[task.Seq].udp.readyChan)
		manager.socksStatusMap[task.Seq].udp.headerPairs = nil
	}

	delete(manager.socksStatusMap, task.Seq) // upstream not waiting
}

func (manager *socksManager) checkSocksReady() {
	if len(manager.socksStatusMap) == 0 {
		manager.ResultChan <- &socksResult{OK: true}
	} else {
		manager.ResultChan <- &socksResult{OK: false}
	}
}
