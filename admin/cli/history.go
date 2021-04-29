package cli

import (
	"container/list"
)

const (
	// Mode
	RECORD = iota
	SEARCH
	// Type
	NORMAL
	SHELL
	SSH
	// Order
	BEGIN
	PREV
	NEXT
)

type History struct {
	normal *historyList
	shell  *historyList
	ssh    *historyList

	TaskChan   chan *HistoryTask
	ResultChan chan string
}

type historyList struct {
	storeList *list.List
	now       *list.Element
	capacity  int
}

type HistoryTask struct {
	Mode    int
	Type    int
	Order   int
	Command string
}

func NewHistory() *History {
	history := new(History)
	history.normal = new(historyList)
	history.normal.storeList = list.New()
	history.normal.capacity = 100

	history.shell = new(historyList)
	history.shell.storeList = list.New()
	history.shell.capacity = 100

	history.ssh = new(historyList)
	history.ssh.storeList = list.New()
	history.ssh.capacity = 100

	history.TaskChan = make(chan *HistoryTask)
	history.ResultChan = make(chan string)
	return history
}

func (history *History) Run() {
	for {
		task := <-history.TaskChan
		switch task.Mode {
		case RECORD:
			history.record(task)
		case SEARCH:
			history.search(task)
		}
	}
}

func (history *History) record(task *HistoryTask) {
	switch task.Type {
	case NORMAL:
		history.normal.storeList.PushFront(task.Command)
		if history.normal.storeList.Len() > history.normal.capacity*2 {
			history.clean(task)
		}
	case SHELL:
		history.shell.storeList.PushFront(task.Command)
		if history.shell.storeList.Len() > history.shell.capacity*2 {
			history.clean(task)
		}
	case SSH:
		history.ssh.storeList.PushFront(task.Command)
		if history.ssh.storeList.Len() > history.ssh.capacity*2 {
			history.clean(task)
		}
	}
}

func (history *History) search(task *HistoryTask) {
	switch task.Order {
	case BEGIN:
		switch task.Type {
		case NORMAL:
			if history.normal.storeList.Len() > 0 { // avoid list is empty
				history.normal.now = history.normal.storeList.Front()
			}
		case SHELL:
			if history.shell.storeList.Len() > 0 {
				history.shell.now = history.shell.storeList.Front()
			}
		case SSH:
			if history.ssh.storeList.Len() > 0 {
				history.ssh.now = history.ssh.storeList.Front()
			}
		}
	case PREV:
		switch task.Type {
		case NORMAL:
			if history.normal.now != nil && history.normal.now.Prev() != nil {
				history.normal.now = history.normal.now.Prev()
			}
		case SHELL:
			if history.shell.now != nil && history.shell.now.Prev() != nil {
				history.shell.now = history.shell.now.Prev()
			}
		case SSH:
			if history.ssh.now != nil && history.ssh.now.Prev() != nil {
				history.ssh.now = history.ssh.now.Prev()
			}
		}
	case NEXT:
		switch task.Type {
		case NORMAL:
			if history.normal.now != nil && history.normal.now.Next() != nil {
				history.normal.now = history.normal.now.Next()
			}
		case SHELL:
			if history.shell.now != nil && history.shell.now.Next() != nil {
				history.shell.now = history.shell.now.Next()
			}
		case SSH:
			if history.ssh.now != nil && history.ssh.now.Next() != nil {
				history.ssh.now = history.ssh.now.Next()
			}
		}
	}

	switch task.Type {
	case NORMAL:
		if history.normal.now != nil {
			command := history.normal.now.Value.(string)
			history.ResultChan <- command
		}
		if history.normal.storeList.Len() == 0 { // avoid blocking the interactive panel if user press arrowup or arrowdown when no history node exists
			history.ResultChan <- ""
		}
	case SHELL:
		if history.shell.now != nil {
			command := history.shell.now.Value.(string)
			history.ResultChan <- command
		}
		if history.shell.storeList.Len() == 0 {
			history.ResultChan <- ""
		}
	case SSH:
		if history.ssh.now != nil {
			command := history.ssh.now.Value.(string)
			history.ResultChan <- command
		}
		if history.ssh.storeList.Len() == 0 {
			history.ResultChan <- ""
		}
	}
}

func (history *History) clean(task *HistoryTask) {
	switch task.Type {
	case NORMAL:
		for elementsRemain := history.normal.storeList.Len() - history.normal.capacity; elementsRemain > 0; elementsRemain-- {
			element := history.normal.storeList.Back()
			history.normal.storeList.Remove(element)
		}
	case SHELL:
		for elementsRemain := history.shell.storeList.Len() - history.shell.capacity; elementsRemain > 0; elementsRemain-- {
			element := history.shell.storeList.Back()
			history.shell.storeList.Remove(element)
		}
	case SSH:
		for elementsRemain := history.ssh.storeList.Len() - history.ssh.capacity; elementsRemain > 0; elementsRemain-- {
			element := history.ssh.storeList.Back()
			history.ssh.storeList.Remove(element)
		}
	}
}
