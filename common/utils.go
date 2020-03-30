package common

import (
	"net"
	"runtime"
	"sync"
)

/*-------------------------Admin相关状态变量代码--------------------------*/
type AdminStatus struct {
	ReadyChange      chan bool
	IsShellMode      chan bool
	SshSuccess       chan bool
	NodeSocksStarted chan bool
	GetName          chan bool
	CannotRead       chan bool
	NodesReadyToadd  chan map[uint32]string
	AESKey           []byte
}

func NewAdminStatus() *AdminStatus {
	nas := new(AdminStatus)
	nas.ReadyChange = make(chan bool, 1)
	nas.IsShellMode = make(chan bool, 1)
	nas.SshSuccess = make(chan bool, 1)
	nas.NodeSocksStarted = make(chan bool, 1)
	nas.GetName = make(chan bool, 1)
	nas.CannotRead = make(chan bool, 1)
	nas.NodesReadyToadd = make(chan map[uint32]string)
	return nas
}

/*-------------------------Admin零散变量代码--------------------------*/
type AdminStuff struct {
	StartNode              string
	AdminCommandChan       chan []string
	SocksNum               uint32
	SocksListenerForClient []net.Listener
}

func NewAdminStuff() *AdminStuff {
	nas := new(AdminStuff)
	nas.StartNode = "0.0.0.0"
	nas.SocksNum = 0
	nas.AdminCommandChan = make(chan []string)
	return nas
}

/*-------------------------Agent相关状态变量代码--------------------------*/
type AgentStatus struct {
	NODEID     uint32
	NotLastOne bool
	Waiting    bool
	ReConnCome chan bool
	AESKey     []byte
}

func NewAgentStatus() *AgentStatus {
	nas := new(AgentStatus)
	nas.NODEID = uint32(1)
	nas.NotLastOne = false
	nas.Waiting = false
	nas.ReConnCome = make(chan bool, 1)
	return nas
}

/*-------------------------Node信息代码--------------------------*/
type NodeStatus struct {
	Nodes    map[uint32]string
	Nodenote map[uint32]string
}

func NewNodeStatus() *NodeStatus {
	nns := new(NodeStatus)
	nns.Nodes = make(map[uint32]string)
	nns.Nodenote = make(map[uint32]string)
	return nns
}

/*-------------------------Forward配置相关代码--------------------------*/
type ForwardStatus struct {
	ForwardIsValid             chan bool
	ForwardNum                 uint32
	CurrentPortForwardListener []net.Listener
}

func NewForwardStatus() *ForwardStatus {
	nfs := new(ForwardStatus)
	nfs.ForwardNum = 0
	nfs.ForwardIsValid = make(chan bool, 1)
	return nfs
}

/*-------------------------Socks5配置相关代码--------------------------*/
type SocksSetting struct {
	SocksUsername string
	SocksPass     string
}

func NewSocksSetting() *SocksSetting {
	nss := new(SocksSetting)
	return nss
}

/*-------------------------File upload/download配置相关代码--------------------------*/
type FileStatus struct {
	TotalSilceNum       int
	FileSize            int64
	TotalConfirm        chan bool
	ReceiveFileSize     chan bool
	ReceiveFileSliceNum chan bool
}

func NewFileStatus() *FileStatus {
	nfs := new(FileStatus)
	nfs.TotalConfirm = make(chan bool, 1)
	nfs.ReceiveFileSliceNum = make(chan bool, 1)
	nfs.ReceiveFileSize = make(chan bool, 1)
	return nfs
}

/*-------------------------ProxyChan相关代码--------------------------*/
type ProxyChan struct {
	ProxyChanToLowerNode chan []byte
	ProxyChanToUpperNode chan []byte
}

func NewProxyChan() *ProxyChan {
	npc := new(ProxyChan)
	npc.ProxyChanToLowerNode = make(chan []byte, 1)
	npc.ProxyChanToUpperNode = make(chan []byte, 1)
	return npc
}

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
