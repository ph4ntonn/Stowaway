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
	// first,register a new backward,bind listener and port together
	mgrTask := &manager.BackwardTask{
		Mode:     manager.B_NEWBACKWARD,
		Listener: backward.Listener,
		RPort:    backward.Rport,
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
		// if get a conn successfully,then tell admin mission start and BTW,ask a seq num
		seqHeader := &protocol.Header{
			Sender:      component.UUID,
			Accepter:    protocol.ADMIN_UUID,
			MessageType: protocol.BACKWARDSTART,
			RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))),
			Route:       protocol.TEMP_ROUTE,
		}

		seqMess := &protocol.BackwardStart{
			UUIDLen:  uint16(len(component.UUID)),
			UUID:     component.UUID,
			LPortLen: uint16(len(backward.Lport)),
			LPort:    backward.Lport,
			RPortLen: uint16(len(backward.Rport)),
			RPort:    backward.Rport,
		}

		protocol.ConstructMessage(sMessage, seqHeader, seqMess)
		sMessage.SendMessage()
		// get the corresponding seqchan,ready to receive seq num from admin
		mgrTask = &manager.BackwardTask{
			Mode:  manager.B_GETSEQCHAN,
			RPort: backward.Rport,
		}
		mgr.BackwardManager.TaskChan <- mgrTask
		result := <-mgr.BackwardManager.ResultChan
		mgr.BackwardManager.Done <- true
		if !result.OK {
			conn.Close()
			return
		}

		backwardSeqResult, ok := <-result.SeqChan
		if !ok {
			conn.Close()
			return
		}

		// waiting for admin's gift
		defer func() {
			// tell dispatcher can go ahead
			mgr.BackwardManager.SeqReady <- true
		}()

		if backwardSeqResult.OK == 0 {
			conn.Close()
			return
		}

		seq := backwardSeqResult.Seq
		// if node get seq from seqChan successfully,then register the conn and handle the conn
		mgrTask = &manager.BackwardTask{
			Mode:           manager.B_ADDCONN,
			RPort:          backward.Rport,
			Seq:            seq,
			BackwardSocket: conn,
		}
		mgr.BackwardManager.TaskChan <- mgrTask
		result = <-mgr.BackwardManager.ResultChan
		if !result.OK {
			conn.Close()
			return
		}
		// handle it
		go backward.handleBackward(mgr, component, conn, seq)
	}
}

func (backward *Backward) handleBackward(mgr *manager.Manager, component *protocol.MessageComponent, conn net.Conn, seq uint64) {
	sMessage := protocol.PrepareAndDecideWhichSProtoToUpper(component.Conn, component.Secret, component.UUID)

	defer func() {
		finHeader := &protocol.Header{
			Sender:      component.UUID,
			Accepter:    protocol.ADMIN_UUID,
			MessageType: protocol.BACKWARDFIN,
			RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))),
			Route:       protocol.TEMP_ROUTE,
		}

		finMess := &protocol.BackWardFin{
			Seq: seq,
		}

		protocol.ConstructMessage(sMessage, finHeader, finMess)
		sMessage.SendMessage()
	}()

	// ask for corresponding datachan
	mgrTask := &manager.BackwardTask{
		Mode:  manager.B_GETDATACHAN,
		RPort: backward.Rport,
		Seq:   seq,
	}
	mgr.BackwardManager.TaskChan <- mgrTask
	result := <-mgr.BackwardManager.ResultChan
	if !result.OK {
		return
	}

	dataChan := result.DataChan
	// as same as admin
	go func() {
		for {
			if data, ok := <-dataChan; ok {
				conn.Write(data)
			} else {
				return
			}
		}
	}()
	// as same as admin,too
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
	// check if node can listen on assigned port
	listenAddr := fmt.Sprintf("0.0.0.0:%s", rPort)
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		// if can't,tell admin mission failed&&return
		protocol.ConstructMessage(sMessage, header, failMess)
		sMessage.SendMessage()
		return
	}
	// if can,than run a new backward
	backward := newBackward(listener, lPort, rPort)
	// start it
	go backward.start(mgr, component)
	// tell upstream everything's fine,just go ahead
	protocol.ConstructMessage(sMessage, header, succMess)
	sMessage.SendMessage()
}

func DispatchBackwardMess(mgr *manager.Manager, component *protocol.MessageComponent) {
	for {
		message := <-mgr.BackwardManager.BackwardMessChan
		switch message.(type) {
		case *protocol.BackwardTest:
			// at the beginning of backforward,node will get test command,so test first
			mess := message.(*protocol.BackwardTest)
			go testBackward(mgr, component, mess.LPort, mess.RPort)
		case *protocol.BackwardSeq:
			// get the seq num that admin assigned for node,put it in the corresponding chan
			mess := message.(*protocol.BackwardSeq)
			// ask for seq chan
			mgrTask := &manager.BackwardTask{
				Mode:  manager.B_GETSEQCHAN,
				RPort: mess.RPort,
				Seq:   mess.Seq,
			}
			mgr.BackwardManager.TaskChan <- mgrTask
			result := <-mgr.BackwardManager.ResultChan
			// put it
			if result.OK {
				result.SeqChan <- mess
				mgr.BackwardManager.Done <- true
				<-mgr.BackwardManager.SeqReady // make sure ADDCONN operation execute first,must before BackwardData come
			}

			mgr.BackwardManager.Done <- true
		case *protocol.BackwardData:
			mess := message.(*protocol.BackwardData)

			mgrTask := &manager.BackwardTask{
				Mode: manager.B_GETDATACHAN_WITHOUTUUID,
				Seq:  mess.Seq,
			}
			mgr.BackwardManager.TaskChan <- mgrTask
			result := <-mgr.BackwardManager.ResultChan

			if result.OK {
				result.DataChan <- mess.Data
			}
			mgr.BackwardManager.Done <- true
		case *protocol.BackWardFin:
			mess := message.(*protocol.BackWardFin)
			mgrTask := &manager.ForwardTask{
				Mode: manager.F_CLOSETCP,
				Seq:  mess.Seq,
			}
			mgr.ForwardManager.TaskChan <- mgrTask
		}
	}
}
