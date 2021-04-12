package handler

import (
	"Stowaway/admin/manager"
	"Stowaway/protocol"
	"fmt"
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

func (backward *Backward) LetBackward(component *protocol.MessageComponent, mgr *manager.Manager, route string, uuid string) error {
	sMessage := protocol.PrepareAndDecideWhichSProtoToLower(component.Conn, component.Secret, component.UUID)

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

	protocol.ConstructMessage(sMessage, header, testMess)
	sMessage.SendMessage()

	if ready := <-mgr.BackwardManager.BackwardReady; !ready {
		err := fmt.Errorf("Fail to map remote port %s to local port %s,node cannot listen on port %s", backward.RPort, backward.LPort, backward.RPort)
		return err
	}

	return nil
}

func DispatchBackwardMess(mgr *manager.Manager) {
	for {
		message := <-mgr.BackwardManager.BackwardMessChan

		switch message.(type) {
		case *protocol.BackwardReady:
			mess := message.(*protocol.BackwardReady)
			if mess.OK == 1 {
				mgr.BackwardManager.BackwardReady <- true
			} else {
				mgr.BackwardManager.BackwardReady <- false
			}
		}
	}
}
