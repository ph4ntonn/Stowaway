/*
 * @Author: ph4ntom
 * @Date: 2021-03-11 14:59:13
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-03-20 11:54:54
 */
package cli

import (
	"container/list"
	"fmt"
)

const (
	BEGIN = iota
	PREV
	NEXT
)

type History struct {
	StoreList *list.List
	Capacity  int
	Record    chan string
	Search    chan int
	Display   chan interface{}
	Result    chan string
}

func NewHistory() *History {
	history := new(History)
	history.StoreList = list.New()
	history.Capacity = 100
	history.Record = make(chan string)
	history.Search = make(chan int)
	history.Display = make(chan interface{})
	history.Result = make(chan string, 1)
	return history
}

func (history *History) Run() {
	go history.record()
	go history.search()
	go history.display()
}

func (history *History) record() {
	for {
		command := <-history.Record
		history.StoreList.PushFront(command)
		if history.StoreList.Len() > history.Capacity*2 {
			history.clean()
		}
	}
}

func (history *History) search() {
	var now *list.Element

	for {
		switch <-history.Search {
		case BEGIN:
			if history.StoreList.Len() > 0 { // avoid list is empty
				now = history.StoreList.Front()
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
			history.Display <- now.Value
			history.Result <- now.Value.(string)
		}

		if history.StoreList.Len() == 0 { // avoid blocking the interactive panel if user press arrowup or arrowdown when no history node exists
			history.Display <- ""
			history.Result <- ""
		}
	}
}

func (history *History) display() {
	for {
		command := <-history.Display
		fCommand := command.(string)
		fmt.Print(fCommand)
	}
}

func (history *History) clean() {
	for elementsRemain := history.StoreList.Len() - history.Capacity; elementsRemain > 0; elementsRemain-- {
		element := history.StoreList.Back()
		history.StoreList.Remove(element)
	}
}
