package handler

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"Stowaway/agent/manager"
	"Stowaway/global"
	"Stowaway/protocol"
	"Stowaway/utils"
)

type Socks struct {
	Username string
	Password string
}

type Setting struct {
	method       string
	isAuthed     bool
	tcpConnected bool
	isUDP        bool
	success      bool
	tcpConn      net.Conn
	udpListener  *net.UDPConn
}

func newSocks() *Socks {
	return new(Socks)
}

func (socks *Socks) start(mgr *manager.Manager) {
	sMessage := protocol.PrepareAndDecideWhichSProtoToUpper(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)

	header := &protocol.Header{
		Sender:      global.G_Component.UUID,
		Accepter:    protocol.ADMIN_UUID,
		MessageType: protocol.SOCKSREADY,
		RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))), // No need to set route when agent send mess to admin
		Route:       protocol.TEMP_ROUTE,
	}

	succMess := &protocol.SocksReady{
		OK: 1,
	}

	failMess := &protocol.SocksReady{
		OK: 0,
	}

	mgrTask := &manager.SocksTask{
		Mode: manager.S_CHECKSOCKSREADY, // to make sure the map is clean
	}
	mgr.SocksManager.TaskChan <- mgrTask
	result := <-mgr.SocksManager.ResultChan
	if !result.OK {
		protocol.ConstructMessage(sMessage, header, failMess, false)
		sMessage.SendMessage()
		return
	}

	protocol.ConstructMessage(sMessage, header, succMess, false)
	sMessage.SendMessage()
}

func (socks *Socks) handleSocks(mgr *manager.Manager, dataChan chan []byte, seq uint64) {
	setting := new(Setting)

	sMessage := protocol.PrepareAndDecideWhichSProtoToUpper(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)

	defer func() { // no matter what happened, after the function return,tell admin that works done
		finHeader := &protocol.Header{
			Sender:      global.G_Component.UUID,
			Accepter:    protocol.ADMIN_UUID,
			MessageType: protocol.SOCKSTCPFIN,
			RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))), // No need to set route when agent send mess to admin
			Route:       protocol.TEMP_ROUTE,
		}

		finMess := &protocol.SocksTCPFin{
			Seq: seq,
		}

		protocol.ConstructMessage(sMessage, finHeader, finMess, false)
		sMessage.SendMessage()
	}()

	for {
		if !setting.isAuthed && setting.method == "" {
			data, ok := <-dataChan
			if !ok { //重连后原先引用失效，当chan释放后，若不捕捉，会无限循环
				return
			}
			socks.checkMethod(setting, data, seq)
		} else if !setting.isAuthed && setting.method == "PASSWORD" {
			data, ok := <-dataChan
			if !ok {
				return
			}

			socks.auth(setting, data, seq)
		} else if setting.isAuthed && !setting.tcpConnected && !setting.isUDP {
			data, ok := <-dataChan
			if !ok {
				return
			}

			socks.buildConn(mgr, setting, data, seq)

			if !setting.tcpConnected && !setting.isUDP {
				return
			}
		} else if setting.isAuthed && setting.tcpConnected && !setting.isUDP { //All done!
			go proxyC2STCP(setting.tcpConn, dataChan)
			proxyS2CTCP(setting.tcpConn, seq)
			return
		} else if setting.isAuthed && setting.isUDP && setting.success {
			go proxyC2SUDP(mgr, setting.udpListener, seq)
			proxyS2CUDP(mgr, setting.udpListener, seq)
			return
		} else {
			return
		}
	}
}

func (socks *Socks) checkMethod(setting *Setting, data []byte, seq uint64) {
	sMessage := protocol.PrepareAndDecideWhichSProtoToUpper(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)

	header := &protocol.Header{
		Sender:      global.G_Component.UUID,
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
		Data:    []byte{0x05, 0x02},
	}

	// avoid the scenario that we can get full socks protocol header (rarely happen,just in case)
	defer func() {
		if r := recover(); r != nil {
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
			protocol.ConstructMessage(sMessage, header, failMess, false)
			sMessage.SendMessage()
			setting.method = "ILLEGAL"
			return
		}

		if noAuthFinded && (socks.Username == "" && socks.Password == "") {
			protocol.ConstructMessage(sMessage, header, noneMess, false)
			sMessage.SendMessage()
			setting.method = "NONE"
			setting.isAuthed = true
			return
		} else if userPassFinded && (socks.Username != "" && socks.Password != "") {
			protocol.ConstructMessage(sMessage, header, passMess, false)
			sMessage.SendMessage()
			setting.method = "PASSWORD"
			return
		} else {
			protocol.ConstructMessage(sMessage, header, failMess, false)
			sMessage.SendMessage()
			setting.method = "ILLEGAL"
			return
		}
	}
	// send nothing
	setting.method = "ILLEGAL"
}

func (socks *Socks) auth(setting *Setting, data []byte, seq uint64) {
	sMessage := protocol.PrepareAndDecideWhichSProtoToUpper(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)

	header := &protocol.Header{
		Sender:      global.G_Component.UUID,
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
			setting.isAuthed = false
		}
	}()

	ulen := int(data[1])
	slen := int(data[2+ulen])
	clientName := string(data[2 : 2+ulen])
	clientPass := string(data[3+ulen : 3+ulen+slen])

	if clientName != socks.Username || clientPass != socks.Password {
		protocol.ConstructMessage(sMessage, header, failMess, false)
		sMessage.SendMessage()
		setting.isAuthed = false
		return
	}
	// username && password all fits!
	protocol.ConstructMessage(sMessage, header, succMess, false)
	sMessage.SendMessage()
	setting.isAuthed = true
}

func (socks *Socks) buildConn(mgr *manager.Manager, setting *Setting, data []byte, seq uint64) {
	sMessage := protocol.PrepareAndDecideWhichSProtoToUpper(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)

	header := &protocol.Header{
		Sender:      global.G_Component.UUID,
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
		protocol.ConstructMessage(sMessage, header, failMess, false)
		sMessage.SendMessage()
		return
	}

	if data[0] == 0x05 {
		switch data[1] {
		case 0x01:
			tcpConnect(mgr, setting, data, seq, length)
		case 0x02:
			tcpBind(mgr, setting, data, seq, length)
		case 0x03:
			udpAssociate(mgr, setting, data, seq, length)
		default:
			protocol.ConstructMessage(sMessage, header, failMess, false)
			sMessage.SendMessage()
		}
	}
}

// TCPConnect 如果是代理tcp
func tcpConnect(mgr *manager.Manager, setting *Setting, data []byte, seq uint64, length int) {
	var host string
	var err error

	sMessage := protocol.PrepareAndDecideWhichSProtoToUpper(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)

	header := &protocol.Header{
		Sender:      global.G_Component.UUID,
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
		protocol.ConstructMessage(sMessage, header, failMess, false)
		sMessage.SendMessage()
		setting.tcpConnected = false
		return
	}

	port := utils.Int2Str(int(data[length-2])<<8 | int(data[length-1]))

	setting.tcpConn, err = net.DialTimeout("tcp", net.JoinHostPort(host, port), 10*time.Second)

	if err != nil {
		protocol.ConstructMessage(sMessage, header, failMess, false)
		sMessage.SendMessage()
		setting.tcpConnected = false
		return
	}

	mgrTask := &manager.SocksTask{
		Mode: manager.S_CHECKTCP,
		Seq:  seq,
	}
	mgr.SocksManager.TaskChan <- mgrTask
	socksResult := <-mgr.SocksManager.ResultChan
	if !socksResult.OK { // if admin has already send fin,then close the conn and set setting.tcpConnected -> false
		setting.tcpConn.Close()
		protocol.ConstructMessage(sMessage, header, failMess, false)
		sMessage.SendMessage()
		setting.tcpConnected = false
		return
	}

	protocol.ConstructMessage(sMessage, header, succMess, false)
	sMessage.SendMessage()
	setting.tcpConnected = true
}

func proxyC2STCP(conn net.Conn, dataChan chan []byte) {
	for {
		data, ok := <-dataChan
		if !ok { // no need to send FIN actively
			conn.Close()
			return
		}
		conn.Write(data)
	}
}

func proxyS2CTCP(conn net.Conn, seq uint64) {
	sMessage := protocol.PrepareAndDecideWhichSProtoToUpper(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)

	header := &protocol.Header{
		Sender:      global.G_Component.UUID,
		Accepter:    protocol.ADMIN_UUID,
		MessageType: protocol.SOCKSTCPDATA,
		RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))), // No need to set route when agent send mess to admin
		Route:       protocol.TEMP_ROUTE,
	}

	buffer := make([]byte, 20480)
	for {
		length, err := conn.Read(buffer)
		if err != nil {
			conn.Close() // close conn immediately
			return
		}

		dataMess := &protocol.SocksTCPData{
			Seq:     seq,
			DataLen: uint64(length),
			Data:    buffer[:length],
		}

		protocol.ConstructMessage(sMessage, header, dataMess, false)
		sMessage.SendMessage()
	}
}

// TCPBind TCPBind方式
func tcpBind(mgr *manager.Manager, setting *Setting, data []byte, seq uint64, length int) {
	fmt.Println("Not ready") //limited use, add to Todo
	setting.tcpConnected = false
}

type socksLocalAddr struct {
	Host string
	Port int
}

func (addr *socksLocalAddr) byteArray() []byte {
	bytes := make([]byte, 6)
	copy(bytes[:4], net.ParseIP(addr.Host).To4())
	bytes[4] = byte(addr.Port >> 8)
	bytes[5] = byte(addr.Port % 256)
	return bytes
}

// Based on rfc1928,agent must send message strictly
// UDPAssociate UDPAssociate方式
func udpAssociate(mgr *manager.Manager, setting *Setting, data []byte, seq uint64, length int) {
	setting.isUDP = true

	sMessage := protocol.PrepareAndDecideWhichSProtoToUpper(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)

	dataHeader := &protocol.Header{
		Sender:      global.G_Component.UUID,
		Accepter:    protocol.ADMIN_UUID,
		MessageType: protocol.SOCKSTCPDATA,
		RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))),
		Route:       protocol.TEMP_ROUTE,
	}

	assHeader := &protocol.Header{
		Sender:      global.G_Component.UUID,
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
		protocol.ConstructMessage(sMessage, dataHeader, failMess, false)
		sMessage.SendMessage()
		setting.success = false
		return
	}

	port := utils.Int2Str(int(data[length-2])<<8 | int(data[length-1])) //先拿到客户端想要发送数据的ip:port地址

	udpListenerAddr, err := net.ResolveUDPAddr("udp", "0.0.0.0:0")
	if err != nil {
		protocol.ConstructMessage(sMessage, dataHeader, failMess, false)
		sMessage.SendMessage()
		setting.success = false
		return
	}

	udpListener, err := net.ListenUDP("udp", udpListenerAddr)
	if err != nil {
		protocol.ConstructMessage(sMessage, dataHeader, failMess, false)
		sMessage.SendMessage()
		setting.success = false
		return
	}

	sourceAddr := net.JoinHostPort(host, port)

	mgrTask := &manager.SocksTask{
		Mode: manager.S_CHECKUDP,
		Seq:  seq,
	}

	mgr.SocksManager.TaskChan <- mgrTask
	socksResult := <-mgr.SocksManager.ResultChan
	if !socksResult.OK {
		udpListener.Close() // close listener,because tcp conn is closed
		protocol.ConstructMessage(sMessage, dataHeader, failMess, false)
		sMessage.SendMessage()
		setting.success = false
		return
	}

	mgrTask = &manager.SocksTask{
		Mode: manager.S_GETUDPCHANS,
		Seq:  seq,
	}
	mgr.SocksManager.TaskChan <- mgrTask
	socksResult = <-mgr.SocksManager.ResultChan

	if !socksResult.OK { // no need to close listener,cuz TCPFIN has helped us
		protocol.ConstructMessage(sMessage, dataHeader, failMess, false)
		sMessage.SendMessage()
		setting.success = false
		return
	}

	readyChan := socksResult.ReadyChan

	assMess := &protocol.UDPAssStart{
		Seq:           seq,
		SourceAddrLen: uint16(len([]byte(sourceAddr))),
		SourceAddr:    sourceAddr,
	}

	protocol.ConstructMessage(sMessage, assHeader, assMess, false)
	sMessage.SendMessage()

	if adminResponse, ok := <-readyChan; adminResponse != "" && ok {
		temp := strings.Split(adminResponse, ":")
		adminAddr := temp[0]
		adminPort, _ := strconv.Atoi(temp[1])

		localAddr := socksLocalAddr{adminAddr, adminPort}
		buf := make([]byte, 10)
		copy(buf, []byte{0x05, 0x00, 0x00, 0x01})
		copy(buf[4:], localAddr.byteArray())

		dataMess := &protocol.SocksTCPData{
			Seq:     seq,
			DataLen: 10,
			Data:    buf,
		}

		protocol.ConstructMessage(sMessage, dataHeader, dataMess, false)
		sMessage.SendMessage()

		setting.udpListener = udpListener
		setting.success = true
		return
	}

	protocol.ConstructMessage(sMessage, dataHeader, failMess, false)
	sMessage.SendMessage()
	setting.success = false
}

// proxyC2SUDP 代理C-->Sudp流量
func proxyC2SUDP(mgr *manager.Manager, listener *net.UDPConn, seq uint64) {
	mgrTask := &manager.SocksTask{
		Mode: manager.S_GETUDPCHANS,
		Seq:  seq,
	}
	mgr.SocksManager.TaskChan <- mgrTask
	result := <-mgr.SocksManager.ResultChan
	// no need to check if OK,cuz if not,"data, ok := <-dataChan" will help us to exit
	dataChan := result.DataChan

	defer func() {
		// Just avoid panic
		if r := recover(); r != nil {
			go func() { //continue to read channel,avoid some remaining data sended by admin blocking our dispatcher
				for {
					_, ok := <-dataChan
					if !ok {
						return
					}
				}
			}()
		}
	}()

	for {
		var remote string
		var udpData []byte

		data, ok := <-dataChan
		if !ok {
			listener.Close()
			return
		}

		buf := []byte(data)

		if buf[0] != 0x00 || buf[1] != 0x00 || buf[2] != 0x00 {
			continue
		}

		udpHeader := make([]byte, 0, 1024)
		addrtype := buf[3]

		if addrtype == 0x01 { //IPV4
			ip := net.IPv4(buf[4], buf[5], buf[6], buf[7])
			remote = fmt.Sprintf("%s:%d", ip.String(), uint(buf[8])<<8+uint(buf[9]))
			udpData = buf[10:]
			udpHeader = append(udpHeader, buf[:10]...)
		} else if addrtype == 0x03 { //DOMAIN
			nmlen := int(buf[4])
			nmbuf := buf[5 : 5+nmlen+2]
			remote = fmt.Sprintf("%s:%d", nmbuf[:nmlen], uint(nmbuf[nmlen])<<8+uint(nmbuf[nmlen+1]))
			udpData = buf[8+nmlen:]
			udpHeader = append(udpHeader, buf[:8+nmlen]...)
		} else if addrtype == 0x04 { //IPV6
			ip := net.IP{buf[4], buf[5], buf[6], buf[7],
				buf[8], buf[9], buf[10], buf[11], buf[12],
				buf[13], buf[14], buf[15], buf[16], buf[17],
				buf[18], buf[19]}
			remote = fmt.Sprintf("[%s]:%d", ip.String(), uint(buf[20])<<8+uint(buf[21]))
			udpData = buf[22:]
			udpHeader = append(udpHeader, buf[:22]...)
		} else {
			continue
		}

		remoteAddr, err := net.ResolveUDPAddr("udp", remote)
		if err != nil {
			continue
		}

		mgrTask = &manager.SocksTask{
			Mode:            manager.S_UPDATEUDPHEADER,
			Seq:             seq,
			SocksHeaderAddr: remote,
			SocksHeader:     udpHeader,
		}
		mgr.SocksManager.TaskChan <- mgrTask
		<-mgr.SocksManager.ResultChan

		listener.WriteToUDP(udpData, remoteAddr)
	}
}

// proxyS2CUDP 代理S-->Cudp流量
func proxyS2CUDP(mgr *manager.Manager, listener *net.UDPConn, seq uint64) {
	sMessage := protocol.PrepareAndDecideWhichSProtoToUpper(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)

	header := &protocol.Header{
		Sender:      global.G_Component.UUID,
		Accepter:    protocol.ADMIN_UUID,
		MessageType: protocol.SOCKSUDPDATA,
		RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))),
		Route:       protocol.TEMP_ROUTE,
	}

	buffer := make([]byte, 20480)
	var data []byte
	var finalLength int

	for {
		length, addr, err := listener.ReadFromUDP(buffer)
		if err != nil {
			listener.Close()
			return
		}

		mgrTask := &manager.SocksTask{
			Mode:            manager.S_GETUDPHEADER,
			Seq:             seq,
			SocksHeaderAddr: addr.String(),
		}
		mgr.SocksManager.TaskChan <- mgrTask
		result := <-mgr.SocksManager.ResultChan
		if result.OK {
			finalLength = len(result.SocksUDPHeader) + length
			data = make([]byte, 0, finalLength)
			data = append(data, result.SocksUDPHeader...)
			data = append(data, buffer[:length]...)
		} else {
			return
		}

		dataMess := &protocol.SocksUDPData{
			Seq:     seq,
			DataLen: uint64(finalLength),
			Data:    data,
		}

		protocol.ConstructMessage(sMessage, header, dataMess, false)
		sMessage.SendMessage()
	}
}

func DispathSocksMess(mgr *manager.Manager) {
	socks := newSocks()

	for {
		message := <-mgr.SocksManager.SocksMessChan

		switch mess := message.(type) {
		case *protocol.SocksStart:
			socks.Username = mess.Username
			socks.Password = mess.Password
			go socks.start(mgr)
		case *protocol.SocksTCPData:
			mgrTask := &manager.SocksTask{
				Mode: manager.S_GETTCPDATACHAN,
				Seq:  mess.Seq,
			}
			mgr.SocksManager.TaskChan <- mgrTask
			result := <-mgr.SocksManager.ResultChan

			result.DataChan <- mess.Data

			// if not exist
			if !result.SocksSeqExist {
				go socks.handleSocks(mgr, result.DataChan, mess.Seq)
			}
		case *protocol.SocksTCPFin:
			mgrTask := &manager.SocksTask{
				Mode: manager.S_CLOSETCP,
				Seq:  mess.Seq,
			}
			mgr.SocksManager.TaskChan <- mgrTask
		case *protocol.SocksUDPData:
			mgrTask := &manager.SocksTask{
				Mode: manager.S_GETUDPCHANS,
				Seq:  mess.Seq,
			}
			mgr.SocksManager.TaskChan <- mgrTask
			result := <-mgr.SocksManager.ResultChan

			if result.OK {
				result.DataChan <- mess.Data
			}
		case *protocol.UDPAssRes:
			mgrTask := &manager.SocksTask{
				Mode: manager.S_GETUDPCHANS,
				Seq:  mess.Seq,
			}
			mgr.SocksManager.TaskChan <- mgrTask
			result := <-mgr.SocksManager.ResultChan

			if result.OK {
				result.ReadyChan <- mess.Addr
			}
		}

	}
}
