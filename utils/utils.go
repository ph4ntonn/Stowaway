package utils

import (
	"crypto/md5"
	"encoding/hex"
	"net"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/cheggaaa/pb/v3"
)

var AdminId = "0000000000"
var StartNodeId = "0000000001"

/*-------------------------Admin命令参数结构代码--------------------------*/
type AdminOptions struct {
	Secret     string
	Listen     string
	Connect    string
	Rhostreuse bool
}

/*-------------------------Agent命令参数结构代码--------------------------*/
type AgentOptions struct {
	Secret      string
	Listen      string
	Reconnect   string
	Reverse     bool
	Monitor     string
	IsStartNode bool
	ReuseHost   string
	ReusePort   string
	RhostReuse  bool
}

/*-------------------------Admin相关状态变量代码--------------------------*/
type AdminStatus struct {
	ReadyChange      chan bool
	IsShellMode      chan bool
	SSHSuccess       chan bool
	NodeSocksStarted chan bool
	GetName          chan bool
	CannotRead       chan bool
	NodesReadyToadd  chan map[string]string //等待加入的node
	HandleNode       string                 //正在操作的节点编号
	AESKey           []byte
}

func NewAdminStatus() *AdminStatus {
	nas := new(AdminStatus)
	nas.ReadyChange = make(chan bool, 1)
	nas.IsShellMode = make(chan bool, 1)
	nas.SSHSuccess = make(chan bool, 1)
	nas.NodeSocksStarted = make(chan bool, 1)
	nas.GetName = make(chan bool, 1)
	nas.CannotRead = make(chan bool, 1)
	nas.NodesReadyToadd = make(chan map[string]string)
	nas.HandleNode = AdminId
	return nas
}

/*-------------------------Admin零散变量代码--------------------------*/
type AdminStuff struct {
	sync.RWMutex
	StartNode              string
	AdminCommandChan       chan []string
	SocksNum               uint32
	ReflectNum             uint32
	SocksListenerForClient *StrListenerSliceMap
	SocksMapping           *StrUint32SliceMap
}

func NewAdminStuff() *AdminStuff {
	nas := new(AdminStuff)
	nas.StartNode = "0.0.0.0"
	nas.SocksNum = 0
	nas.ReflectNum = 0
	nas.AdminCommandChan = make(chan []string, 1)
	nas.SocksListenerForClient = NewStrListenerSliceMap()
	nas.SocksMapping = NewStrUint32SliceMap()
	return nas
}

/*-------------------------Agent相关状态变量代码--------------------------*/
type AgentStatus struct {
	Nodeid            string
	NodeNote          string
	NotLastOne        bool
	Waiting           bool
	ReConnCome        chan bool
	WaitForIDAllocate chan string
	AESKey            []byte
}

func NewAgentStatus() *AgentStatus {
	nas := new(AgentStatus)
	nas.Nodeid = StartNodeId
	nas.NodeNote = ""
	nas.NotLastOne = false
	nas.Waiting = false
	nas.ReConnCome = make(chan bool, 1)
	nas.WaitForIDAllocate = make(chan string, 1)
	return nas
}

/*-------------------------Node状态代码--------------------------*/
type NodeStatus struct {
	NodeIP       map[string]string
	Nodenote     map[string]string
	NodeHostname map[string]string
	NodeUser     map[string]string
}

func NewNodeStatus() *NodeStatus {
	nns := new(NodeStatus)
	nns.NodeIP = make(map[string]string)
	nns.Nodenote = make(map[string]string)
	nns.NodeHostname = make(map[string]string)
	nns.NodeUser = make(map[string]string)
	return nns
}

/*-------------------------Node信息代码--------------------------*/
type NodeInfo struct {
	UpperNode string
	LowerNode *StrConnMap
}

func NewNodeInfo() *NodeInfo {
	nni := new(NodeInfo)
	nni.UpperNode = AdminId
	nni.LowerNode = NewStrConnMap()
	return nni
}

/*-------------------------Node上下级信息代码--------------------------*/
type NodeStuff struct {
	ControlConnForLowerNodeChan chan net.Conn //下级节点控制信道
	Adminconn                   chan net.Conn
	ReOnlineConn                chan net.Conn
	NewNodeMessageChan          chan []byte //新节点加入消息
	IsAdmin                     chan bool   //分辨连接是属于admin还是agent
	PrepareForReOnlineNodeReady chan bool
	ReOnlineID                  chan string
	Offline                     bool //判断当前状态是否是掉线状态
}

func NewNodeStuff() *NodeStuff {
	nns := new(NodeStuff)
	nns.ControlConnForLowerNodeChan = make(chan net.Conn, 1)
	nns.Adminconn = make(chan net.Conn, 1)
	nns.ReOnlineConn = make(chan net.Conn, 1)
	nns.NewNodeMessageChan = make(chan []byte, 1)
	nns.IsAdmin = make(chan bool, 1)
	nns.PrepareForReOnlineNodeReady = make(chan bool, 1)
	nns.ReOnlineID = make(chan string, 1)
	nns.Offline = false
	return nns
}

/*-------------------------Node上下级信息代码--------------------------*/
type Node struct {
	Uppernode string
	Lowernode []string
}

func NewNode() *Node {
	nn := new(Node)
	nn.Lowernode = make([]string, 0)
	return nn
}

/*-------------------------传递给下级节点结构代码--------------------------*/
type PassToLowerNodeData struct {
	Route string
	Data  []byte
}

func NewPassToLowerNodeData() *PassToLowerNodeData {
	nptlnd := new(PassToLowerNodeData)
	return nptlnd
}

/*-------------------------Forward配置相关代码--------------------------*/
type ForwardStatus struct {
	sync.RWMutex
	ForwardIsValid             chan bool
	ForwardNum                 uint32
	CurrentPortForwardListener *StrListenerSliceMap
	ForwardMapping             *StrUint32SliceMap
}

func NewForwardStatus() *ForwardStatus {
	nfs := new(ForwardStatus)
	nfs.ForwardNum = 0
	nfs.ForwardIsValid = make(chan bool, 1)
	nfs.CurrentPortForwardListener = NewStrListenerSliceMap()
	nfs.ForwardMapping = NewStrUint32SliceMap()
	return nfs
}

/*-------------------------Reflect配置相关代码--------------------------*/
type ReflectStatus struct {
	ReflectNum chan uint32
}

func NewReflectStatus() *ReflectStatus {
	nrs := new(ReflectStatus)
	nrs.ReflectNum = make(chan uint32)
	return nrs
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
	ProxyChanToLowerNode chan *PassToLowerNodeData
	ProxyChanToUpperNode chan []byte
}

func NewProxyChan() *ProxyChan {
	npc := new(ProxyChan)
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

type StrConnMap struct {
	sync.RWMutex
	Payload map[string]net.Conn
}

type Uint32ConnMap struct {
	sync.RWMutex
	Payload map[uint32]net.Conn
}

type Uint32StrMap struct {
	sync.RWMutex
	Payload map[uint32]string
}

type SafeNodeMap struct {
	sync.RWMutex
	AllNode map[string]*Node
}

type SafeRouteMap struct {
	sync.RWMutex
	Route map[string]string
}

/*-------------------------不加锁map相关代码--------------------------*/
type StrListenerSliceMap struct {
	Payload map[string][]net.Listener
}

type StrUint32SliceMap struct {
	Payload map[string][]uint32
}

/*-------------------------初始化各类map相关代码--------------------------*/
func NewUint32ChanStrMap() *Uint32ChanStrMap {
	sm := new(Uint32ChanStrMap)
	sm.Payload = make(map[uint32]chan string, 10)
	return sm
}

func NewStrConnMap() *StrConnMap {
	nscm := new(StrConnMap)
	nscm.Payload = make(map[string]net.Conn)
	return nscm
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

func NewStrListenerSliceMap() *StrListenerSliceMap {
	nilsm := new(StrListenerSliceMap)
	nilsm.Payload = make(map[string][]net.Listener)
	return nilsm
}

func NewStrUint32SliceMap() *StrUint32SliceMap {
	nuusm := new(StrUint32SliceMap)
	nuusm.Payload = make(map[string][]uint32)
	return nuusm
}

func NewSafeNodeMap() *SafeNodeMap {
	nsnm := new(SafeNodeMap)
	nsnm.AllNode = make(map[string]*Node)
	return nsnm
}

func NewSafeRouteMap() *SafeRouteMap {
	nsrm := new(SafeRouteMap)
	nsrm.Route = make(map[string]string)
	return nsrm
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

/*-------------------------根据操作系统返回系统信息相关代码--------------------------*/
func GetInfoViaSystem() string {
	var os = runtime.GOOS
	switch os {
	case "windows":
		fallthrough
	case "linux":
		fallthrough
	case "darwin":
		temHostname, err := exec.Command("hostname").Output()
		if err != nil {
			temHostname = []byte("Null")
		}
		temUsername, err := exec.Command("whoami").Output()
		if err != nil {
			temUsername = []byte("Null")
		}

		hostname := strings.Replace(string(temHostname), "\n", "", -1)
		username := strings.Replace(string(temUsername), "\n", "", -1)
		hostname = strings.Replace(hostname, "\r", "", -1)
		username = strings.Replace(username, "\r", "", -1)

		return hostname + ":::stowaway:::" + username
	default:
		return "Null" + ":::stowaway:::" + "Null"
	}
}

/*-------------------------进度条生成相关代码--------------------------*/
func NewBar(length int64) *pb.ProgressBar {
	bar := pb.New64(int64(length))
	bar.SetTemplate(pb.Full)
	bar.Set(pb.Bytes, true)
	return bar
}

/*-------------------------操作功能性代码--------------------------*/
//uint32转换至string类型
func Uint32Str(num uint32) string {
	b := strconv.Itoa(int(num))
	return b
}

//string转换至uint32
func StrUint32(str string) uint32 {
	num, _ := strconv.ParseInt(str, 10, 32)
	return uint32(num)
}

//排序
func CheckRange(nodes []int) {
	for m := len(nodes) - 1; m > 0; m-- {
		var flag bool = false
		for n := 0; n < m; n++ {
			if nodes[n] > nodes[n+1] {
				temp := nodes[n]
				nodes[n] = nodes[n+1]
				nodes[n+1] = temp
				flag = true
			}
		}
		if !flag {
			break
		}
	}
}

//倒置[]string
func StringReverse(src []string) {
	if src == nil {
		return
	}
	count := len(src)
	mid := count / 2
	for i := 0; i < mid; i++ {
		tmp := src[i]
		src[i] = src[count-1]
		src[count-1] = tmp
		count--
	}
}

//获取slice中的特定值
func FindSpecFromSlice(nodeid string, nodes []string) int {
	for key, value := range nodes {
		if nodeid == value {
			return key
		}
	}
	return -1
}

//生成md5值
func GetStringMd5(s string) string {
	md5 := md5.New()
	md5.Write([]byte(s))
	md5Str := hex.EncodeToString(md5.Sum(nil))
	return md5Str
}
