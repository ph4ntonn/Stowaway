package handler

import (
	"Stowaway/protocol"
	"fmt"

	"Stowaway/admin/manager"
	"Stowaway/global"
)

func LetShellStart(route string, uuid string) {
	sMessage := protocol.PrepareAndDecideWhichSProtoToLower(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)

	header := &protocol.Header{
		Sender:      protocol.ADMIN_UUID,
		Accepter:    uuid,
		MessageType: protocol.SHELLREQ,
		RouteLen:    uint32(len([]byte(route))),
		Route:       route,
	}

	shellReqMess := &protocol.ShellReq{
		Start: 1,
	}

	protocol.ConstructMessage(sMessage, header, shellReqMess, false)
	sMessage.SendMessage()
}

func DispatchShellMess(mgr *manager.Manager) {
	for {
		message := <-mgr.ShellManager.ShellMessChan

		switch mess := message.(type) {
		case *protocol.ShellRes:
			if mess.OK == 1 {
				mgr.ConsoleManager.OK <- true
			} else {
				mgr.ConsoleManager.OK <- false
			}
		case *protocol.ShellResult:
			//tmp, err := gcharset.ToUTF8("GBK", mess.Result)
			//if err == nil {
			//	mess.Result = tmp
			//}
			fmt.Print(mess.Result)
		case *protocol.ShellExit:
			mgr.ConsoleManager.Exit <- true
		}
	}
}
