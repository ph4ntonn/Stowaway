package protocol

import (
	"net"

	"Stowaway/crypto"
)

var Upstream string
var Downstream string

const (
	HI = iota
	UUID
	CHILDUUIDREQ
	CHILDUUIDRES
	MYINFO
	MYMEMO
	SHELLREQ
	SHELLRES
	SHELLCOMMAND
	SHELLRESULT
	SHELLEXIT
	LISTENREQ
	LISTENRES
	SSHREQ
	SSHRES
	SSHCOMMAND
	SSHRESULT
	SSHEXIT
	SSHTUNNELREQ
	SSHTUNNELRES
	FILESTATREQ
	FILESTATRES
	FILEDATA
	FILEERR
	FILEDOWNREQ
	FILEDOWNRES
	SOCKSSTART
	SOCKSTCPDATA
	SOCKSUDPDATA
	UDPASSSTART
	UDPASSRES
	SOCKSTCPFIN
	SOCKSREADY
	FORWARDTEST
	FORWARDSTART
	FORWARDREADY
	FORWARDDATA
	FORWARDFIN
	BACKWARDTEST
	BACKWARDSTART
	BACKWARDSEQ
	BACKWARDREADY
	BACKWARDDATA
	BACKWARDFIN
	BACKWARDSTOP
	BACKWARDSTOPDONE
	CONNECTSTART
	CONNECTDONE
	NODEOFFLINE
	NODEREONLINE
	UPSTREAMOFFLINE
	UPSTREAMREONLINE
	SHUTDOWN
	HEARTBEAT
)

const ADMIN_UUID = "IAMADMINXD"
const TEMP_UUID = "IAMNEWHERE"
const TEMP_ROUTE = "THEREISNOROUTE"

type Proto interface {
	CNegotiate() error
	SNegotiate() error
}

type NegParam struct {
	Domain string
	Conn   net.Conn
}

type Message interface {
	ConstructHeader()
	ConstructData(*Header, interface{}, bool)
	ConstructSuffix()
	DeconstructHeader()
	DeconstructData() (*Header, interface{}, error)
	DeconstructSuffix()
	SendMessage()
}

func ConstructMessage(message Message, header *Header, mess interface{}, isPass bool) {
	message.ConstructData(header, mess, isPass)
	message.ConstructHeader()
	message.ConstructSuffix()
}

func DestructMessage(message Message) (*Header, interface{}, error) {
	message.DeconstructHeader()
	header, mess, err := message.DeconstructData()
	message.DeconstructSuffix()
	return header, mess, err
}

type Header struct {
	Sender      string // sender and accepter are both 10bytes
	Accepter    string
	MessageType uint16
	RouteLen    uint32
	Route       string
	DataLen     uint64
}

type HIMess struct {
	GreetingLen uint16
	Greeting    string
	UUIDLen     uint16
	UUID        string
	IsAdmin     uint16
	IsReconnect uint16
}

type UUIDMess struct {
	UUIDLen uint16
	UUID    string
}

type ChildUUIDReq struct {
	ParentUUIDLen uint16
	ParentUUID    string
	IPLen         uint16
	IP            string
}

type ChildUUIDRes struct {
	UUIDLen uint16
	UUID    string
}

type MyInfo struct {
	UUIDLen     uint16
	UUID        string
	UsernameLen uint64
	Username    string
	HostnameLen uint64
	Hostname    string
	MemoLen     uint64
	Memo        string
}

type MyMemo struct {
	MemoLen uint64
	Memo    string
}

type ShellReq struct {
	Start uint16
}

type ShellRes struct {
	OK uint16
}

type ShellCommand struct {
	CommandLen uint64
	Command    string
}

type ShellResult struct {
	ResultLen uint64
	Result    string
}

type ShellExit struct {
	OK uint16
}

type ListenReq struct {
	Method  uint16
	AddrLen uint64
	Addr    string
}

type ListenRes struct {
	OK uint16
}

type SSHReq struct {
	Method         uint16
	AddrLen        uint16
	Addr           string
	UsernameLen    uint64
	Username       string
	PasswordLen    uint64
	Password       string
	CertificateLen uint64
	Certificate    []byte
}

type SSHRes struct {
	OK uint16
}

type SSHCommand struct {
	CommandLen uint64
	Command    string
}

type SSHResult struct {
	ResultLen uint64
	Result    string
}

type SSHExit struct {
	OK uint16
}

type SSHTunnelReq struct {
	Method         uint16
	AddrLen        uint16
	Addr           string
	PortLen        uint16
	Port           string
	UsernameLen    uint64
	Username       string
	PasswordLen    uint64
	Password       string
	CertificateLen uint64
	Certificate    []byte
}

type SSHTunnelRes struct {
	OK uint16
}

type FileStatReq struct {
	FilenameLen uint32
	Filename    string
	FileSize    uint64
	SliceNum    uint64
}

type FileStatRes struct {
	OK uint16
}

type FileData struct {
	DataLen uint64
	Data    []byte
}

type FileErr struct {
	Error uint16
}

type FileDownReq struct {
	FilePathLen uint32
	FilePath    string
	FilenameLen uint32
	Filename    string
}

type FileDownRes struct {
	OK uint16
}

type SocksStart struct {
	UsernameLen uint64
	Username    string
	PasswordLen uint64
	Password    string
}

type SocksTCPData struct {
	Seq     uint64
	DataLen uint64
	Data    []byte
}

type SocksUDPData struct {
	Seq     uint64
	DataLen uint64
	Data    []byte
}

type UDPAssStart struct {
	Seq           uint64
	SourceAddrLen uint16
	SourceAddr    string
}

type UDPAssRes struct {
	Seq     uint64
	OK      uint16
	AddrLen uint16
	Addr    string
}

type SocksTCPFin struct {
	Seq uint64
}

type SocksReady struct {
	OK uint16
}

type ForwardTest struct {
	AddrLen uint16
	Addr    string
}

type ForwardStart struct {
	Seq     uint64
	AddrLen uint16
	Addr    string
}

type ForwardReady struct {
	OK uint16
}

type ForwardData struct {
	Seq     uint64
	DataLen uint64
	Data    []byte
}

type ForwardFin struct {
	Seq uint64
}

type BackwardTest struct {
	LPortLen uint16
	LPort    string
	RPortLen uint16
	RPort    string
}

type BackwardStart struct {
	UUIDLen  uint16
	UUID     string
	LPortLen uint16
	LPort    string
	RPortLen uint16
	RPort    string
}

type BackwardReady struct {
	OK uint16
}

type BackwardSeq struct {
	Seq      uint64
	RPortLen uint16
	RPort    string
}

type BackwardData struct {
	Seq     uint64
	DataLen uint64
	Data    []byte
}

type BackWardFin struct {
	Seq uint64
}

type BackwardStop struct {
	All      uint16
	RPortLen uint16
	RPort    string
}

type BackwardStopDone struct {
	All      uint16
	UUIDLen  uint16
	UUID     string
	RPortLen uint16
	RPort    string
}

type ConnectStart struct {
	AddrLen uint16
	Addr    string
}

type ConnectDone struct {
	OK uint16
}

type NodeOffline struct {
	UUIDLen uint16
	UUID    string
}

type NodeReonline struct {
	ParentUUIDLen uint16
	ParentUUID    string
	UUIDLen       uint16
	UUID          string
	IPLen         uint16
	IP            string
}

type UpstreamOffline struct {
	OK uint16
}

type UpstreamReonline struct {
	OK uint16
}

type Shutdown struct {
	OK uint16
}

type HeartbeatMsg struct {
	Ping uint16
}

type MessageComponent struct {
	UUID   string
	Conn   net.Conn
	Secret string
}

func SetUpDownStream(upstream, downstream string) {
	if upstream == "ws" {
		Upstream = "ws"
	} else if upstream == "http" {
		Upstream = "http"
	} else {
		Upstream = "raw"
	}

	if downstream == "ws" {
		Downstream = "ws"
	} else if downstream == "http" {
		Downstream = "http"
	} else {
		Downstream = "raw"
	}
}

func NewUpProto(param *NegParam) Proto {
	switch Upstream {
	case "raw":
		tProto := new(RawProto)
		return tProto
	case "http":
		tProto := new(HTTPProto)
		return tProto
	case "ws":
		tProto := new(WSProto)
		tProto.domain = param.Domain
		tProto.conn = param.Conn
		return tProto
	}
	return nil
}

func NewDownProto(param *NegParam) Proto {
	switch Downstream {
	case "raw":
		tProto := new(RawProto)
		return tProto
	case "http":
		tProto := new(HTTPProto)
		return tProto
	case "ws":
		tProto := new(WSProto)
		tProto.domain = param.Domain
		tProto.conn = param.Conn
		return tProto
	}
	return nil
}

func NewUpMsg(conn net.Conn, secret string, uuid string) Message {
	switch Upstream {
	case "raw":
		tMessage := new(RawMessage)
		tMessage.Conn = conn
		tMessage.UUID = uuid
		tMessage.CryptoSecret = crypto.KeyPadding([]byte(secret))
		return tMessage
	case "ws":
		tMessage := new(WSMessage)
		tMessage.RawMessage = new(RawMessage)
		tMessage.RawMessage.Conn = conn
		tMessage.RawMessage.UUID = uuid
		tMessage.RawMessage.CryptoSecret = crypto.KeyPadding([]byte(secret))
		return tMessage
	case "http":
		tMessage := new(HTTPMessage)
		tMessage.RawMessage = new(RawMessage)
		tMessage.RawMessage.Conn = conn
		tMessage.RawMessage.UUID = uuid
		tMessage.RawMessage.CryptoSecret = crypto.KeyPadding([]byte(secret))
		return tMessage
	}
	return nil
}

func NewDownMsg(conn net.Conn, secret string, uuid string) Message {
	switch Downstream {
	case "raw":
		tMessage := new(RawMessage)
		tMessage.Conn = conn
		tMessage.UUID = uuid
		tMessage.CryptoSecret = crypto.KeyPadding([]byte(secret))
		return tMessage
	case "ws":
		tMessage := new(WSMessage)
		tMessage.RawMessage = new(RawMessage)
		tMessage.RawMessage.Conn = conn
		tMessage.RawMessage.UUID = uuid
		tMessage.RawMessage.CryptoSecret = crypto.KeyPadding([]byte(secret))
		return tMessage
	case "http":
		tMessage := new(HTTPMessage)
		tMessage.RawMessage = new(RawMessage)
		tMessage.RawMessage.Conn = conn
		tMessage.RawMessage.UUID = uuid
		tMessage.RawMessage.CryptoSecret = crypto.KeyPadding([]byte(secret))
		return tMessage
	}
	return nil
}
