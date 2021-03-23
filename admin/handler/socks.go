/*
 * @Author: ph4ntom
 * @Date: 2021-03-19 18:40:13
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-03-23 19:23:30
 */
package handler

import (
	"Stowaway/protocol"
	"Stowaway/share"
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

func (socks *Socks) LetSocks(component *protocol.MessageComponent, mgr *share.Manager, route string, nodeID string, idNum int) error {
	socksAddr := fmt.Sprintf("0.0.0.0:%s", socks.Port)
	socksListener, err := net.Listen("tcp", socksAddr)
	if err != nil {
		return err
	}
	//把此监听地址记录
	task := &share.ManagerTask{
		Category:      share.SOCKS,
		Mode:          share.NEWSOCKS,
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

	header := protocol.Header{
		Sender:      protocol.ADMIN_UUID,
		Accepter:    nodeID,
		MessageType: protocol.SOCKSSTART,
		RouteLen:    uint32(len([]byte(route))),
		Route:       route,
	}

	socksStartMess := protocol.SocksStart{
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
		//有请求时记录此socket，并启动HandleNewSocksConn对此socket进行处理
		task := &share.ManagerTask{
			Category:    share.SOCKS,
			Mode:        share.ADDSOCKSSOCKET,
			SocksSocket: conn,
		}
		mgr.TaskChan <- task
		<-mgr.SocksResultChan

		go socks.handleSocks(component, mgr, conn, route, nodeID)
	}
}

func (socks *Socks) handleSocks(component *protocol.MessageComponent, mgr *share.Manager, conn net.Conn, route string, nodeID string) {
	sMessage := protocol.PrepareAndDecideWhichSProtoToLower(component.Conn, component.Secret, component.UUID)

	header := protocol.Header{
		Sender:      protocol.ADMIN_UUID,
		Accepter:    nodeID,
		MessageType: protocol.SOCKSDATA,
		RouteLen:    uint32(len([]byte(route))),
		Route:       route,
	}

	task := &share.ManagerTask{
		Category: share.SOCKS,
		Mode:     share.GETSOCKSID,
	}
	mgr.TaskChan <- task
	result := <-mgr.SocksResultChan
	id := result.SocksID

	buffer := make([]byte, 20480)
	for {
		length, err := conn.Read(buffer)
		if err != nil {
			return
		}

		socksDataMess := protocol.SocksData{
			ID:      id,
			DataLen: uint64(length),
			Data:    buffer[:length],
		}

		protocol.ConstructMessage(sMessage, header, socksDataMess)
		sMessage.SendMessage()
	}
}
