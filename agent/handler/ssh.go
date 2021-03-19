/*
 * @Author: ph4ntom
 * @Date: 2021-03-18 18:56:20
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-03-19 17:42:28
 */

package handler

import (
	"fmt"
	"io"
	"net"

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
func (mySSH *SSH) Start(conn net.Conn, nodeID string, secret string) {
	var authPayload ssh.AuthMethod

	sMessage := protocol.PrepareAndDecideWhichSProto(conn, secret, nodeID)

	sshResheader := protocol.Header{
		Sender:      nodeID,
		Accepter:    protocol.ADMIN_UUID,
		MessageType: protocol.SSHRES,
		RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))), // No need to set route when agent send mess to admin
		Route:       protocol.TEMP_ROUTE,
	}

	sshResultheader := protocol.Header{
		Sender:      nodeID,
		Accepter:    protocol.ADMIN_UUID,
		MessageType: protocol.SSHRESULT,
		RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))), // No need to set route when agent send mess to admin
		Route:       protocol.TEMP_ROUTE,
	}

	sshResSuccMess := protocol.SSHRes{
		OK: 1,
	}

	sshResFailMess := protocol.SSHRes{
		OK: 0,
	}

	switch mySSH.Method {
	case UPMETHOD:
		authPayload = ssh.Password(mySSH.Password)
	case CERMETHOD:
		key, err := ssh.ParsePrivateKey(mySSH.Certificate)
		if err != nil {
			protocol.ConstructMessage(sMessage, sshResheader, sshResFailMess)
			sMessage.SendMessage()
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
		protocol.ConstructMessage(sMessage, sshResheader, sshResFailMess)
		sMessage.SendMessage()
		return
	}

	mySSH.sshHost, err = sshDial.NewSession()
	if err != nil {
		fmt.Println(err.Error())
		protocol.ConstructMessage(sMessage, sshResheader, sshResFailMess)
		sMessage.SendMessage()
		return
	}

	mySSH.stdout, err = mySSH.sshHost.StdoutPipe()
	if err != nil {
		fmt.Println(err.Error())
		protocol.ConstructMessage(sMessage, sshResheader, sshResFailMess)
		sMessage.SendMessage()
		return
	}

	mySSH.stdin, err = mySSH.sshHost.StdinPipe()
	if err != nil {
		fmt.Println(err.Error())
		protocol.ConstructMessage(sMessage, sshResheader, sshResFailMess)
		sMessage.SendMessage()
		return
	}

	mySSH.sshHost.Stderr = mySSH.sshHost.Stdout

	err = mySSH.sshHost.Shell()
	if err != nil {
		fmt.Println(err.Error())
		protocol.ConstructMessage(sMessage, sshResheader, sshResFailMess)
		sMessage.SendMessage()
		return
	}

	protocol.ConstructMessage(sMessage, sshResheader, sshResSuccMess)
	sMessage.SendMessage()

	buffer := make([]byte, 20480)
	for {
		length, err := mySSH.stdout.Read(buffer)
		if err != nil {
			break
		}

		sshResultMess := protocol.SSHResult{
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
