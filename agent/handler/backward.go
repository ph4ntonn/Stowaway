package handler

import (
	"Stowaway/agent/manager"
	"Stowaway/protocol"
	"fmt"
	"net"
)

type Backward struct {
	Lport    string
	Rport    string
	Listener net.Listener
}

func newBackward(listener net.Listener, lPort, rPort string) *Backward {
	backward := new(Backward)
	backward.Listener = listener
	backward.Lport = lPort
	backward.Rport = rPort
	return backward
}

func (backward *Backward) start(mgr *manager.Manager, component *protocol.MessageComponent) {
	mgrTask := &manager.BackwardTask{
		Mode:     manager.B_NEWBACKWARD,
		Listener: backward.Listener,
		Port:     backward.Rport,
	}

	mgr.BackwardManager.TaskChan <- mgrTask
	<-mgr.BackwardManager.ResultChan

	sMessage := protocol.PrepareAndDecideWhichSProtoToUpper(component.Conn, component.Secret, component.UUID)

	for {
		conn, err := backward.Listener.Accept()
		if err != nil {
			backward.Listener.Close() // todo:closebackward消息处理
			return
		}

		seqHeader := &protocol.Header{
			Sender:      component.UUID,
			Accepter:    protocol.ADMIN_UUID,
			MessageType: protocol.BACKWARDSEQREQ,
			RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))),
			Route:       protocol.TEMP_ROUTE,
		}

		seqMess := protocol.BackwardSeqReq{
			UUIDLen:  uint16(len(component.UUID)),
			UUID:     component.UUID,
			LPortLen: uint16(len(backward.Lport)),
			LPort:    backward.Lport,
			RPortLen: uint16(len(backward.Rport)),
			RPort:    backward.Rport,
		}

		protocol.ConstructMessage(sMessage, seqHeader, seqMess)
		sMessage.SendMessage()

		mgrTask = &manager.BackwardTask{
			Mode: manager.B_GETSEQCHAN,
			Port: backward.Rport,
		}
		mgr.BackwardManager.TaskChan <- mgrTask
		result := <-mgr.BackwardManager.ResultChan
		mgr.BackwardManager.Done <- true
		if !result.OK {
			return
		}

		seq, ok := <-result.SeqChan
		if !ok {
			return
		}

		mgrTask = &manager.BackwardTask{
			Mode:           manager.B_ADDCONN,
			Port:           backward.Rport,
			Seq:            seq,
			BackwardSocket: conn,
		}
		mgr.BackwardManager.TaskChan <- mgrTask
		result = <-mgr.BackwardManager.ResultChan
		if !result.OK {
			return
		}

		go backward.handleBackward(mgr, component, conn, seq)
	}
}

func (backward *Backward) handleBackward(mgr *manager.Manager, component *protocol.MessageComponent, conn net.Conn, seq uint64) {
	sMessage := protocol.PrepareAndDecideWhichSProtoToUpper(component.Conn, component.Secret, component.UUID)

	mgrTask := &manager.BackwardTask{
		Mode: manager.B_GETDATACHAN,
		Port: backward.Rport,
		Seq:  seq,
	}
	mgr.BackwardManager.TaskChan <- mgrTask
	result := <-mgr.BackwardManager.ResultChan
	if !result.OK {
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
		MessageType: protocol.BACKWARDDATA,
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

		backwardDataMess := &protocol.BackwardData{
			Seq:     seq,
			DataLen: uint64(length),
			Data:    buffer[:length],
		}

		protocol.ConstructMessage(sMessage, dataHeader, backwardDataMess)
		sMessage.SendMessage()
	}
}

func testBackward(mgr *manager.Manager, component *protocol.MessageComponent, lPort, rPort string) {
	sMessage := protocol.PrepareAndDecideWhichSProtoToUpper(component.Conn, component.Secret, component.UUID)

	header := &protocol.Header{
		Sender:      component.UUID,
		Accepter:    protocol.ADMIN_UUID,
		MessageType: protocol.BACKWARDREADY,
		RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))),
		Route:       protocol.TEMP_ROUTE,
	}

	succMess := &protocol.BackwardReady{
		OK: 1,
	}

	failMess := &protocol.BackwardReady{
		OK: 0,
	}

	listenAddr := fmt.Sprintf("0.0.0.0:%s", rPort)
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		protocol.ConstructMessage(sMessage, header, failMess)
		sMessage.SendMessage()
		return
	}

	backward := newBackward(listener, lPort, rPort)

	go backward.start(mgr, component)

	protocol.ConstructMessage(sMessage, header, succMess)
	sMessage.SendMessage()
}

func DispatchBackwardMess(mgr *manager.Manager, component *protocol.MessageComponent) {
	for {
		message := <-mgr.BackwardManager.BackwardMessChan
		switch message.(type) {
		case *protocol.BackwardTest:
			mess := message.(*protocol.BackwardTest)
			go testBackward(mgr, component, mess.LPort, mess.RPort)
		}
	}
}
