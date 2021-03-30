/*
 * @Author: ph4ntom
 * @Date: 2021-03-11 14:59:13
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-03-30 13:04:18
 */
package cli

import (
	"container/list"
	"fmt"
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
	var now *list.Element

	switch task.Order {
	case BEGIN:
		if history.storeList.Len() > 0 { // avoid list is empty
			now = history.storeList.Front()
		}
	case PREV:
		if now != nil && now.Prev() != nil {
			now = now.Prev()
		}
	case NEXT:
		if now != nil && now.Next() != nil {
			now = now.Next()
		}
	}

	if now != nil {
		command := now.Value.(string)
		history.display(command)
		history.ResultChan <- command
	}

	if history.storeList.Len() == 0 { // avoid blocking the interactive panel if user press arrowup or arrowdown when no history node exists
		history.display("")
		history.ResultChan <- ""
	}
}

func (history *History) display(command string) {
	fmt.Print(command)
}

func (history *History) clean() {
	for elementsRemain := history.storeList.Len() - history.capacity; elementsRemain > 0; elementsRemain-- {
		element := history.storeList.Back()
		history.storeList.Remove(element)
	}
}
