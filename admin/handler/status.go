package handler

import (
	"Stowaway/admin/manager"
	"fmt"
)

func ShowStatus(mgr *manager.Manager, uuid string) {
	forwardTask := &manager.ForwardTask{
		Mode: manager.F_GETFORWARDINFO,
		UUID: uuid,
	}
	mgr.ForwardManager.TaskChan <- forwardTask
	forwardResult := <-mgr.ForwardManager.ResultChan

	backwardTask := &manager.BackwardTask{
		Mode: manager.B_GETBACKWARDINFO,
		UUID: uuid,
	}
	mgr.BackwardManager.TaskChan <- backwardTask
	backwardResult := <-mgr.BackwardManager.ResultChan

	socksTask := &manager.SocksTask{
		Mode: manager.S_GETSOCKSINFO,
		UUID: uuid,
	}
	mgr.SocksManager.TaskChan <- socksTask
	socksResult := <-mgr.SocksManager.ResultChan
	// show socks
	fmt.Print("\r\nSocks status:")
	if socksResult.OK {
		fmt.Printf(
			"\r\n      ListenAddr: %s:%s    Username: %s   Password: %s",
			socksResult.SocksInfo.Addr,
			socksResult.SocksInfo.Port,
			socksResult.SocksInfo.Username,
			socksResult.SocksInfo.Password,
		)
	}
	fmt.Print("\r\n-------------------------------------------------------------------------------------------")
	// show forward
	fmt.Print("\r\nForward status:")
	if forwardResult.OK {
		for _, info := range forwardResult.ForwardInfo {
			fmt.Printf(
				"\r\n      [%d] Listening Addr: %s , Remote Addr: %s , Active Connnections: %d",
				info.Seq,
				info.Laddr,
				info.Raddr,
				info.ActiveNum,
			)
		}
	}
	fmt.Print("\r\n-------------------------------------------------------------------------------------------")
	// show backward
	fmt.Print("\r\nBackward status:")
	if backwardResult.OK {
		for _, info := range backwardResult.BackwardInfo {
			fmt.Printf(
				"\r\n      [%d] Remote Port: %s , Local Port: %s , Active Connnections: %d",
				info.Seq,
				info.RPort,
				info.LPort,
				info.ActiveNum,
			)
		}
	}
}
