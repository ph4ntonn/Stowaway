/*
 * @Author: ph4ntom
 * @Date: 2021-03-11 14:59:13
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-03-30 14:55:10
 */
package cli

import (
	"container/list"
)

const (
	// Mode
	RECORD = iota
	SEARCH
	// Order
	BEGIN
	PREV
	NEXT
)

type History struct {
	storeList *list.List
	now       *list.Element
	capacity  int

	TaskChan   chan *HistoryTask
	ResultChan chan string
}

type HistoryTask struct {
	Mode    int
	Order   int
	Command string
}

func NewHistory() *History {
	history := new(History)
	history.storeList = list.New()
	history.capacity = 100
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
	history.storeList.PushFront(task.Command)
	if history.storeList.Len() > history.capacity*2 {
		history.clean()
	}
}

func (history *History) search(task *HistoryTask) {
	switch task.Order {
	case BEGIN:
		if history.storeList.Len() > 0 { // avoid list is empty
			history.now = history.storeList.Front()
		}
	case PREV:
		if history.now != nil && history.now.Prev() != nil {
			history.now = history.now.Prev()
		}
	case NEXT:
		if history.now != nil && history.now.Next() != nil {
			history.now = history.now.Next()
		}
	}

	if history.now != nil {
		command := history.now.Value.(string)
		history.ResultChan <- command
	}

	if history.storeList.Len() == 0 { // avoid blocking the interactive panel if user press arrowup or arrowdown when no history node exists
		history.ResultChan <- ""
	}
}

func (history *History) clean() {
	for elementsRemain := history.storeList.Len() - history.capacity; elementsRemain > 0; elementsRemain-- {
		element := history.storeList.Back()
		history.storeList.Remove(element)
	}
}
