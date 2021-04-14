/*
 * @Author: ph4ntom
 * @Date: 2021-04-02 14:22:02
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-04-02 17:39:46
 */
package handler

import (
	"Stowaway/agent/manager"
	"Stowaway/protocol"
	"net"
	"time"
)

type Forward struct {
	Seq  uint64
	Addr string
}

func newForward(seq uint64, addr string) *Forward {
	forward := new(Forward)
	forward.Seq = seq
	forward.Addr = addr
	return forward
}

func (forward *Forward) start(mgr *manager.Manager, component *protocol.MessageComponent) {
	sMessage := protocol.PrepareAndDecideWhichSProtoToUpper(component.Conn, component.Secret, component.UUID)

	finHeader := &protocol.Header{
		Sender:      component.UUID,
		Accepter:    protocol.ADMIN_UUID,
		MessageType: protocol.FORWARDFIN,
		RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))),
		Route:       protocol.TEMP_ROUTE,
	}

	finMess := &protocol.ForwardFin{
		Seq: forward.Seq,
	}

	defer func() {
		protocol.ConstructMessage(sMessage, finHeader, finMess)
		sMessage.SendMessage()
	}()

	conn, err := net.DialTimeout("tcp", forward.Addr, 10*time.Second)
	if err != nil {
		return
	}

	task := &manager.ForwardTask{
		Mode:          manager.F_UPDATEFORWARD,
		Seq:           forward.Seq,
		ForwardSocket: conn,
	}
	mgr.ForwardManager.TaskChan <- task
	result := <-mgr.ForwardManager.ResultChan
	if !result.OK {
		conn.Close()
		return
	}

	task = &manager.ForwardTask{
		Mode: manager.F_GETDATACHAN,
		Seq:  forward.Seq,
	}
	mgr.ForwardManager.TaskChan <- task
	result = <-mgr.ForwardManager.ResultChan
	if !result.OK { // no need to close conn,cuz conn has been already recorded,so if FIN occur between F_UPDATEFORWARD and F_GETDATACHAN,closeTCP will help us to close the conn
		return
	}

	dataChan := result.DataChan

	go func() {
		for {
			if data, ok := <-dataChan; ok {
				conn.Write(data)
			} else {
				return
			}
		}
	}()

	dataHeader := &protocol.Header{
		Sender:      component.UUID,
		Accepter:    protocol.ADMIN_UUID,
		MessageType: protocol.FORWARDDATA,
		RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))),
		Route:       protocol.TEMP_ROUTE,
	}

	buffer := make([]byte, 20480)

	for {
		length, err := conn.Read(buffer)
		if err != nil {
			conn.Close()
			return
		}

		forwardDataMess := &protocol.ForwardData{
			Seq:     forward.Seq,
			DataLen: uint64(length),
			Data:    buffer[:length],
		}

		protocol.ConstructMessage(sMessage, dataHeader, forwardDataMess)
		sMessage.SendMessage()
	}
}

func testForward(component *protocol.MessageComponent, addr string) {
	sMessage := protocol.PrepareAndDecideWhichSProtoToUpper(component.Conn, component.Secret, component.UUID)

	header := &protocol.Header{
		Sender:      component.UUID,
		Accepter:    protocol.ADMIN_UUID,
		MessageType: protocol.FORWARDREADY,
		RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))),
		Route:       protocol.TEMP_ROUTE,
	}

	succMess := &protocol.ForwardReady{
		OK: 1,
	}

	failMess := &protocol.ForwardReady{
		OK: 0,
	}

	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		protocol.ConstructMessage(sMessage, header, failMess)
		sMessage.SendMessage()
		return
	}

	conn.Close()

	protocol.ConstructMessage(sMessage, header, succMess)
	sMessage.SendMessage()
}

func DispatchForwardMess(mgr *manager.Manager, component *protocol.MessageComponent) {
	for {
		message := <-mgr.ForwardManager.ForwardMessChan
		switch message.(type) {
		case *protocol.ForwardTest:
			mess := message.(*protocol.ForwardTest)
			go testForward(component, mess.Addr)
		case *protocol.ForwardStart:
			mess := message.(*protocol.ForwardStart)
			task := &manager.ForwardTask{
				Mode: manager.F_NEWFORWARD,
				Seq:  mess.Seq,
			}
			mgr.ForwardManager.TaskChan <- task
			<-mgr.ForwardManager.ResultChan
			forward := newForward(mess.Seq, mess.Addr)
			go forward.start(mgr, component)
		case *protocol.ForwardData:
			mess := message.(*protocol.ForwardData)
			mgrTask := &manager.ForwardTask{
				Mode: manager.F_GETDATACHAN,
				Seq:  mess.Seq,
			}
			mgr.ForwardManager.TaskChan <- mgrTask
			result := <-mgr.ForwardManager.ResultChan
			if result.OK {
				result.DataChan <- mess.Data
			}
		case *protocol.ForwardFin:
			mess := message.(*protocol.ForwardFin)
			mgrTask := &manager.ForwardTask{
				Mode: manager.F_CLOSETCP,
				Seq:  mess.Seq,
			}
			mgr.ForwardManager.TaskChan <- mgrTask
		}
	}
}
