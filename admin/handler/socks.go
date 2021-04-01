/*
 * @Author: ph4ntom
 * @Date: 2021-03-19 18:40:13
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-04-01 19:37:29
 */
package handler

import (
	"Stowaway/admin/manager"
	"Stowaway/admin/topology"
	"Stowaway/protocol"
	"errors"
	"fmt"
	"net"
)

type Socks struct {
	Username string
	Password string
	Port     string
}

func NewSocks() *Socks {
	return new(Socks)
}

func (socks *Socks) LetSocks(component *protocol.MessageComponent, mgr *manager.Manager, route string, uuid string, uuidNum int) error {
	sMessage := protocol.PrepareAndDecideWhichSProtoToLower(component.Conn, component.Secret, component.UUID)

	header := &protocol.Header{
		Sender:      protocol.ADMIN_UUID,
		Accepter:    uuid,
		MessageType: protocol.SOCKSSTART,
		RouteLen:    uint32(len([]byte(route))),
		Route:       route,
	}

	socksStartMess := &protocol.SocksStart{
		UsernameLen: uint64(len([]byte(socks.Username))),
		Username:    socks.Username,
		PasswordLen: uint64(len([]byte(socks.Password))),
		Password:    socks.Password,
	}

	protocol.ConstructMessage(sMessage, header, socksStartMess)
	sMessage.SendMessage()

	if ready := <-mgr.SocksReady; !ready {
		err := errors.New("[*]Fail to start socks.If you just stop socks service,please wait for the cleanup done")
		return err
	}

	addr := fmt.Sprintf("0.0.0.0:%s", socks.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	// register brand new socks service
	mgrTask := &manager.ManagerTask{
		Category:         manager.SOCKS,
		Mode:             manager.S_NEWSOCKS,
		UUIDNum:          uuidNum,
		SocksPort:        socks.Port,
		SocksUsername:    socks.Username,
		SocksPassword:    socks.Password,
		SocksTCPListener: listener,
	}

	mgr.TaskChan <- mgrTask
	result := <-mgr.ResultChan // wait for "add" done
	if !result.OK {            // node and socks service must be one-to-one
		listener.Close()
		return err
	}

	go socks.handleListener(component, mgr, listener, route, uuid, uuidNum)

	return nil
}

func (socks *Socks) handleListener(component *protocol.MessageComponent, mgr *manager.Manager, listener net.Listener, route string, uuid string, uuidNum int) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			listener.Close()
			return
		}

		// ask new seq num
		mgrTask := &manager.ManagerTask{
			Category: manager.SOCKS,
			Mode:     manager.S_GETNEWSEQ,
			UUIDNum:  uuidNum,
		}
		mgr.TaskChan <- mgrTask
		result := <-mgr.ResultChan
		seq := result.SocksID

		// save the socket
		mgrTask = &manager.ManagerTask{
			Category:       manager.SOCKS,
			UUIDNum:        uuidNum,
			Seq:            seq,
			Mode:           manager.S_ADDTCPSOCKET,
			SocksTCPSocket: conn,
		}
		mgr.TaskChan <- mgrTask
		result = <-mgr.ResultChan
		if !result.OK {
			return
		}

		// handle it!
		go socks.handleSocks(component, mgr, conn, route, uuid, uuidNum, seq)
	}
}

func (socks *Socks) handleSocks(component *protocol.MessageComponent, mgr *manager.Manager, conn net.Conn, route string, uuid string, uuidNum int, seq uint64) {
	sMessage := protocol.PrepareAndDecideWhichSProtoToLower(component.Conn, component.Secret, component.UUID)

	header := &protocol.Header{
		Sender:      protocol.ADMIN_UUID,
		Accepter:    uuid,
		MessageType: protocol.SOCKSTCPDATA,
		RouteLen:    uint32(len([]byte(route))),
		Route:       route,
	}

	mgrTask := &manager.ManagerTask{
		Category: manager.SOCKS,
		UUIDNum:  uuidNum,
		Seq:      seq,
		Mode:     manager.S_GETTCPDATACHAN,
	}
	mgr.TaskChan <- mgrTask
	result := <-mgr.ResultChan
	if !result.OK {
		return
	}

	tcpDataChan := result.TCPDataChan

	// handle data from dispatcher
	go func() {
		for {
			if data, ok := <-tcpDataChan; ok {
				conn.Write(data)
			} else {
				return
			}
		}
	}()

	var sendSth bool

	// SendTCPFin after browser close the conn
	defer func() {
		// check if "sendSth" is true
		// if true, then tell agent that the conn is closed
		// but keep "handle received data" working to achieve socksdata from agent that still on the way
		// if false, don't tell agent and do cleanup alone
		if !sendSth {
			// call HandleTCPFin by myself
			mgrTask := &manager.ManagerTask{
				Mode:     manager.S_CLOSETCP,
				Category: manager.SOCKS,
				Seq:      seq,
			}
			mgr.TaskChan <- mgrTask
			return
		}

		finHeader := &protocol.Header{
			Sender:      protocol.ADMIN_UUID,
			Accepter:    uuid,
			MessageType: protocol.SOCKSTCPFIN,
			RouteLen:    uint32(len([]byte(route))),
			Route:       route,
		}
		finMess := &protocol.SocksTCPFin{
			Seq: seq,
		}

		protocol.ConstructMessage(sMessage, finHeader, finMess)
		sMessage.SendMessage()
	}()

	// handling data that comes from browser
	buffer := make([]byte, 20480)

	// try to receive first packet
	// avoid browser to close the conn but sends nothing
	length, err := conn.Read(buffer)
	if err != nil {
		conn.Close() // close conn immediately
		return
	}

	socksDataMess := &protocol.SocksTCPData{
		Seq:     seq,
		DataLen: uint64(length),
		Data:    buffer[:length],
	}

	protocol.ConstructMessage(sMessage, header, socksDataMess)
	sMessage.SendMessage()

	// browser sends sth, so handling conn normally and setting sendSth->true
	for {
		length, err := conn.Read(buffer)
		if err != nil {
			sendSth = true
			conn.Close() // close conn immediately,in case of sth is sended after TCPFin
			return
		}

		socksDataMess := &protocol.SocksTCPData{
			Seq:     seq,
			DataLen: uint64(length),
			Data:    buffer[:length],
		}

		protocol.ConstructMessage(sMessage, header, socksDataMess)
		sMessage.SendMessage()
	}
}

func StartUDPAss(mgr *manager.Manager, topo *topology.Topology, conn net.Conn, secret string, seq uint64) {
	var (
		err             error
		udpListenerAddr *net.UDPAddr
		udpListener     *net.UDPConn
	)

	component := &protocol.MessageComponent{
		Secret: secret,
		Conn:   conn,
		UUID:   protocol.ADMIN_UUID,
	}

	mgrTask := &manager.ManagerTask{
		Category: manager.SOCKS,
		Mode:     manager.S_GETUDPSTARTINFO,
		Seq:      seq,
	}
	mgr.TaskChan <- mgrTask
	socksResult := <-mgr.ResultChan
	uuidNum := socksResult.UUIDNum

	topoTask := &topology.TopoTask{
		Mode:    topology.GETUUID,
		UUIDNum: uuidNum,
	}
	topo.TaskChan <- topoTask
	topoResult := <-topo.ResultChan
	uuid := topoResult.UUID

	topoTask = &topology.TopoTask{
		Mode:    topology.GETROUTE,
		UUIDNum: uuidNum,
	}
	topo.TaskChan <- topoTask
	topoResult = <-topo.ResultChan
	route := topoResult.Route

	header := &protocol.Header{
		Sender:      protocol.ADMIN_UUID,
		Accepter:    uuid,
		MessageType: protocol.UDPASSRES,
		RouteLen:    uint32(len([]byte(route))),
		Route:       route,
	}

	failMess := &protocol.UDPAssRes{
		Seq: seq,
		OK:  0,
	}

	sMessage := protocol.PrepareAndDecideWhichSProtoToLower(component.Conn, component.Secret, component.UUID)

	defer func() {
		if err != nil {
			protocol.ConstructMessage(sMessage, header, failMess)
			sMessage.SendMessage()
		}
	}()

	if socksResult.OK {
		udpListenerAddr, err = net.ResolveUDPAddr("udp", socksResult.TCPAddr+":0")
		if err != nil {
			return
		}

		udpListener, err = net.ListenUDP("udp", udpListenerAddr)
		if err != nil {
			return
		}

		mgrTask = &manager.ManagerTask{
			Category:           manager.SOCKS,
			Mode:               manager.S_UPDATEUDP,
			Seq:                seq,
			SocksUDPListenAddr: udpListener.LocalAddr().String(),
			SocksUDPListener:   udpListener,
		}
		mgr.TaskChan <- mgrTask
		socksResult = <-mgr.ResultChan
		if !socksResult.OK {
			err = errors.New("TCP conn seems disconnected!")
			return
		}

		go HandleUDPAss(mgr, component, udpListener, route, uuid, uuidNum, seq)

		succMess := &protocol.UDPAssRes{
			Seq:     seq,
			OK:      1,
			AddrLen: uint16(len(udpListener.LocalAddr().String())),
			Addr:    udpListener.LocalAddr().String(),
		}

		protocol.ConstructMessage(sMessage, header, succMess)
		sMessage.SendMessage()
	} else {
		err = errors.New("TCP conn seems disconnected!")
		return
	}
}

func HandleUDPAss(mgr *manager.Manager, component *protocol.MessageComponent, listener *net.UDPConn, route string, uuid string, uuidNum int, seq uint64) {
	sMessage := protocol.PrepareAndDecideWhichSProtoToLower(component.Conn, component.Secret, component.UUID)

	dataHeader := &protocol.Header{
		Sender:      protocol.ADMIN_UUID,
		Accepter:    uuid,
		MessageType: protocol.SOCKSUDPDATA,
		RouteLen:    uint32(len([]byte(route))),
		Route:       route,
	}

	mgrTask := &manager.ManagerTask{
		Category: manager.SOCKS,
		UUIDNum:  uuidNum,
		Seq:      seq,
		Mode:     manager.S_GETUDPDATACHAN,
	}
	mgr.TaskChan <- mgrTask
	result := <-mgr.ResultChan
	mgr.Done <- true

	if !result.OK {
		return
	}

	udpDataChan := result.UDPDataChan

	buffer := make([]byte, 20480)

	var alreadyGetAddr bool
	for {
		length, addr, err := listener.ReadFromUDP(buffer)
		if !alreadyGetAddr {
			go func() {
				for {
					if data, ok := <-udpDataChan; ok {
						listener.WriteToUDP(data, addr)
					} else {
						return
					}
				}
			}()
			alreadyGetAddr = true
		}

		if err != nil {
			listener.Close()
			return
		}

		udpDataMess := &protocol.SocksUDPData{
			Seq:     seq,
			DataLen: uint64(length),
			Data:    buffer[:length],
		}

		protocol.ConstructMessage(sMessage, dataHeader, udpDataMess)
		sMessage.SendMessage()
	}
}

func DispathTCPData(mgr *manager.Manager) {
	for {
		data := <-mgr.SocksTCPDataChan

		switch data.(type) {
		case *protocol.SocksTCPData:
			message := data.(*protocol.SocksTCPData)
			mgrTask := &manager.ManagerTask{
				Category: manager.SOCKS,
				Seq:      message.Seq,
				Mode:     manager.S_GETTCPDATACHAN_WITHOUTUUID,
			}
			mgr.TaskChan <- mgrTask
			result := <-mgr.ResultChan
			if result.OK {
				result.TCPDataChan <- message.Data
			}
			mgr.Done <- true
		case *protocol.SocksTCPFin:
			message := data.(*protocol.SocksTCPFin)
			mgrTask := &manager.ManagerTask{
				Mode:     manager.S_CLOSETCP,
				Category: manager.SOCKS,
				Seq:      message.Seq,
			}
			mgr.TaskChan <- mgrTask
		}
	}
}

func DispathUDPData(mgr *manager.Manager) {
	for {
		data := <-mgr.SocksUDPDataChan

		mgrTask := &manager.ManagerTask{
			Category: manager.SOCKS,
			Seq:      data.Seq,
			Mode:     manager.S_GETUDPDATACHAN_WITHOUTUUID,
		}
		mgr.TaskChan <- mgrTask
		result := <-mgr.ResultChan
		if result.OK {
			result.UDPDataChan <- data.Data
		}
		mgr.Done <- true
	}
}

func GetSocksInfo(mgr *manager.Manager, uuidNum int) bool {
	mgrTask := &manager.ManagerTask{
		Category: manager.SOCKS,
		UUIDNum:  uuidNum,
		Mode:     manager.S_GETSOCKSINFO,
	}
	mgr.TaskChan <- mgrTask
	result := <-mgr.ResultChan

	fmt.Print(result.SocksInfo)

	return result.OK
}

func StopSocks(component *protocol.MessageComponent, mgr *manager.Manager, route string, uuid string, uuidNum int) {
	mgrTask := &manager.ManagerTask{
		Category: manager.SOCKS,
		UUIDNum:  uuidNum,
		Mode:     manager.S_CLOSESOCKS,
	}
	mgr.TaskChan <- mgrTask
	<-mgr.ResultChan
}
