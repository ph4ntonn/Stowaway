package handler

import (
	"Stowaway/agent/manager"
	"Stowaway/protocol"
	"Stowaway/share"
)

func DispatchFileMess(mgr *manager.Manager, component *protocol.MessageComponent) {
	for {
		message := <-mgr.FileManager.FileMessChan

		switch message.(type) {
		case *protocol.FileStatReq:
			mess := message.(*protocol.FileStatReq)
			mgr.FileManager.File.FileName = mess.Filename
			mgr.FileManager.File.SliceNum = mess.SliceNum
			err := mgr.FileManager.File.CheckFileStat(component, protocol.TEMP_ROUTE, protocol.ADMIN_UUID, share.AGENT)
			if err == nil {
				go mgr.FileManager.File.Receive(component, protocol.TEMP_ROUTE, protocol.ADMIN_UUID, share.AGENT)
			}
		case *protocol.FileStatRes:
			mess := message.(*protocol.FileStatRes)
			if mess.OK == 1 {
				go mgr.FileManager.File.Upload(component, protocol.TEMP_ROUTE, protocol.ADMIN_UUID, share.AGENT)
			} else {
				mgr.FileManager.File.Handler.Close()
			}
		case *protocol.FileDownReq:
			mess := message.(*protocol.FileDownReq)
			mgr.FileManager.File.FilePath = mess.FilePath
			mgr.FileManager.File.FileName = mess.Filename
			go mgr.FileManager.File.SendFileStat(component, protocol.TEMP_ROUTE, protocol.ADMIN_UUID, share.AGENT)
		case *protocol.FileData:
			mess := message.(*protocol.FileData)
			mgr.FileManager.File.DataChan <- mess.Data
		case *protocol.FileErr:
			mgr.FileManager.File.ErrChan <- true
		}
	}
}
