package agent

import (
	"Stowaway/common"
	"Stowaway/socks"
	"net"
)

func StartSocks(controlConnToAdmin net.Conn, socksPort, socksUsername, socksPass string) {
	socksstartmess, _ := common.ConstructCommand("SOCKSRESP", "SUCCESS", NODEID, AESKey)
	controlConnToAdmin.Write(socksstartmess)
}

func HanleClientSocksConn(info chan string, socksUsername, socksPass string, checknum uint32) {
	var (
		server       net.Conn
		serverflag   bool
		isAuthed     bool   = false
		method       string = ""
		tcpconnected bool   = false
	)
	for {
		if isAuthed == false && method == "" {
			data := <-info
			method = socks.CheckMethod(DataConnToAdmin, []byte(data), socksUsername, socksPass, checknum, AESKey)
			if method == "NONE" {
				isAuthed = true
			}
		} else if isAuthed == false && method == "PASSWORD" {
			data := <-info
			isAuthed = socks.AuthClient(DataConnToAdmin, []byte(data), socksUsername, socksPass, checknum, AESKey)
		} else if isAuthed == true && tcpconnected == false {
			data := <-info
			server, tcpconnected, serverflag = socks.ConfirmTarget(DataConnToAdmin, []byte(data), checknum, AESKey)
			if serverflag == false {
				return
			}
		} else if isAuthed == true && tcpconnected == true && serverflag == true {
			go func() {
				for {
					data := <-info
					_, err := server.Write([]byte(data))
					if err != nil {
						close(info)
						delete(SocksDataChanMap.SocksDataChan, checknum)
						return
					}
				}
			}()
			err := socks.Proxyhttp(DataConnToAdmin, server, checknum, AESKey)
			if err != nil {
				return
			}
		}
	}
}
