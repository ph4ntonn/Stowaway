package handler

import (
	"errors"
	"fmt"
	"time"

	"Stowaway/agent/manager"
	"Stowaway/global"
	"Stowaway/protocol"
	"Stowaway/share"

	"golang.org/x/crypto/ssh"
)

type SSHTunnel struct {
	Method      int
	Addr        string
	Port        string
	Username    string
	Password    string
	Certificate []byte
}

func newSSHTunnel(method int, addr, port, username, password string, certificate []byte) *SSHTunnel {
	sshTunnel := new(SSHTunnel)
	sshTunnel.Method = method
	sshTunnel.Addr = addr
	sshTunnel.Port = port
	sshTunnel.Username = username
	sshTunnel.Password = password
	sshTunnel.Certificate = certificate
	return sshTunnel
}

func (sshTunnel *SSHTunnel) start(mgr *manager.Manager) {
	var authPayload ssh.AuthMethod
	var err error
	var sUMessage, sLMessage, rMessage protocol.Message

	sUMessage = protocol.NewUpMsg(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)

	sshTunnelResheader := &protocol.Header{
		Sender:      global.G_Component.UUID,
		Accepter:    protocol.ADMIN_UUID,
		MessageType: protocol.SSHTUNNELRES,
		RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))), // No need to set route when agent send mess to admin
		Route:       protocol.TEMP_ROUTE,
	}

	sshTunnelResSuccMess := &protocol.SSHTunnelRes{
		OK: 1,
	}

	sshTunnelResFailMess := &protocol.SSHTunnelRes{
		OK: 0,
	}

	defer func() {
		if err != nil {
			protocol.ConstructMessage(sUMessage, sshTunnelResheader, sshTunnelResFailMess, false)
			sUMessage.SendMessage()
		}
	}()

	switch sshTunnel.Method {
	case UPMETHOD:
		authPayload = ssh.Password(sshTunnel.Password)
	case CERMETHOD:
		var key ssh.Signer
		key, err = ssh.ParsePrivateKey(sshTunnel.Certificate)
		if err != nil {
			return
		}
		authPayload = ssh.PublicKeys(key)
	}

	sshDial, err := ssh.Dial("tcp", sshTunnel.Addr, &ssh.ClientConfig{
		User:            sshTunnel.Username,
		Auth:            []ssh.AuthMethod{authPayload},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	})
	if err != nil {
		return
	}

	conn, err := sshDial.Dial("tcp", fmt.Sprintf("127.0.0.1:%s", sshTunnel.Port))
	if err != nil {
		return
	}

	if err = share.ActivePreAuth(conn); err != nil {
		return
	}

	sLMessage = protocol.NewDownMsg(conn, global.G_Component.Secret, protocol.ADMIN_UUID)

	hiHeader := &protocol.Header{
		Sender:      protocol.ADMIN_UUID, // fake admin
		Accepter:    protocol.TEMP_UUID,
		MessageType: protocol.HI,
		RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))),
		Route:       protocol.TEMP_ROUTE,
	}

	// fake admin
	hiMess := &protocol.HIMess{
		GreetingLen: uint16(len("Shhh...")),
		Greeting:    "Shhh...",
		UUIDLen:     uint16(len(protocol.ADMIN_UUID)),
		UUID:        protocol.ADMIN_UUID,
		IsAdmin:     1,
		IsReconnect: 0,
	}

	protocol.ConstructMessage(sLMessage, hiHeader, hiMess, false)
	sLMessage.SendMessage()

	rMessage = protocol.NewDownMsg(conn, global.G_Component.Secret, protocol.ADMIN_UUID)
	fHeader, fMessage, err := protocol.DestructMessage(rMessage)
	if err != nil {
		conn.Close()
		return
	}

	if fHeader.MessageType == protocol.HI {
		mmess := fMessage.(*protocol.HIMess)
		if mmess.Greeting == "Keep slient" && mmess.IsAdmin == 0 {
			childIP := conn.RemoteAddr().String()

			cUUIDReqHeader := &protocol.Header{
				Sender:      global.G_Component.UUID,
				Accepter:    protocol.ADMIN_UUID,
				MessageType: protocol.CHILDUUIDREQ,
				RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))),
				Route:       protocol.TEMP_ROUTE,
			}

			cUUIDMess := &protocol.ChildUUIDReq{
				ParentUUIDLen: uint16(len(global.G_Component.UUID)),
				ParentUUID:    global.G_Component.UUID,
				IPLen:         uint16(len(childIP)),
				IP:            childIP,
			}

			protocol.ConstructMessage(sUMessage, cUUIDReqHeader, cUUIDMess, false)
			sUMessage.SendMessage()

			childUUID := <-mgr.ListenManager.ChildUUIDChan

			uuidHeader := &protocol.Header{
				Sender:      protocol.ADMIN_UUID,
				Accepter:    protocol.TEMP_UUID,
				MessageType: protocol.UUID,
				RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))),
				Route:       protocol.TEMP_ROUTE,
			}

			uuidMess := &protocol.UUIDMess{
				UUIDLen: uint16(len(childUUID)),
				UUID:    childUUID,
			}

			protocol.ConstructMessage(sLMessage, uuidHeader, uuidMess, false)
			sLMessage.SendMessage()

			childrenTask := &manager.ChildrenTask{
				Mode: manager.C_NEWCHILD,
				UUID: childUUID,
				Conn: conn,
			}
			mgr.ChildrenManager.TaskChan <- childrenTask
			<-mgr.ChildrenManager.ResultChan

			mgr.ChildrenManager.ChildComeChan <- &manager.ChildInfo{UUID: childUUID, Conn: conn}

			protocol.ConstructMessage(sUMessage, sshTunnelResheader, sshTunnelResSuccMess, false)
			sUMessage.SendMessage()

			return
		}
	}

	conn.Close()
	err = errors.New("node seems illegal")
}

func DispatchSSHTunnelMess(mgr *manager.Manager) {
	for {
		message := <-mgr.SSHTunnelManager.SSHTunnelMessChan

		switch mess := message.(type) {
		case *protocol.SSHTunnelReq:
			sshTunnel := newSSHTunnel(int(mess.Method), mess.Addr, mess.Port, mess.Username, mess.Password, mess.Certificate)
			go sshTunnel.start(mgr)
		}
	}
}
