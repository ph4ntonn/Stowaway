package handler

import (
	"fmt"
	"net"

	"Stowaway/admin/manager"
	"Stowaway/global"
	"Stowaway/protocol"
)

type Forward struct {
	Addr string
	Port string
}

func NewForward(port, addr string) *Forward {
	forward := new(Forward)
	forward.Port = port
	forward.Addr = addr
	return forward
}

func (forward *Forward) LetForward(mgr *manager.Manager, route string, uuid string) error {
	listenAddr := fmt.Sprintf("0.0.0.0:%s", forward.Port)
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return err
	}

	sMessage := protocol.PrepareAndDecideWhichSProtoToLower(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)

	header := &protocol.Header{
		Sender:      protocol.ADMIN_UUID,
		Accepter:    uuid,
		MessageType: protocol.FORWARDTEST,
		RouteLen:    uint32(len([]byte(route))),
		Route:       route,
	}

	testMess := &protocol.ForwardTest{
		AddrLen: uint16(len([]byte(forward.Addr))),
		Addr:    forward.Addr,
	}

	protocol.ConstructMessage(sMessage, header, testMess, false)
	sMessage.SendMessage()

	if ready := <-mgr.ForwardManager.ForwardReady; !ready {
		listener.Close()
		err := fmt.Errorf("fail to forward port %s to remote addr %s,remote addr is not responding", forward.Port, forward.Addr)
		return err
	}

	mgrTask := &manager.ForwardTask{
		Mode:       manager.F_NEWFORWARD,
		UUID:       uuid,
		Listener:   listener,
		Port:       forward.Port,
		RemoteAddr: forward.Addr,
	}

	mgr.ForwardManager.TaskChan <- mgrTask
	<-mgr.ForwardManager.ResultChan

	go forward.handleForwardListener(mgr, listener, route, uuid)

	return nil
}

func (forward *Forward) handleForwardListener(mgr *manager.Manager, listener net.Listener, route string, uuid string) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			listener.Close() // todo:map没有释放
			return
		}

		mgrTask := &manager.ForwardTask{
			Mode: manager.F_GETNEWSEQ,
			UUID: uuid,
			Port: forward.Port,
		}
		mgr.ForwardManager.TaskChan <- mgrTask
		result := <-mgr.ForwardManager.ResultChan
		seq := result.ForwardSeq

		mgrTask = &manager.ForwardTask{
			Mode: manager.F_ADDCONN,
			UUID: uuid,
			Seq:  seq,
			Port: forward.Port,
		}
		mgr.ForwardManager.TaskChan <- mgrTask
		result = <-mgr.ForwardManager.ResultChan
		if !result.OK {
			conn.Close()
			return
		}

		go forward.handleForward(mgr, conn, route, uuid, seq)
	}
}

func (forward *Forward) handleForward(mgr *manager.Manager, conn net.Conn, route string, uuid string, seq uint64) {
	sMessage := protocol.PrepareAndDecideWhichSProtoToLower(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)
	// tell agent to start
	startHeader := &protocol.Header{
		Sender:      protocol.ADMIN_UUID,
		Accepter:    uuid,
		MessageType: protocol.FORWARDSTART,
		RouteLen:    uint32(len([]byte(route))),
		Route:       route,
	}

	startMess := &protocol.ForwardStart{
		Seq:     seq,
		AddrLen: uint16(len([]byte(forward.Addr))),
		Addr:    forward.Addr,
	}

	// Seems strange hh
	// but the reason why i split this into two part (line 139 and line 168)
	// is that if i want to use "Done chan" to make sure "stopforward" won't accidentally close the datachan that line 267 get(which can lead to panic)
	// i must make sure that "Done chan" won't block the forwardManager first
	// think about this scenario：when admin send startMess to agent,and agent send data back just before line 140 "F_GETDATACHAN" done
	// what will happen?
	// since i use "Done chan" to block the forwardManager, so i must do done<-true to make forwardManager go ahead
	// but sadly, if i want send true to done,i need to send forward data to corresponding chan
	// and at the same time "F_GETDATACHAN" is trying to achieve forwardManager's response(which will never response because "Done chan" is blocking it)
	// the DEAD LOCK happened
	// so to avoid DEAD LOCK, i split task and result to make sure agent must send data after admin get all things done
	mgrTask := &manager.ForwardTask{
		Mode: manager.F_GETDATACHAN,
		UUID: uuid,
		Seq:  seq,
		Port: forward.Port,
	}
	mgr.ForwardManager.TaskChan <- mgrTask
	result := <-mgr.ForwardManager.ResultChan

	protocol.ConstructMessage(sMessage, startHeader, startMess, false)
	sMessage.SendMessage()

	defer func() {
		finHeader := &protocol.Header{
			Sender:      protocol.ADMIN_UUID,
			Accepter:    uuid,
			MessageType: protocol.FORWARDFIN,
			RouteLen:    uint32(len([]byte(route))),
			Route:       route,
		}

		finMess := &protocol.ForwardFin{
			Seq: seq,
		}

		protocol.ConstructMessage(sMessage, finHeader, finMess, false)
		sMessage.SendMessage()
	}()

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
		Sender:      protocol.ADMIN_UUID,
		Accepter:    uuid,
		MessageType: protocol.FORWARDDATA,
		RouteLen:    uint32(len([]byte(route))),
		Route:       route,
	}

	buffer := make([]byte, 20480)

	for {
		length, err := conn.Read(buffer)
		if err != nil {
			conn.Close()
			return
		}

		forwardDataMess := &protocol.ForwardData{
			Seq:     seq,
			DataLen: uint64(length),
			Data:    buffer[:length],
		}

		protocol.ConstructMessage(sMessage, dataHeader, forwardDataMess, false)
		sMessage.SendMessage()
	}
}

func GetForwardInfo(mgr *manager.Manager, uuid string) (int, bool) {
	mgrTask := &manager.ForwardTask{
		Mode: manager.F_GETFORWARDINFO,
		UUID: uuid,
	}
	mgr.ForwardManager.TaskChan <- mgrTask
	result := <-mgr.ForwardManager.ResultChan

	for _, info := range result.ForwardInfo {
		fmt.Print(info)
	}

	return len(result.ForwardInfo) - 1, result.OK
}

func StopForward(mgr *manager.Manager, uuid string, choice int) {
	if choice == 0 {
		mgrTask := &manager.ForwardTask{
			Mode: manager.F_CLOSESINGLEALL,
			UUID: uuid,
		}
		mgr.ForwardManager.TaskChan <- mgrTask
		<-mgr.ForwardManager.ResultChan
	} else {
		mgrTask := &manager.ForwardTask{
			Mode:        manager.F_CLOSESINGLE,
			UUID:        uuid,
			CloseTarget: choice,
		}
		mgr.ForwardManager.TaskChan <- mgrTask
		<-mgr.ForwardManager.ResultChan
	}
}

func DispatchForwardMess(mgr *manager.Manager) {
	for {
		message := <-mgr.ForwardManager.ForwardMessChan

		switch mess := message.(type) {
		case *protocol.ForwardReady:
			if mess.OK == 1 {
				mgr.ForwardManager.ForwardReady <- true
			} else {
				mgr.ForwardManager.ForwardReady <- false
			}
		case *protocol.ForwardData:
			mgrTask := &manager.ForwardTask{
				Mode: manager.F_GETDATACHAN_WITHOUTUUID,
				Seq:  mess.Seq,
			}
			mgr.ForwardManager.TaskChan <- mgrTask
			result := <-mgr.ForwardManager.ResultChan
			if result.OK {
				result.DataChan <- mess.Data
			}
			mgr.ForwardManager.Done <- true
		case *protocol.ForwardFin:
			mgrTask := &manager.ForwardTask{
				Mode: manager.F_CLOSETCP,
				Seq:  mess.Seq,
			}
			mgr.ForwardManager.TaskChan <- mgrTask
		}
	}
}
