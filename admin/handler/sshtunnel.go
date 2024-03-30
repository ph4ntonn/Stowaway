package handler

import (
	"io/ioutil"

	"Stowaway/admin/manager"
	"Stowaway/global"
	"Stowaway/protocol"
	"Stowaway/utils"
)

type SSHTunnel struct {
	Method          int
	Addr            string
	Port            string
	Username        string
	Password        string
	CertificatePath string
	Certificate     []byte
}

func NewSSHTunnel(port, addr string) *SSHTunnel {
	sshTunnel := new(SSHTunnel)
	sshTunnel.Port = port
	sshTunnel.Addr = addr
	return sshTunnel
}

func (sshTunnel *SSHTunnel) LetSSHTunnel(route, uuid string) error {
	_, _, err := utils.CheckIPPort(sshTunnel.Addr)
	if err != nil {
		return err
	}

	if sshTunnel.Method == CERMETHOD {
		if err := sshTunnel.getCertificate(); err != nil {
			return err
		}
	}

	sMessage := protocol.NewDownMsg(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)

	header := &protocol.Header{
		Sender:      protocol.ADMIN_UUID,
		Accepter:    uuid,
		MessageType: protocol.SSHTUNNELREQ,
		RouteLen:    uint32(len([]byte(route))),
		Route:       route,
	}

	sshTunnelReqMess := &protocol.SSHTunnelReq{
		Method:         uint16(sshTunnel.Method),
		AddrLen:        uint16(len(sshTunnel.Addr)),
		Addr:           sshTunnel.Addr,
		PortLen:        uint16(len(sshTunnel.Port)),
		Port:           sshTunnel.Port,
		UsernameLen:    uint64(len(sshTunnel.Username)),
		Username:       sshTunnel.Username,
		PasswordLen:    uint64(len(sshTunnel.Password)),
		Password:       sshTunnel.Password,
		CertificateLen: uint64(len(sshTunnel.Certificate)),
		Certificate:    sshTunnel.Certificate,
	}

	protocol.ConstructMessage(sMessage, header, sshTunnelReqMess, false)
	sMessage.SendMessage()

	return nil
}

func (sshTunnel *SSHTunnel) getCertificate() (err error) {
	sshTunnel.Certificate, err = ioutil.ReadFile(sshTunnel.CertificatePath)
	if err != nil {
		return
	}
	return
}

func DispatchSSHTunnelMess(mgr *manager.Manager) {
	for {
		message := <-mgr.SSHTunnelManager.SSHTunnelMessChan

		switch mess := message.(type) {
		case *protocol.SSHTunnelRes:
			if mess.OK == 1 {
				mgr.ConsoleManager.OK <- true
			} else {
				mgr.ConsoleManager.OK <- false
			}
		}
	}
}
