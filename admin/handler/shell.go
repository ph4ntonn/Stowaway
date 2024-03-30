package handler

import (
	"fmt"

	"Stowaway/admin/manager"
	"Stowaway/global"
	"Stowaway/protocol"
)

func LetShellStart(route string, uuid string) {
	sMessage := protocol.NewDownMsg(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)

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
			fmt.Print(mess.Result)
		case *protocol.ShellExit:
			mgr.ConsoleManager.Exit <- true
		}
	}
}
