/*
 * @Author: ph4ntom
 * @Date: 2021-04-02 16:01:58
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-04-02 18:46:53
 */
package manager

import (
	"fmt"
	"net"
)

const (
	F_GETNEWSEQ = iota
	F_NEWFORWARD
	F_ADDCONN
	F_GETDATACHAN
	F_GETDATACHAN_WITHOUTUUID
	F_GETFORWARDINFO
	F_CLOSETCP
	F_CLOSESINGLE
	F_CLOSESINGLEALL
	F_FORCESHUTDOWN
)

type forwardManager struct {
	forwardSeq      uint64
	forwardSeqMap   map[uint64]*fwSeqRelationship  // map[seq](port+uuid) just for accelerate the speed of searching detail only by seq
	forwardMap      map[string]map[string]*forward // map[uuid]map[port]*forward's detail record forward status
	forwardReadyDel map[int]string                 // map[user's option]port(no need to initial it in newForwardManager())

	ForwardMessChan chan interface{}
	ForwardReady    chan bool

	TaskChan   chan *ForwardTask
	ResultChan chan *forwardResult
	Done       chan bool
}

type ForwardTask struct {
	Mode int
	UUID string // node uuid
	Seq  uint64 // seq

	Port        string
	RemoteAddr  string
	CloseTarget int
	Listener    net.Listener
}

type forwardResult struct {
	OK bool

	ForwardSeq  uint64
	DataChan    chan []byte
	ForwardInfo []string
}

type forward struct {
	remoteAddr string
	listener   net.Listener

	forwardStatusMap map[uint64]*forwardStatus
}

type forwardStatus struct {
	dataChan chan []byte
}

type fwSeqRelationship struct {
	uuid string
	port string
}

func newForwardManager() *forwardManager {
	manager := new(forwardManager)

	manager.forwardMap = make(map[string]map[string]*forward)
	manager.forwardSeqMap = make(map[uint64]*fwSeqRelationship)
	manager.ForwardMessChan = make(chan interface{}, 5)
	manager.ForwardReady = make(chan bool)

	manager.TaskChan = make(chan *ForwardTask)
	manager.ResultChan = make(chan *forwardResult)
	manager.Done = make(chan bool)

	return manager
}

func (manager *forwardManager) run() {
	for {
		task := <-manager.TaskChan

		switch task.Mode {
		case F_NEWFORWARD:
			manager.newForward(task)
		case F_GETNEWSEQ:
			manager.getNewSeq(task)
		case F_ADDCONN:
			manager.addConn(task)
		case F_GETDATACHAN:
			manager.getDatachan(task)
		case F_GETDATACHAN_WITHOUTUUID:
			manager.getDatachanWithoutUUID(task)
			<-manager.Done
		case F_GETFORWARDINFO:
			manager.getForwardInfo(task)
		case F_CLOSETCP:
			manager.closeTCP(task)
		case F_CLOSESINGLE:
			manager.closeSingle(task)
		case F_CLOSESINGLEALL:
			manager.closeSingleAll(task)
		case F_FORCESHUTDOWN:
			manager.forceShutdown(task)
		}
	}
}

// 2022.7.19 Fix nil pointer bug,thx to @zyylhn
func (manager *forwardManager) newForward(task *ForwardTask) {
	if _, ok := manager.forwardMap[task.UUID]; !ok {
		manager.forwardMap[task.UUID] = make(map[string]*forward)
	}

	manager.forwardMap[task.UUID][task.Port] = new(forward)
	manager.forwardMap[task.UUID][task.Port].listener = task.Listener
	manager.forwardMap[task.UUID][task.Port].remoteAddr = task.RemoteAddr
	manager.forwardMap[task.UUID][task.Port].forwardStatusMap = make(map[uint64]*forwardStatus)

	manager.ResultChan <- &forwardResult{OK: true}
}

func (manager *forwardManager) getNewSeq(task *ForwardTask) {
	manager.forwardSeqMap[manager.forwardSeq] = &fwSeqRelationship{uuid: task.UUID, port: task.Port}
	manager.ResultChan <- &forwardResult{ForwardSeq: manager.forwardSeq}
	manager.forwardSeq++
}

func (manager *forwardManager) addConn(task *ForwardTask) {
	if _, ok := manager.forwardSeqMap[task.Seq]; !ok {
		manager.ResultChan <- &forwardResult{OK: false}
		return
	}

	manager.forwardMap[task.UUID][task.Port].forwardStatusMap[task.Seq] = new(forwardStatus)
	manager.forwardMap[task.UUID][task.Port].forwardStatusMap[task.Seq].dataChan = make(chan []byte, 5)
	manager.ResultChan <- &forwardResult{OK: true}
}

func (manager *forwardManager) getDatachan(task *ForwardTask) {
	if _, ok := manager.forwardSeqMap[task.Seq]; !ok {
		manager.ResultChan <- &forwardResult{OK: false}
		return
	}

	if _, ok := manager.forwardMap[task.UUID][task.Port].forwardStatusMap[task.Seq]; ok { // need to check ,because you will never know when fin come
		manager.ResultChan <- &forwardResult{
			OK:       true,
			DataChan: manager.forwardMap[task.UUID][task.Port].forwardStatusMap[task.Seq].dataChan,
		}
	} else {
		manager.ResultChan <- &forwardResult{OK: false}
	}
}

func (manager *forwardManager) getDatachanWithoutUUID(task *ForwardTask) {
	if _, ok := manager.forwardSeqMap[task.Seq]; !ok {
		manager.ResultChan <- &forwardResult{OK: false}
		return
	}

	uuid := manager.forwardSeqMap[task.Seq].uuid
	port := manager.forwardSeqMap[task.Seq].port

	manager.ResultChan <- &forwardResult{ // no need to chek forwardStatusMap[task.Seq] like above,because no more data after fin
		OK:       true,
		DataChan: manager.forwardMap[uuid][port].forwardStatusMap[task.Seq].dataChan,
	}
}

func (manager *forwardManager) getForwardInfo(task *ForwardTask) {
	manager.forwardReadyDel = make(map[int]string)

	var forwardInfo []string
	infoNum := 1

	if _, ok := manager.forwardMap[task.UUID]; ok {
		forwardInfo = append(forwardInfo, "\r\n[0] All")
		for port, info := range manager.forwardMap[task.UUID] {
			manager.forwardReadyDel[infoNum] = port
			detail := fmt.Sprintf("\r\n[%d] Listening Addr : %s , Remote Addr : %s , Current Active Connnections : %d", infoNum, info.listener.Addr().String(), info.remoteAddr, len(info.forwardStatusMap))
			forwardInfo = append(forwardInfo, detail)
			infoNum++
		}
		manager.ResultChan <- &forwardResult{
			OK:          true,
			ForwardInfo: forwardInfo,
		}
	} else {
		forwardInfo = append(forwardInfo, "\r\nForward service isn't running!")
		manager.ResultChan <- &forwardResult{
			OK:          false,
			ForwardInfo: forwardInfo,
		}
	}
}

func (manager *forwardManager) closeTCP(task *ForwardTask) {
	if _, ok := manager.forwardSeqMap[task.Seq]; !ok {
		return
	}

	uuid := manager.forwardSeqMap[task.Seq].uuid
	port := manager.forwardSeqMap[task.Seq].port

	close(manager.forwardMap[uuid][port].forwardStatusMap[task.Seq].dataChan)

	delete(manager.forwardMap[uuid][port].forwardStatusMap, task.Seq)
}

func (manager *forwardManager) closeSingle(task *ForwardTask) {
	// find port that user want to del
	port := manager.forwardReadyDel[task.CloseTarget]
	// close corresponding listener
	manager.forwardMap[task.UUID][port].listener.Close()
	// clear every single connection's resources
	for seq, status := range manager.forwardMap[task.UUID][port].forwardStatusMap {
		close(status.dataChan)
		delete(manager.forwardMap[task.UUID][port].forwardStatusMap, seq)
	}
	// delete the target port
	delete(manager.forwardMap[task.UUID], port)
	// clear the seqmap that match relationship.uuid == task.UUID && relationship.port == port
	for seq, relationship := range manager.forwardSeqMap {
		if relationship.uuid == task.UUID && relationship.port == port {
			delete(manager.forwardSeqMap, seq)
		}
	}
	// if no other forward services running on current node,delete node from manager.forwardMap
	if len(manager.forwardMap[task.UUID]) == 0 {
		delete(manager.forwardMap, task.UUID)
	}

	manager.ResultChan <- &forwardResult{OK: true}
}

func (manager *forwardManager) closeSingleAll(task *ForwardTask) {
	for port, forward := range manager.forwardMap[task.UUID] {
		forward.listener.Close()

		for seq, status := range forward.forwardStatusMap {
			close(status.dataChan)
			delete(forward.forwardStatusMap, seq)
		}

		delete(manager.forwardMap[task.UUID], port)
	}

	for seq, relationship := range manager.forwardSeqMap {
		if relationship.uuid == task.UUID {
			delete(manager.forwardSeqMap, seq)
		}
	}

	delete(manager.forwardMap, task.UUID)

	manager.ResultChan <- &forwardResult{OK: true}
}

func (manager *forwardManager) forceShutdown(task *ForwardTask) {
	if _, ok := manager.forwardMap[task.UUID]; ok {
		manager.closeSingleAll(task)
	} else {
		manager.ResultChan <- &forwardResult{OK: true}
	}
}
