package common

import (
	"net"
	"sync"
)

/*-------------------------加锁map相关代码--------------------------*/
type Uint32StrMap struct {
	sync.RWMutex
	Payload map[uint32]chan string
}

type IntStrMap struct {
	sync.RWMutex
	Payload map[int]string
}

type Uint32ConnMap struct {
	sync.RWMutex
	Payload map[uint32]net.Conn
}

/*-------------------------初始化各类map相关代码--------------------------*/
func NewUint32StrMap() *Uint32StrMap {
	sm := new(Uint32StrMap)
	sm.Payload = make(map[uint32]chan string, 10)
	return sm
}

func NewIntStrMap() *IntStrMap {
	sm := new(IntStrMap)
	sm.Payload = make(map[int]string)
	return sm
}

func NewUint32ConnMap() *Uint32ConnMap {
	sm := new(Uint32ConnMap)
	sm.Payload = make(map[uint32]net.Conn)
	return sm
}
