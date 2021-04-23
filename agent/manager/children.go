package manager

import (
	"net"
)

const (
	C_NEWCHILD = iota
	C_GETCONN
	C_GETALLCHILDREN
	C_DELCHILD
)

type childrenManager struct {
	children      map[string]*child
	ChildComeChan chan *ChildInfo

	TaskChan   chan *ChildrenTask
	ResultChan chan *ChildrenResult
}

type ChildrenTask struct {
	Mode int

	UUID string
	Conn net.Conn
}

type ChildrenResult struct {
	Conn     net.Conn
	OK       bool
	Children []string
}

type ChildInfo struct {
	UUID string
	Conn net.Conn
}

type child struct {
	conn net.Conn
}

func newChildrenManager() *childrenManager {
	manager := new(childrenManager)
	manager.children = make(map[string]*child)
	manager.ChildComeChan = make(chan *ChildInfo)
	manager.TaskChan = make(chan *ChildrenTask)
	manager.ResultChan = make(chan *ChildrenResult)
	return manager
}

func (manager *childrenManager) run() {
	for {
		task := <-manager.TaskChan

		switch task.Mode {
		case C_NEWCHILD:
			manager.newChild(task)
		case C_GETCONN:
			manager.getConn(task)
		case C_GETALLCHILDREN:
			manager.getAllChildren()
		case C_DELCHILD:
			manager.delChild(task)
		}
	}
}

func (manager *childrenManager) newChild(task *ChildrenTask) {
	manager.children[task.UUID] = new(child)
	manager.children[task.UUID].conn = task.Conn
	manager.ResultChan <- &ChildrenResult{OK: true}
}

func (manager *childrenManager) getConn(task *ChildrenTask) {
	if _, ok := manager.children[task.UUID]; ok {
		manager.ResultChan <- &ChildrenResult{
			OK:   true,
			Conn: manager.children[task.UUID].conn,
		}
	} else {
		manager.ResultChan <- &ChildrenResult{OK: false}
	}
}

func (manager *childrenManager) getAllChildren() {
	var children []string

	for child, _ := range manager.children {
		children = append(children, child)
	}

	manager.ResultChan <- &ChildrenResult{Children: children}
}

func (manager *childrenManager) delChild(task *ChildrenTask) {
	delete(manager.children, task.UUID)
	manager.ResultChan <- &ChildrenResult{OK: true}
}
