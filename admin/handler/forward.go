/*
 * @Author: ph4ntom
 * @Date: 2021-04-02 13:22:25
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-04-02 16:48:04
 */
package handler

import (
	"Stowaway/admin/manager"
	"Stowaway/protocol"
	"fmt"
	"net"
)

func LetForward(component *protocol.MessageComponent, mgr *manager.Manager, port string, addr string, route string, uuid string, uuidNum int) error {
	sMessage := protocol.PrepareAndDecideWhichSProtoToLower(component.Conn, component.Secret, component.UUID)

	header := &protocol.Header{
		Sender:      protocol.ADMIN_UUID,
		Accepter:    uuid,
		MessageType: protocol.FORWARDSTART,
		RouteLen:    uint32(len([]byte(route))),
		Route:       route,
	}

	listenAddr := fmt.Sprintf("0.0.0.0:%s", port)
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return err
	}

	mgrTask := &manager.ForwardTask{
		Mode:    manager.F_GETNEWSEQ,
		UUIDNum: uuidNum,
	}
	mgr.ForwardManager.TaskChan <- mgrTask
	result := <-mgr.ForwardManager.ResultChan

	seq := result.ForwardSeq

	startMess := &protocol.ForwardStart{
		Seq:     seq,
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

	return nil
}
