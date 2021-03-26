/*
 * @Author: ph4ntom
 * @Date: 2021-03-19 18:40:13
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-03-26 16:49:05
 */
package handler

import (
	"Stowaway/admin/manager"
	"Stowaway/protocol"
	"errors"
	"fmt"
	"net"
)

type Socks struct {
	Username string
	Password string
	Port     int
}

func NewSocks() *Socks {
	return new(Socks)
}

func (socks *Socks) LetSocks(component *protocol.MessageComponent, mgr *manager.Manager, route string, nodeID string, idNum int) error {
	socksAddr := fmt.Sprintf("0.0.0.0:%s", socks.Port)
	socksListener, err := net.Listen("tcp", socksAddr)
	if err != nil {
		return err
	}
	//把此监听地址记录
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
	if !result.OK {
		socksListener.Close()
		return errors.New("Socks service has already running on this node!")
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

	for {
		conn, err := socksListener.Accept()
		if err != nil {
			return err
		}

		task := &manager.ManagerTask{
			Category: manager.SOCKS,
			Mode:     manager.S_GETNEWSEQ,
			UUIDNum:  idNum,
		}
		mgr.TaskChan <- task
		result := <-mgr.SocksResultChan
		seq := result.SocksID

		//有请求时记录此socket，并启动HandleNewSocksConn对此socket进行处理
		task = &manager.ManagerTask{
			Category:       manager.SOCKS,
			UUIDNum:        idNum,
			Seq:            seq,
			Mode:           manager.S_ADDTCPSOCKET,
			SocksTCPSocket: conn,
		}
		mgr.TaskChan <- task
		<-mgr.SocksResultChan

		go socks.handleSocks(component, mgr, conn, route, nodeID, idNum, seq)
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

	// handle received data
	go func() {
		for {
			if data, ok := <-tcpDataChan; ok {
				conn.Write(data)
			} else {
				return
			}
		}
	}()

	// handle sended data
	buffer := make([]byte, 20480)
	for {
		length, err := conn.Read(buffer)
		if err != nil {
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
