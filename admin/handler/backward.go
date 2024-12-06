package handler

import (
	"fmt"
	"net"
	"time"

	"Stowaway/admin/manager"
	"Stowaway/admin/topology"
	"Stowaway/global"
	"Stowaway/protocol"
)

type Backward struct {
	LPort string
	RPort string
}

func NewBackward(lPort, rPort string) *Backward {
	backward := new(Backward)
	backward.LPort = lPort
	backward.RPort = rPort
	return backward
}

func (backward *Backward) LetBackward(mgr *manager.Manager, route string, uuid string) error {
	sMessage := protocol.NewDownMsg(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)
	// test if node can listen on assigned port
	header := &protocol.Header{
		Sender:      protocol.ADMIN_UUID,
		Accepter:    uuid,
		MessageType: protocol.BACKWARDTEST,
		RouteLen:    uint32(len([]byte(route))),
		Route:       route,
	}

	testMess := &protocol.BackwardTest{
		LPortLen: uint16(len([]byte(backward.LPort))),
		LPort:    backward.LPort,
		RPortLen: uint16(len([]byte(backward.RPort))),
		RPort:    backward.RPort,
	}

	protocol.ConstructMessage(sMessage, header, testMess, false)
	sMessage.SendMessage()
	// node can listen on assigned port?
	if ready := <-mgr.BackwardManager.BackwardReady; !ready {
		// can't listen
		err := fmt.Errorf("fail to map remote port %s to local port %s,node cannot listen on port %s", backward.RPort, backward.LPort, backward.RPort)
		return err
	}
	// node can listen,it means no backward service is running on the assigned port,so just register a brand new backward
	backwardTask := &manager.BackwardTask{
		Mode:  manager.B_NEWBACKWARD,
		LPort: backward.LPort,
		RPort: backward.RPort,
		UUID:  uuid,
	}
	mgr.BackwardManager.TaskChan <- backwardTask
	<-mgr.BackwardManager.ResultChan
	// tell upstream all good,just go ahead
	return nil
}

func (backward *Backward) start(mgr *manager.Manager, topo *topology.Topology, uuid string) {
	sMessage := protocol.NewDownMsg(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)
	// first , admin need to know the route to target node,so ask topo for the answer
	topoTask := &topology.TopoTask{
		Mode: topology.GETROUTE,
		UUID: uuid,
	}
	topo.TaskChan <- topoTask
	topoResult := <-topo.ResultChan
	route := topoResult.Route
	// ask backward manager to assign a new seq num
	backwardTask := &manager.BackwardTask{
		Mode:  manager.B_GETNEWSEQ,
		RPort: backward.RPort,
		UUID:  uuid,
	}
	mgr.BackwardManager.TaskChan <- backwardTask
	result := <-mgr.BackwardManager.ResultChan
	seq := result.BackwardSeq

	backwardTask = &manager.BackwardTask{
		Mode:  manager.B_ADDCONN,
		RPort: backward.RPort,
		UUID:  uuid,
		Seq:   seq,
	}
	mgr.BackwardManager.TaskChan <- backwardTask
	<-mgr.BackwardManager.ResultChan

	header := &protocol.Header{
		Sender:      protocol.ADMIN_UUID,
		Accepter:    uuid,
		MessageType: protocol.BACKWARDSEQ,
		RouteLen:    uint32(len([]byte(route))),
		Route:       route,
	}

	seqMess := &protocol.BackwardSeq{
		Seq:      seq,
		RPortLen: uint16(len([]byte(backward.RPort))),
		RPort:    backward.RPort,
	}

	protocol.ConstructMessage(sMessage, header, seqMess, false)
	sMessage.SendMessage()

	// send fin after all done
	defer func() {
		finHeader := &protocol.Header{
			Sender:      protocol.ADMIN_UUID,
			Accepter:    uuid,
			MessageType: protocol.BACKWARDFIN,
			RouteLen:    uint32(len([]byte(route))),
			Route:       route,
		}

		finMess := &protocol.BackWardFin{
			Seq: seq,
		}

		protocol.ConstructMessage(sMessage, finHeader, finMess, false)
		sMessage.SendMessage()
	}()

	backwardConn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", backward.LPort), 10*time.Second)
	if err != nil {
		return
	}

	backwardTask = &manager.BackwardTask{
		Mode:  manager.B_CHECKBACKWARD,
		RPort: backward.RPort,
		UUID:  uuid,
		Seq:   seq,
	}
	mgr.BackwardManager.TaskChan <- backwardTask
	result = <-mgr.BackwardManager.ResultChan
	if !result.OK {
		backwardConn.Close()
		return
	}

	backwardTask = &manager.BackwardTask{
		Mode:  manager.B_GETDATACHAN,
		RPort: backward.RPort,
		UUID:  uuid,
		Seq:   seq,
	}
	mgr.BackwardManager.TaskChan <- backwardTask
	result = <-mgr.BackwardManager.ResultChan
	if !result.OK {
		return
	}

	dataChan := result.DataChan

	// proxy C2S
	go func() {
		for {
			if data, ok := <-dataChan; ok {
				backwardConn.Write(data)
			} else {
				backwardConn.Close()
				return
			}
		}
	}()
	// proxy S2C
	dataHeader := &protocol.Header{
		Sender:      protocol.ADMIN_UUID,
		Accepter:    uuid,
		MessageType: protocol.BACKWARDDATA,
		RouteLen:    uint32(len([]byte(route))),
		Route:       route,
	}

	buffer := make([]byte, 20480)

	for {
		length, err := backwardConn.Read(buffer)
		if err != nil {
			backwardConn.Close()
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

func GetBackwardInfo(mgr *manager.Manager, uuid string) (int, bool) {
	mgrTask := &manager.BackwardTask{
		Mode: manager.B_GETBACKWARDINFO,
		UUID: uuid,
	}
	mgr.BackwardManager.TaskChan <- mgrTask
	result := <-mgr.BackwardManager.ResultChan

	if result.OK {
		fmt.Print("\r\n[0] All")
		for _, info := range result.BackwardInfo {
			fmt.Printf(
				"\r\n[%d] Remote Port: %s , Local Port: %s , Active Connnections: %d",
				info.Seq,
				info.RPort,
				info.LPort,
				info.ActiveNum,
			)
		}
	}

	return len(result.BackwardInfo) - 1, result.OK
}

func StopBackward(mgr *manager.Manager, uuid, route string, choice int) {
	sMessage := protocol.NewDownMsg(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)

	header := &protocol.Header{
		Sender:      protocol.ADMIN_UUID,
		Accepter:    uuid,
		MessageType: protocol.BACKWARDSTOP,
		RouteLen:    uint32(len([]byte(route))),
		Route:       route,
	}

	if choice == 0 {
		stopMess := &protocol.BackwardStop{
			All: 1,
		}

		protocol.ConstructMessage(sMessage, header, stopMess, false)
		sMessage.SendMessage()
	} else {
		backwardTask := &manager.BackwardTask{
			Mode:   manager.B_GETSTOPRPORT,
			Choice: choice,
		}
		mgr.BackwardManager.TaskChan <- backwardTask
		result := <-mgr.BackwardManager.ResultChan
		stopMess := &protocol.BackwardStop{
			All:      0,
			RPortLen: uint16(len([]byte(result.RPort))),
			RPort:    result.RPort,
		}
		protocol.ConstructMessage(sMessage, header, stopMess, false)
		sMessage.SendMessage()
	}
}

func DispatchBackwardMess(mgr *manager.Manager, topo *topology.Topology) {
	for {
		message := <-mgr.BackwardManager.BackwardMessChan

		switch mess := message.(type) {
		case *protocol.BackwardReady:
			if mess.OK == 1 {
				mgr.BackwardManager.BackwardReady <- true
			} else {
				mgr.BackwardManager.BackwardReady <- false
			}
		case *protocol.BackwardStart:
			// get the start message from node,so just start a backward
			backward := NewBackward(mess.LPort, mess.RPort)
			go backward.start(mgr, topo, mess.UUID)
		case *protocol.BackwardData:
			// get node's data,just put it in the corresponding chan
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
		case *protocol.BackwardStopDone:
			if mess.All == 1 {
				backwardTask := &manager.BackwardTask{
					Mode: manager.B_CLOSESINGLEALL,
					UUID: mess.UUID,
				}
				mgr.BackwardManager.TaskChan <- backwardTask
			} else {
				backwardTask := &manager.BackwardTask{
					Mode:  manager.B_CLOSESINGLE,
					UUID:  mess.UUID,
					RPort: mess.RPort,
				}
				mgr.BackwardManager.TaskChan <- backwardTask
			}
			<-mgr.BackwardManager.ResultChan
		}
	}
}
