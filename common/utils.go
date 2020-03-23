package common

import (
	"net"
	"runtime"
	"sync"
)

/*-------------------------加锁map相关代码--------------------------*/
type Uint32ChanStrMap struct {
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

type Uint32StrMap struct {
	sync.RWMutex
	Payload map[uint32]string
}

/*-------------------------初始化各类map相关代码--------------------------*/
func NewUint32ChanStrMap() *Uint32ChanStrMap {
	sm := new(Uint32ChanStrMap)
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

func NewUint32StrMap() *Uint32StrMap {
	sm := new(Uint32StrMap)
	sm.Payload = make(map[uint32]string)
	return sm
}

/*-------------------------chan状态判断相关代码--------------------------*/
//判断chan是否已经被释放
func IsClosed(ch chan string) bool {
	select {
	case <-ch:
		return true
	default:
	}
	return false
}

/*-------------------------操作系统判断相关代码--------------------------*/
func CheckSystem() (sysType uint32) {
	var os = runtime.GOOS
	switch os {
	case "windows":
		sysType = 0x01
	default:
		sysType = 0xff
	}
	return
}
