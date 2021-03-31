/*
 * @Author: ph4ntom
 * @Date: 2021-03-19 18:40:13
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-03-31 16:31:30
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

func (socks *Socks) LetSocks(component *protocol.MessageComponent, mgr *manager.Manager, route string, uuid string, uuidNum int) {
	socksAddr := fmt.Sprintf("0.0.0.0:%s", socks.Port)
	socksListener, err := net.Listen("tcp", socksAddr)
	if err != nil {
		fmt.Printf("\r\n[*]Error: %s", err.Error())
		return
	}

	// register brand new socks service
	mgrTask := &manager.ManagerTask{
		Category:         manager.SOCKS,
		Mode:             manager.S_NEWSOCKS,
		UUIDNum:          uuidNum,
		SocksPort:        socks.Port,
		SocksUsername:    socks.Username,
		SocksPassword:    socks.Password,
		SocksTCPListener: socksListener,
	}

	mgr.TaskChan <- mgrTask
	result := <-mgr.SocksResultChan // wait for "add" done
	if !result.OK {                 // node and socks service must be one-to-one
		socksListener.Close()
		fmt.Printf("\r\n[*]Error: Socks service has already running on this node!")
		return
	}

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

	// run a dispatcher to dispatch all socks TCP/UDP data
	go socks.dispathTCPData(mgr)
	go socks.dispathUDPData(mgr)

	for {
		conn, err := socksListener.Accept()
		if err != nil {
			socksListener.Close()
			fmt.Printf("\r\n[*]Error: %s", err.Error())
			return
		}

		// ask new seq num
		mgrTask := &manager.ManagerTask{
			Category: manager.SOCKS,
			Mode:     manager.S_GETNEWSEQ,
			UUIDNum:  uuidNum,
		}
		mgr.TaskChan <- mgrTask
		result := <-mgr.SocksResultChan
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
		<-mgr.SocksResultChan

		// handle it!
		go socks.handleSocks(component, mgr, conn, route, uuid, uuidNum, seq)
	}
}

func (socks *Socks) dispathTCPData(mgr *manager.Manager) {
	for {
		data, ok := <-mgr.SocksTCPDataChan
		if ok {
			mgrTask := &manager.ManagerTask{
				Category: manager.SOCKS,
				Seq:      data.Seq,
				Mode:     manager.S_GETTCPDATACHAN_WITHOUTUUID,
			}
			mgr.TaskChan <- mgrTask
			result := <-mgr.SocksResultChan
			if result.OK {
				result.TCPDataChan <- data.Data
			}
		} else {
			return
		}
	}
}

func (socks *Socks) dispathUDPData(mgr *manager.Manager) {
	for {
		data, ok := <-mgr.SocksUDPDataChan
		if ok {
			mgrTask := &manager.ManagerTask{
				Category: manager.SOCKS,
				Seq:      data.Seq,
				Mode:     manager.S_GETUDPDATACHAN_WITHOUTUUID,
			}
			mgr.TaskChan <- mgrTask
			result := <-mgr.SocksResultChan
			if result.OK {
				result.UDPDataChan <- data.Data
			}
		} else {
			return
		}
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
	result := <-mgr.SocksResultChan

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
			HandleTCPFin(mgr, seq)
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

func HandleTCPFin(mgr *manager.Manager, seq uint64) {
	mgrTask := &manager.ManagerTask{
		Mode:     manager.S_CLOSETCP,
		Category: manager.SOCKS,
		Seq:      seq,
	}
	mgr.TaskChan <- mgrTask
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
	socksResult := <-mgr.SocksResultChan
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
		socksResult = <-mgr.SocksResultChan
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

	// finHeader := &protocol.Header{
	// 	Sender:      protocol.ADMIN_UUID,
	// 	Accepter:    uuid,
	// 	MessageType: protocol.SOCKSUDPFIN,
	// 	RouteLen:    uint32(len([]byte(route))),
	// 	Route:       route,
	// }

	// finMess := &protocol.SocksUDPFin{
	// 	Seq: seq,
	// }

	mgrTask := &manager.ManagerTask{
		Category: manager.SOCKS,
		UUIDNum:  uuidNum,
		Seq:      seq,
		Mode:     manager.S_GETUDPDATACHAN,
	}
	mgr.TaskChan <- mgrTask
	result := <-mgr.SocksResultChan

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
			fmt.Print(err.Error())
			// add udpfin
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
