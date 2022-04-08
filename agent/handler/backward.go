package handler

import (
	"fmt"
	"net"

	"Stowaway/agent/manager"
	"Stowaway/global"
	"Stowaway/protocol"
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

func (backward *Backward) start(mgr *manager.Manager) {
	mgrTask := &manager.BackwardTask{
		Mode:     manager.B_NEWBACKWARD,
		Listener: backward.Listener,
		RPort:    backward.Rport,
	}

	mgr.BackwardManager.TaskChan <- mgrTask
	<-mgr.BackwardManager.ResultChan

	sMessage := protocol.PrepareAndDecideWhichSProtoToUpper(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)

	for {
		conn, err := backward.Listener.Accept()
		if err != nil {
			backward.Listener.Close() // todo:closebackward消息处理
			return
		}

		seqHeader := &protocol.Header{
			Sender:      global.G_Component.UUID,
			Accepter:    protocol.ADMIN_UUID,
			MessageType: protocol.BACKWARDSTART,
			RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))),
			Route:       protocol.TEMP_ROUTE,
		}

		seqMess := &protocol.BackwardStart{
			UUIDLen:  uint16(len(global.G_Component.UUID)),
			UUID:     global.G_Component.UUID,
			LPortLen: uint16(len(backward.Lport)),
			LPort:    backward.Lport,
			RPortLen: uint16(len(backward.Rport)),
			RPort:    backward.Rport,
		}

		protocol.ConstructMessage(sMessage, seqHeader, seqMess, false)
		sMessage.SendMessage()

		mgrTask = &manager.BackwardTask{
			Mode:  manager.B_GETSEQCHAN,
			RPort: backward.Rport,
		}
		mgr.BackwardManager.TaskChan <- mgrTask
		result := <-mgr.BackwardManager.ResultChan
		if !result.OK {
			conn.Close()
			return
		}

		seq, ok := <-result.SeqChan
		if !ok {
			conn.Close()
			return
		}

		go backward.handleBackward(mgr, conn, seq)
	}
}

func (backward *Backward) handleBackward(mgr *manager.Manager, conn net.Conn, seq uint64) {
	sMessage := protocol.PrepareAndDecideWhichSProtoToUpper(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)

	defer func() {
		finHeader := &protocol.Header{
			Sender:      global.G_Component.UUID,
			Accepter:    protocol.ADMIN_UUID,
			MessageType: protocol.BACKWARDFIN,
			RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))),
			Route:       protocol.TEMP_ROUTE,
		}

		finMess := &protocol.BackWardFin{
			Seq: seq,
		}

		protocol.ConstructMessage(sMessage, finHeader, finMess, false)
		sMessage.SendMessage()
	}()

	mgrTask := &manager.BackwardTask{
		Mode:  manager.B_ADDCONN,
		RPort: backward.Rport,
		Seq:   seq,
	}
	mgr.BackwardManager.TaskChan <- mgrTask
	result := <-mgr.BackwardManager.ResultChan
	mgr.BackwardManager.SeqReady <- true
	if !result.OK {
		conn.Close()
		return
	}

	// ask for corresponding datachan
	mgrTask = &manager.BackwardTask{
		Mode:  manager.B_GETDATACHAN,
		RPort: backward.Rport,
		Seq:   seq,
	}
	mgr.BackwardManager.TaskChan <- mgrTask
	result = <-mgr.BackwardManager.ResultChan
	if !result.OK {
		return
	}

	dataChan := result.DataChan

	go func() {
		for {
			if data, ok := <-dataChan; ok {
				conn.Write(data)
			} else {
				conn.Close()
				return
			}
		}
	}()

	dataHeader := &protocol.Header{
		Sender:      global.G_Component.UUID,
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

		protocol.ConstructMessage(sMessage, dataHeader, backwardDataMess, false)
		sMessage.SendMessage()
	}
}

func testBackward(mgr *manager.Manager, lPort, rPort string) {
	sMessage := protocol.PrepareAndDecideWhichSProtoToUpper(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)

	header := &protocol.Header{
		Sender:      global.G_Component.UUID,
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
		protocol.ConstructMessage(sMessage, header, failMess, false)
		sMessage.SendMessage()
		return
	}

	backward := newBackward(listener, lPort, rPort)

	go backward.start(mgr)

	protocol.ConstructMessage(sMessage, header, succMess, false)
	sMessage.SendMessage()
}

func sendDoneMess(all uint16, rPort string) {
	// here is a problem,if some of the backward conns cannot send FIN before DONE,then the FIN they send cannot be processed by admin
	// but it's not a really big problem,because users must know some data maybe lost since they choose to close backward
	sMessage := protocol.PrepareAndDecideWhichSProtoToUpper(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)

	header := &protocol.Header{
		Sender:      global.G_Component.UUID,
		Accepter:    protocol.ADMIN_UUID,
		MessageType: protocol.BACKWARDSTOPDONE,
		RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))),
		Route:       protocol.TEMP_ROUTE,
	}

	doneMess := &protocol.BackwardStopDone{
		All:      all,
		UUIDLen:  uint16(len(global.G_Component.UUID)),
		UUID:     global.G_Component.UUID,
		RPortLen: uint16(len(rPort)),
		RPort:    rPort,
	}

	protocol.ConstructMessage(sMessage, header, doneMess, false)
	sMessage.SendMessage()
}

func DispatchBackwardMess(mgr *manager.Manager) {
	for {
		message := <-mgr.BackwardManager.BackwardMessChan

		switch mess := message.(type) {
		case *protocol.BackwardTest:
			go testBackward(mgr, mess.LPort, mess.RPort)
		case *protocol.BackwardSeq:
			mgrTask := &manager.BackwardTask{
				Mode:  manager.B_GETSEQCHAN,
				RPort: mess.RPort,
				Seq:   mess.Seq,
			}
			mgr.BackwardManager.TaskChan <- mgrTask
			result := <-mgr.BackwardManager.ResultChan

			if result.OK {
				result.SeqChan <- mess.Seq
				<-mgr.BackwardManager.SeqReady
			} else {
				sMessage := protocol.PrepareAndDecideWhichSProtoToUpper(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)

				finHeader := &protocol.Header{
					Sender:      global.G_Component.UUID,
					Accepter:    protocol.ADMIN_UUID,
					MessageType: protocol.BACKWARDFIN,
					RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))),
					Route:       protocol.TEMP_ROUTE,
				}

				finMess := &protocol.BackWardFin{
					Seq: mess.Seq,
				}

				protocol.ConstructMessage(sMessage, finHeader, finMess, false)
				sMessage.SendMessage()
			}
		case *protocol.BackwardData:
			mgrTask := &manager.BackwardTask{
				Mode: manager.B_GETDATACHAN_WITHOUTUUID,
				Seq:  mess.Seq,
			}
			mgr.BackwardManager.TaskChan <- mgrTask
			result := <-mgr.BackwardManager.ResultChan

			if result.OK {
				result.DataChan <- mess.Data
			}
		case *protocol.BackWardFin:
			mgrTask := &manager.BackwardTask{
				Mode: manager.B_CLOSETCP,
				Seq:  mess.Seq,
			}
			mgr.BackwardManager.TaskChan <- mgrTask
		case *protocol.BackwardStop:
			if mess.All == 1 {
				mgrTask := &manager.BackwardTask{
					Mode: manager.B_CLOSESINGLEALL,
				}
				mgr.BackwardManager.TaskChan <- mgrTask
			} else {
				mgrTask := &manager.BackwardTask{
					Mode:  manager.B_CLOSESINGLE,
					RPort: mess.RPort,
				}
				mgr.BackwardManager.TaskChan <- mgrTask
			}
			<-mgr.BackwardManager.ResultChan
			go sendDoneMess(mess.All, mess.RPort)
		}
	}
}
