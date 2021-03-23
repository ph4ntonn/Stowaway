/*
 * @Author: ph4ntom
 * @Date: 2021-03-23 19:01:26
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-03-23 19:16:49
 */
package share

import (
	"net"
)

const (
	SOCKS = iota
	NEWSOCKS
	ADDSOCKSSOCKET
	GETSOCKSID
)

type Manager struct {
	File *MyFile

	socks5ID uint64
	socks5   map[int]*socks

	TaskChan        chan *ManagerTask
	socksTaskChan   chan *ManagerTask
	SocksResultChan chan *ManagerResult
}

type ManagerTask struct {
	Category int
	Mode     int
	UUIDNum  int
	//socks
	SocksPort     int
	SocksUsername string
	SocksPassword string
	SocksSocket   net.Conn
}

type ManagerResult struct {
	OK bool
	//socks
	SocksID uint64
}

type socks struct {
	Port     int
	Username string
	Password string
	Conn     []net.Conn
}

func NewManager(file *MyFile) *Manager {
	manager := new(Manager)
	manager.File = file
	manager.socks5 = make(map[int]*socks)
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
		case NEWSOCKS:
			manager.newSocks(task)
		case ADDSOCKSSOCKET:
			manager.addSocksSocket(task)
		case GETSOCKSID:
			manager.getSocksID()
		}
	}
}

func (manager *Manager) ifSocksExist(uuidNum int) bool {
	if manager.socks5[uuidNum].Port != 0 {
		return true
	}
	return false
}

func (manager *Manager) newSocks(task *ManagerTask) {
	if !manager.ifSocksExist(task.UUIDNum) {
		manager.socks5[task.UUIDNum].Port = task.SocksPort
		manager.socks5[task.UUIDNum].Username = task.SocksUsername
		manager.socks5[task.UUIDNum].Password = task.SocksPassword
		manager.SocksResultChan <- &ManagerResult{OK: true}
	} else {
		manager.SocksResultChan <- &ManagerResult{OK: false}
	}
}

func (manager *Manager) addSocksSocket(task *ManagerTask) {
	manager.socks5[task.UUIDNum].Conn = append(manager.socks5[task.UUIDNum].Conn, task.SocksSocket)
	manager.SocksResultChan <- &ManagerResult{}
}

func (manager *Manager) getSocksID() {
	manager.SocksResultChan <- &ManagerResult{SocksID: manager.socks5ID}
	manager.socks5ID++
}
