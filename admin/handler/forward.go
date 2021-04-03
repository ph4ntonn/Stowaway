/*
 * @Author: ph4ntom
 * @Date: 2021-04-02 13:22:25
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-04-03 16:01:31
 */
package handler

import (
	"Stowaway/admin/manager"
	"Stowaway/protocol"
	"fmt"
	"net"
)

type Forward struct {
	UUIDNum int
	Port    string
}

func NewForward(uuidNum int, port string) *Forward {
	forward := new(Forward)
	forward.UUIDNum = uuidNum
	forward.Port = port
	return forward
}

func LetForward(component *protocol.MessageComponent, mgr *manager.Manager, port string, addr string, route string, uuid string, uuidNum int) error {
	listenAddr := fmt.Sprintf("0.0.0.0:%s", port)
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return err
	}

	sMessage := protocol.PrepareAndDecideWhichSProtoToLower(component.Conn, component.Secret, component.UUID)

	header := &protocol.Header{
		Sender:      protocol.ADMIN_UUID,
		Accepter:    uuid,
		MessageType: protocol.FORWARDSTART,
		RouteLen:    uint32(len([]byte(route))),
		Route:       route,
	}

	startMess := &protocol.ForwardStart{
		AddrLen: uint16(len([]byte(addr))),
		Addr:    addr,
	}

	protocol.ConstructMessage(sMessage, header, startMess)
	sMessage.SendMessage()

	if ready := <-mgr.ForwardManager.ForwardReady; !ready {
		listener.Close()
		err := fmt.Errorf("[*]Fail to forward port %s to remote addr %s", port, addr)
		return err
	}

	mgrTask := &manager.ForwardTask{
		Mode:     manager.F_NEWFORWARD,
		UUIDNum:  uuidNum,
		Listener: listener,
		Port:     port,
	}

	mgr.ForwardManager.TaskChan <- mgrTask
	<-mgr.ForwardManager.ResultChan

	go handleForwardListener(component, mgr, listener, port, route, uuid, uuidNum)

	return nil
}

func handleForwardListener(component *protocol.MessageComponent, mgr *manager.Manager, listener net.Listener, port string, route string, uuid string, uuidNum int) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			listener.Close()
			return
		}

		// ask new seq num
		mgrTask := &manager.ForwardTask{
			Mode:    manager.F_GETNEWSEQ,
			UUIDNum: uuidNum,
		}
		mgr.ForwardManager.TaskChan <- mgrTask
		result := <-mgr.ForwardManager.ResultChan
		seq := result.ForwardSeq

		// save the socket
		mgrTask = &manager.ForwardTask{
			Mode:    manager.F_ADDCONN,
			UUIDNum: uuidNum,
			Seq:     seq,
			Port:    port,
			Conn:    conn,
		}
		mgr.ForwardManager.TaskChan <- mgrTask
		result = <-mgr.ForwardManager.ResultChan
		if !result.OK {
			return
		}

		// handle it!
		// go socks.handleSocks(component, mgr, conn, route, uuid, uuidNum, seq)
	}
}
