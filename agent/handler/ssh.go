/*
 * @Author: ph4ntom
 * @Date: 2021-03-18 18:56:20
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-03-26 16:52:47
 */

package handler

import (
	"io"

	"Stowaway/protocol"

	"golang.org/x/crypto/ssh"
)

const (
	UPMETHOD = iota
	CERMETHOD
)

type SSH struct {
	stdin           io.Writer
	stdout          io.Reader
	sshHost         *ssh.Session
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

// StartSSH 启动ssh
func (mySSH *SSH) Start(component *protocol.MessageComponent) {
	var authPayload ssh.AuthMethod
	var err error

	sMessage := protocol.PrepareAndDecideWhichSProtoToUpper(component.Conn, component.Secret, component.UUID)

	sshResheader := &protocol.Header{
		Sender:      component.UUID,
		Accepter:    protocol.ADMIN_UUID,
		MessageType: protocol.SSHRES,
		RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))), // No need to set route when agent send mess to admin
		Route:       protocol.TEMP_ROUTE,
	}

	sshResultheader := &protocol.Header{
		Sender:      component.UUID,
		Accepter:    protocol.ADMIN_UUID,
		MessageType: protocol.SSHRESULT,
		RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))), // No need to set route when agent send mess to admin
		Route:       protocol.TEMP_ROUTE,
	}

	sshResSuccMess := &protocol.SSHRes{
		OK: 1,
	}

	sshResFailMess := &protocol.SSHRes{
		OK: 0,
	}

	defer func() {
		if err != nil {
			protocol.ConstructMessage(sMessage, sshResheader, sshResFailMess)
			sMessage.SendMessage()
		}
	}()

	switch mySSH.Method {
	case UPMETHOD:
		authPayload = ssh.Password(mySSH.Password)
	case CERMETHOD:
		var key ssh.Signer
		key, err = ssh.ParsePrivateKey(mySSH.Certificate)
		if err != nil {
			return
		}
		authPayload = ssh.PublicKeys(key)
	}

	sshDial, err := ssh.Dial("tcp", mySSH.Addr, &ssh.ClientConfig{
		User:            mySSH.Username,
		Auth:            []ssh.AuthMethod{authPayload},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	})
	if err != nil {
		return
	}

	mySSH.sshHost, err = sshDial.NewSession()
	if err != nil {
		return
	}

	mySSH.stdout, err = mySSH.sshHost.StdoutPipe()
	if err != nil {
		return
	}

	mySSH.stdin, err = mySSH.sshHost.StdinPipe()
	if err != nil {
		return
	}

	mySSH.sshHost.Stderr = mySSH.sshHost.Stdout

	err = mySSH.sshHost.Shell()
	if err != nil {
		return
	}

	protocol.ConstructMessage(sMessage, sshResheader, sshResSuccMess)
	sMessage.SendMessage()

	buffer := make([]byte, 4096)
	for {
		length, err := mySSH.stdout.Read(buffer)

		if err != nil {
			break
		}

		sshResultMess := &protocol.SSHResult{
			ResultLen: uint64(length),
			Result:    string(buffer[:length]),
		}

		protocol.ConstructMessage(sMessage, sshResultheader, sshResultMess)
		sMessage.SendMessage()
	}
}

// WriteCommand 写入command
func (mySSH *SSH) Input(command string) {
	mySSH.stdin.Write([]byte(command))
}
