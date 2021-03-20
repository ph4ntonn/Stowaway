/*
 * @Author: ph4ntom
 * @Date: 2021-03-19 12:24:52
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-03-20 13:36:10
 */
package handler

import (
	"Stowaway/protocol"
	"Stowaway/utils"
	"io/ioutil"
)

const (
	UPMETHOD = iota
	CERMETHOD
)

type SSH struct {
	Method          int
	Addr            string
	Username        string
	Password        string
	CertificatePath string
	Certificate     []byte
}

func NewSSH() *SSH {
	return new(SSH)
}

func (ssh *SSH) LetSSH(component *protocol.MessageComponent, route string, nodeID string) error {
	_, _, err := utils.CheckIPPort(ssh.Addr)
	if err != nil {
		return err
	}

	if ssh.Method == CERMETHOD {
		if err := ssh.getCertificate(); err != nil {
			return err
		}
	}

	sMessage := protocol.PrepareAndDecideWhichSProto(component.Conn, component.Secret, component.UUID)

	header := protocol.Header{
		Sender:      protocol.ADMIN_UUID,
		Accepter:    nodeID,
		MessageType: protocol.SSHREQ,
		RouteLen:    uint32(len([]byte(route))),
		Route:       route,
	}

	sshReqMess := protocol.SSHReq{
		Method:         uint16(ssh.Method),
		AddrLen:        uint64(len(ssh.Addr)),
		Addr:           ssh.Addr,
		UsernameLen:    uint64(len(ssh.Username)),
		Username:       ssh.Username,
		PasswordLen:    uint64(len(ssh.Password)),
		Password:       ssh.Password,
		CertificateLen: uint64(len(ssh.Certificate)),
		Certificate:    ssh.Certificate,
	}

	protocol.ConstructMessage(sMessage, header, sshReqMess)
	sMessage.SendMessage()

	return nil
}

func (ssh *SSH) getCertificate() (err error) {
	ssh.Certificate, err = ioutil.ReadFile(ssh.CertificatePath)
	if err != nil {
		return
	}
	return
}
