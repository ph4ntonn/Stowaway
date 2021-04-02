/*
 * @Author: ph4ntom
 * @Date: 2021-04-02 14:22:02
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-04-02 17:18:31
 */
package handler

import (
	"Stowaway/agent/manager"
	"Stowaway/protocol"
	"net"
	"time"
)

func DispatchForwardData(mgr *manager.Manager) {
	for {
		forwardData := <-mgr.ForwardManager.ForwardDataChan
		switch forwardData.(type) {

		}
	}
}

func StartForward(mgr *manager.Manager, component *protocol.MessageComponent, addr string, seq uint64) {
	sMessage := protocol.PrepareAndDecideWhichSProtoToUpper(component.Conn, component.Secret, component.UUID)

	header := &protocol.Header{
		Sender:      component.UUID,
		Accepter:    protocol.ADMIN_UUID,
		MessageType: protocol.FORWARDREADY,
		RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))), // No need to set route when agent send mess to admin
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

	mgrTask := &manager.ForwardTask{
		Mode: manager.F_NEWFORWARD,
		Seq:  seq,
	}
	mgr.ForwardManager.TaskChan <- mgrTask
	<-mgr.ForwardManager.ResultChan

	conn.Close()

	protocol.ConstructMessage(sMessage, header, succMess)
	sMessage.SendMessage()
}
