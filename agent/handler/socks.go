/*
 * @Author: ph4ntom
 * @Date: 2021-03-23 18:57:46
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-03-26 16:52:38
 */
package handler

import (
	"Stowaway/agent/manager"
	"Stowaway/protocol"
	"Stowaway/utils"
	"fmt"
	"net"
	"strconv"
	"strings"
)

type Socks struct {
	Username    string
	Password    string
	TCPDataChan chan *protocol.SocksTCPData
}

type Setting struct {
	method       string
	isAuthed     bool
	tcpConnected bool
	isUDP        bool
	success      bool
}

func NewSocks(username, password string) *Socks {
	socks := new(Socks)
	socks.Username = username
	socks.Password = password
	socks.TCPDataChan = make(chan *protocol.SocksTCPData, 10)
	return socks
}

func (socks *Socks) Start(mgr *manager.Manager, component *protocol.MessageComponent) {
	for {
		socksData := <-socks.TCPDataChan

		// check if seq num has already existed
		task := &manager.ManagerTask{
			Mode:          manager.S_CHECKIFSEQEXIST,
			Category:      manager.SOCKS,
			SocksSequence: socksData.Seq,
		}
		mgr.TaskChan <- task
		result := <-mgr.SocksResultChan

		// if not exist
		if !result.SocksSeqExist {
			result.DataChan <- socksData.Data
			go socks.handleSocks(mgr, component, result.DataChan, socksData.Seq)
		}

	}
}

func (socks *Socks) handleSocks(mgr *manager.Manager, component *protocol.MessageComponent, dataChan chan []byte, seq uint64) {
	setting := new(Setting)

	for {
		if setting.isAuthed == false && setting.method == "" {
			data, ok := <-dataChan
			if !ok { //重连后原先引用失效，当chan释放后，若不捕捉，会无限循环
				return
			}
			socks.checkMethod(component, setting, data, seq)
		} else if setting.isAuthed == false && setting.method == "PASSWORD" {
			data, ok := <-dataChan
			if !ok {
				return
			}

			socks.auth(component, setting, data, seq)
		} else if setting.isAuthed == true && setting.tcpConnected == false && !setting.isUDP {
			data, ok := <-dataChan
			if !ok {
				return
			}

			buildConn(mgr, component, setting, data, seq)
			if setting.tcpConnected == false && !setting.isUDP {
				return
			}

		} else if setting.isAuthed == true && setting.tcpConnected == true && !setting.isUDP { //All done!
			defer SendTCPFin(component, seq)

			go ProxyC2STCP(info, server, checkNum)

			if err := ProxyS2CTCP(ConnToAdmin, server, checkNum, AgentStatus.AESKey, currentid); err != nil {
				return
			}
		} else if isAuthed == true && isUDP && success {
			defer SendUDPFin(checkNum)

			go ProxyC2SUDP(checkNum)

			if err := ProxyS2CUDP(ConnToAdmin, checkNum, AgentStatus.AESKey, currentid); err != nil {
				return
			}
		} else {
			return
		}
	}
}

func (socks *Socks) checkMethod(component *protocol.MessageComponent, setting *Setting, data []byte, seq uint64) {
	sMessage := protocol.PrepareAndDecideWhichSProtoToUpper(component.Conn, component.Secret, component.UUID)

	header := &protocol.Header{
		Sender:      component.UUID,
		Accepter:    protocol.ADMIN_UUID,
		MessageType: protocol.SOCKSTCPDATA,
		RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))), // No need to set route when agent send mess to admin
		Route:       protocol.TEMP_ROUTE,
	}

	failMess := &protocol.SocksTCPData{
		Seq:     seq,
		DataLen: uint64(len([]byte{0x05, 0xff})),
		Data:    []byte{0x05, 0xff},
	}

	noneMess := &protocol.SocksTCPData{
		Seq:     seq,
		DataLen: uint64(len([]byte{0x05, 0x00})),
		Data:    []byte{0x05, 0x00},
	}

	passMess := &protocol.SocksTCPData{
		Seq:     seq,
		DataLen: uint64(len([]byte{0x05, 0x02})),
		Data:    []byte{0x05, 0x00},
	}

	// avoid the situation that we can get full socks protocol header (rarely happen,just in case)
	defer func() {
		if r := recover(); r != nil {
			protocol.ConstructMessage(sMessage, header, failMess)
			sMessage.SendMessage()
			setting.method = "ILLEGAL"
		}
	}()

	if data[0] == 0x05 {
		nMethods := int(data[1])

		var supportMethodFinded, userPassFinded, noAuthFinded bool

		for _, method := range data[2 : 2+nMethods] {
			if method == 0x00 {
				noAuthFinded = true
				supportMethodFinded = true
			} else if method == 0x02 {
				userPassFinded = true
				supportMethodFinded = true
			}
		}

		if !supportMethodFinded {
			protocol.ConstructMessage(sMessage, header, failMess)
			sMessage.SendMessage()
			setting.method = "ILLEGAL"
		}

		if noAuthFinded && (socks.Username == "" && socks.Password == "") {
			protocol.ConstructMessage(sMessage, header, noneMess)
			sMessage.SendMessage()
			setting.method = "NONE"
			setting.isAuthed = true
		} else if userPassFinded && (socks.Username != "" && socks.Password != "") {
			protocol.ConstructMessage(sMessage, header, passMess)
			sMessage.SendMessage()
			setting.method = "PASSWORD"
		} else {
			protocol.ConstructMessage(sMessage, header, failMess)
			sMessage.SendMessage()
			setting.method = "ILLEGAL"
		}
	}
	// send nothing
	setting.method = "ILLEGAL"
}

func (socks *Socks) auth(component *protocol.MessageComponent, setting *Setting, data []byte, seq uint64) {
	sMessage := protocol.PrepareAndDecideWhichSProtoToUpper(component.Conn, component.Secret, component.UUID)

	header := &protocol.Header{
		Sender:      component.UUID,
		Accepter:    protocol.ADMIN_UUID,
		MessageType: protocol.SOCKSTCPDATA,
		RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))),
		Route:       protocol.TEMP_ROUTE,
	}

	failMess := &protocol.SocksTCPData{
		Seq:     seq,
		DataLen: uint64(len([]byte{0x01, 0x01})),
		Data:    []byte{0x01, 0x01},
	}

	succMess := &protocol.SocksTCPData{
		Seq:     seq,
		DataLen: uint64(len([]byte{0x01, 0x00})),
		Data:    []byte{0x01, 0x00},
	}

	defer func() {
		if r := recover(); r != nil {
			protocol.ConstructMessage(sMessage, header, failMess)
			sMessage.SendMessage()
			setting.isAuthed = false
		}
	}()

	ulen := int(data[1])
	slen := int(data[2+ulen])
	clientName := string(data[2 : 2+ulen])
	clientPass := string(data[3+ulen : 3+ulen+slen])

	if clientName != socks.Username || clientPass != socks.Password {
		protocol.ConstructMessage(sMessage, header, failMess)
		sMessage.SendMessage()
		setting.isAuthed = false
	}
	// username && password all fits!
	protocol.ConstructMessage(sMessage, header, succMess)
	sMessage.SendMessage()
	setting.isAuthed = true
}

func buildConn(mgr *manager.Manager, component *protocol.MessageComponent, setting *Setting, data []byte, seq uint64) {
	var connected, isUDP, success bool

	sMessage := protocol.PrepareAndDecideWhichSProtoToUpper(component.Conn, component.Secret, component.UUID)

	header := &protocol.Header{
		Sender:      component.UUID,
		Accepter:    protocol.ADMIN_UUID,
		MessageType: protocol.SOCKSTCPDATA,
		RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))), // No need to set route when agent send mess to admin
		Route:       protocol.TEMP_ROUTE,
	}

	failMess := &protocol.SocksTCPData{
		Seq:     seq,
		DataLen: uint64(len([]byte{0x05, 0x04, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})),
		Data:    []byte{0x05, 0x04, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
	}

	length := len(data)

	if length <= 2 {
		protocol.ConstructMessage(sMessage, header, failMess)
		sMessage.SendMessage()
		return connected, isUDP, success
	}

	if data[0] == 0x05 {
		switch data[1] {
		case 0x01:
			TCPConnect(mgr, component, setting, data, seq, length)
		case 0x02:
			TCPBind(mgr, component, setting, data, seq, length)
		case 0x03:
			UDPAssociate(mgr, component, setting, data, seq, length)
		default:
			protocol.ConstructMessage(sMessage, header, failMess)
			sMessage.SendMessage()
		}
	}
}

// TCPConnect 如果是代理tcp
func TCPConnect(mgr *manager.Manager, component *protocol.MessageComponent, setting *Setting, data []byte, seq uint64, length int) {
	var host string
	var err error

	sMessage := protocol.PrepareAndDecideWhichSProtoToUpper(component.Conn, component.Secret, component.UUID)

	header := &protocol.Header{
		Sender:      component.UUID,
		Accepter:    protocol.ADMIN_UUID,
		MessageType: protocol.SOCKSTCPDATA,
		RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))),
		Route:       protocol.TEMP_ROUTE,
	}

	failMess := &protocol.SocksTCPData{
		Seq:     seq,
		DataLen: uint64(len([]byte{0x05, 0x04, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})),
		Data:    []byte{0x05, 0x04, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
	}

	succMess := &protocol.SocksTCPData{
		Seq:     seq,
		DataLen: uint64(len([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})),
		Data:    []byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
	}

	defer func() {
		if r := recover(); r != nil {
			protocol.ConstructMessage(sMessage, header, failMess)
			sMessage.SendMessage()
			setting.tcpConnected = false
		}
	}()

	switch data[3] {
	case 0x01:
		host = net.IPv4(data[4], data[5], data[6], data[7]).String()
	case 0x03:
		host = string(data[5 : length-2])
	case 0x04:
		host = net.IP{data[4], data[5], data[6], data[7],
			data[8], data[9], data[10], data[11], data[12],
			data[13], data[14], data[15], data[16], data[17],
			data[18], data[19]}.String()
	default:
		protocol.ConstructMessage(sMessage, header, failMess)
		sMessage.SendMessage()
		setting.tcpConnected = false
		return
	}

	port := utils.Int2Str(int(data[length-2])<<8 | int(data[length-1]))

	conn, err = net.Dial("tcp", net.JoinHostPort(host, port))

	if err != nil {
		protocol.ConstructMessage(sMessage, header, failMess)
		sMessage.SendMessage()
		setting.tcpConnected = false
		return
	}

	task := &manager.ManagerTask{
		Mode:          manager.S_UPDATETCP,
		Category:      manager.SOCKS,
		SocksSequence: seq,
		SocksSocket:   setting.conn,
	}
	mgr.TaskChan <- task
	<-mgr.SocksReadyChan

	protocol.ConstructMessage(sMessage, header, succMess)
	sMessage.SendMessage()
	setting.tcpConnected = true
}

// SendTCPFin 发送tcp server offline通知
func SendTCPFin(component *protocol.MessageComponent, seq uint64) {
	sMessage := protocol.PrepareAndDecideWhichSProtoToUpper(component.Conn, component.Secret, component.UUID)

	respData, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "FIN", " ", " ", num, AgentStatus.Nodeid, AgentStatus.AESKey, false)
	AgentStuff.ProxyChan.ProxyChanToUpperNode <- respData
}

// TCPBind TCPBind方式
func TCPBind(mgr *manager.Manager, component *protocol.MessageComponent, setting *Setting, data []byte, seq uint64, length int) {
	fmt.Println("Not ready") //limited use, add to Todo
	setting.tcpConnected = false
}

// 基于rfc1928编写，如果客户端没有严格按照rfc1928规定发送数据包，可能导致agent崩溃！
// UDPAssociate UDPAssociate方式
func UDPAssociate(mgr *manager.Manager, component *protocol.MessageComponent, setting *Setting, data []byte, seq uint64, length int) {
	setting.isUDP = true

	sMessage := protocol.PrepareAndDecideWhichSProtoToUpper(component.Conn, component.Secret, component.UUID)

	dataHeader := &protocol.Header{
		Sender:      component.UUID,
		Accepter:    protocol.ADMIN_UUID,
		MessageType: protocol.SOCKSTCPDATA,
		RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))),
		Route:       protocol.TEMP_ROUTE,
	}

	assHeader := &protocol.Header{
		Sender:      component.UUID,
		Accepter:    protocol.ADMIN_UUID,
		MessageType: protocol.UDPASSSTART,
		RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))),
		Route:       protocol.TEMP_ROUTE,
	}

	failMess := &protocol.SocksTCPData{
		Seq:     seq,
		DataLen: uint64(len([]byte{0x05, 0x01, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})),
		Data:    []byte{0x05, 0x01, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
	}

	defer func() {
		if r := recover(); r != nil {
			protocol.ConstructMessage(sMessage, dataHeader, failMess)
			sMessage.SendMessage()
			setting.success = false
		}
	}()

	var host string
	switch data[3] {
	case 0x01:
		host = net.IPv4(data[4], data[5], data[6], data[7]).String()
	case 0x03:
		host = string(data[5 : length-2])
	case 0x04:
		host = net.IP{data[4], data[5], data[6], data[7],
			data[8], data[9], data[10], data[11], data[12],
			data[13], data[14], data[15], data[16], data[17],
			data[18], data[19]}.String()
	default:
		protocol.ConstructMessage(sMessage, dataHeader, failMess)
		sMessage.SendMessage()
		setting.success = false
		return
	}

	port := utils.Int2Str(int(data[length-2])<<8 | int(data[length-1])) //先拿到客户端想要发送数据的ip:port地址

	udpListenerAddr, err := net.ResolveUDPAddr("udp", "0.0.0.0:0")
	if err != nil {
		protocol.ConstructMessage(sMessage, dataHeader, failMess)
		sMessage.SendMessage()
		setting.success = false
		return
	}

	udpListener, err := net.ListenUDP("udp", udpListenerAddr)
	if err != nil {
		protocol.ConstructMessage(sMessage, dataHeader, failMess)
		sMessage.SendMessage()
		setting.success = false
		return
	}

	sourceAddr := net.JoinHostPort(host, port)

	task := &manager.ManagerTask{
		Mode:            manager.S_UPDATEUDP,
		Category:        manager.SOCKS,
		SocksSequence:   seq,
		SocksListener:   udpListener,
		SocksSourceAddr: sourceAddr,
	}

	mgr.TaskChan <- task
	<-mgr.SocksReadyChan

	assMess := &protocol.UDPAssStart{
		Seq:           seq,
		SourceAddrLen: uint16(len([]byte(sourceAddr))),
		SourceAddr:    sourceAddr,
	}

	protocol.ConstructMessage(sMessage, assHeader, assMess)
	sMessage.SendMessage()

	if adminResponse := <-AgentStuff.Socks5UDPAssociate.Info[checkNum].Ready; adminResponse != "" {
		temp := strings.Split(adminResponse, ":")
		adminAddr := temp[0]
		adminPort, _ := strconv.Atoi(temp[1])

		localAddr := utils.SocksLocalAddr{adminAddr, adminPort}
		buf := make([]byte, 10)
		copy(buf, []byte{0x05, 0x00, 0x00, 0x01})
		copy(buf[4:], localAddr.ByteArray())

		utils.ConstructPayloadAndSend(connToUpper, utils.AdminId, "", "DATA", "TSOCKSDATARESP", " ", string(buf), checkNum, currentid, key, false)
		setting.success = true
		return
	}

	utils.ConstructPayloadAndSend(connToUpper, utils.AdminId, "", "DATA", "TSOCKSDATARESP", " ", string([]byte{0x05, 0x01, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}), checkNum, currentid, key, false)
	setting.success = false
}
