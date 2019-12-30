package agent

import (
	"Stowaway/common"
	"Stowaway/socks"
	"net"
	"strconv"
)

func StartSocks(controlConnToAdmin net.Conn) {
	socksstartmess, _ := common.ConstructCommand("SOCKSRESP", "SUCCESS", NODEID, AESKey)
	controlConnToAdmin.Write(socksstartmess)
}

func HanleClientSocksConn(info chan string, socksUsername, socksPass string, checknum uint32, currentid uint32) {
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
			method = socks.CheckMethod(DataConnToAdmin, []byte(data), socksUsername, socksPass, checknum, AESKey, NODEID)
			if method == "NONE" {
				isAuthed = true
			}
		} else if isAuthed == false && method == "PASSWORD" {
			data := <-info
			isAuthed = socks.AuthClient(DataConnToAdmin, []byte(data), socksUsername, socksPass, checknum, AESKey, NODEID)
		} else if isAuthed == true && tcpconnected == false {
			data := <-info
			server, tcpconnected, serverflag = socks.ConfirmTarget(DataConnToAdmin, []byte(data), checknum, AESKey, NODEID)
			if serverflag == false {
				return
			}
		} else if isAuthed == true && tcpconnected == true && serverflag == true {
			go func() {
				for {
					data := <-info
					_, err := server.Write([]byte(data))
					if err != nil {
						// close(info)
						// delete(SocksDataChanMap.SocksDataChan, checknum)
						return
					}
				}
			}()
			err := socks.Proxyhttp(DataConnToAdmin, server, checknum, AESKey, currentid)
			if err != nil {
				go SendFIN(DataConnToAdmin, checknum)
				return
			}
		}
	}
}

func SendFIN(conn net.Conn, num uint32) {
	nodeid := strconv.Itoa(int(NODEID))
	respData, _ := common.ConstructDataResult(0, num, " ", "FIN", nodeid, AESKey, 0)
	conn.Write(respData)
}
