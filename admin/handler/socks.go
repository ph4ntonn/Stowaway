/*
 * @Author: ph4ntom
 * @Date: 2021-03-19 18:40:13
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-03-30 12:26:29
 */
package handler

import (
	"Stowaway/admin/manager"
	"Stowaway/protocol"
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

func (socks *Socks) LetSocks(component *protocol.MessageComponent, mgr *manager.Manager, route string, nodeID string, idNum int) {
	socksAddr := fmt.Sprintf("0.0.0.0:%s", socks.Port)
	socksListener, err := net.Listen("tcp", socksAddr)
	if err != nil {
		fmt.Printf("\r\n[*]Error: %s", err.Error())
		return
	}

	// register brand new socks service
	task := &manager.ManagerTask{
		Category:      manager.SOCKS,
		Mode:          manager.S_NEWSOCKS,
		UUIDNum:       idNum,
		SocksPort:     socks.Port,
		SocksUsername: socks.Username,
		SocksPassword: socks.Password,
	}

	mgr.TaskChan <- task
	result := <-mgr.SocksResultChan // wait for "add" done
	if !result.OK {                 // node and socks service must be one-to-one
		socksListener.Close()
		fmt.Printf("\r\n[*]Error: Socks service has already running on this node!")
		return
	}

	sMessage := protocol.PrepareAndDecideWhichSProtoToLower(component.Conn, component.Secret, component.UUID)

	header := &protocol.Header{
		Sender:      protocol.ADMIN_UUID,
		Accepter:    nodeID,
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

	// run a dispatcher to dispatch all socks TCP data
	go socks.dispathTCPData(mgr)

	for {
		conn, err := socksListener.Accept()
		if err != nil {
			socksListener.Close()
			fmt.Printf("\r\n[*]Error: %s", err.Error())
			return
		}

		// ask new seq num
		task := &manager.ManagerTask{
			Category: manager.SOCKS,
			Mode:     manager.S_GETNEWSEQ,
			UUIDNum:  idNum,
		}
		mgr.TaskChan <- task
		result := <-mgr.SocksResultChan
		seq := result.SocksID

		// save the socket
		task = &manager.ManagerTask{
			Category:       manager.SOCKS,
			UUIDNum:        idNum,
			Seq:            seq,
			Mode:           manager.S_ADDTCPSOCKET,
			SocksTCPSocket: conn,
		}
		mgr.TaskChan <- task
		<-mgr.SocksResultChan

		// handle it!
		go socks.handleSocks(component, mgr, conn, route, nodeID, idNum, seq)
	}
}

func (socks *Socks) dispathTCPData(mgr *manager.Manager) {
	for {
		data, ok := <-mgr.SocksTCPDataChan
		if ok {
			task := &manager.ManagerTask{
				Category: manager.SOCKS,
				Seq:      data.Seq,
				Mode:     manager.S_GETTCPDATACHAN_WITHOUTUUID,
			}
			mgr.TaskChan <- task
			result := <-mgr.SocksResultChan
			if result.OK {
				result.TCPDataChan <- data.Data
			}
		} else {
			return
		}
	}
}

func (socks *Socks) handleSocks(component *protocol.MessageComponent, mgr *manager.Manager, conn net.Conn, route string, nodeID string, idNum int, seq uint64) {
	sMessage := protocol.PrepareAndDecideWhichSProtoToLower(component.Conn, component.Secret, component.UUID)

	header := &protocol.Header{
		Sender:      protocol.ADMIN_UUID,
		Accepter:    nodeID,
		MessageType: protocol.SOCKSTCPDATA,
		RouteLen:    uint32(len([]byte(route))),
		Route:       route,
	}

	task := &manager.ManagerTask{
		Category: manager.SOCKS,
		UUIDNum:  idNum,
		Seq:      seq,
		Mode:     manager.S_GETTCPDATACHAN,
	}
	mgr.TaskChan <- task
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
			Accepter:    nodeID,
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
	task := &manager.ManagerTask{
		Mode:     manager.S_CLOSETCP,
		Category: manager.SOCKS,
		Seq:      seq,
	}
	mgr.TaskChan <- task
}
