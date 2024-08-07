package handler

import (
	"io"
	"time"

	"Stowaway/agent/manager"
	"Stowaway/global"
	"Stowaway/protocol"
	"Stowaway/utils"

	"golang.org/x/crypto/ssh"
)

const (
	UPMETHOD = iota
	CERMETHOD
)

type SSH struct {
	stdin       io.Writer
	stdout      io.Reader
	sshHost     *ssh.Session
	Method      int
	Addr        string
	Username    string
	Password    string
	Certificate []byte
}

func newSSH() *SSH {
	return new(SSH)
}

// StartSSH 启动ssh
func (mySSH *SSH) start() {
	var authPayload ssh.AuthMethod
	var err error

	sMessage := protocol.NewUpMsg(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)

	sshResheader := &protocol.Header{
		Sender:      global.G_Component.UUID,
		Accepter:    protocol.ADMIN_UUID,
		MessageType: protocol.SSHRES,
		RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))), // No need to set route when agent send mess to admin
		Route:       protocol.TEMP_ROUTE,
	}

	sshResultheader := &protocol.Header{
		Sender:      global.G_Component.UUID,
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
			protocol.ConstructMessage(sMessage, sshResheader, sshResFailMess, false)
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
		Timeout:         10 * time.Second,
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

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	var term string

	switch utils.CheckSystem() {
	case 0x01:
		term = ""
	case 0x02:
		term = "linux"
	case 0x03:
		term = "xterm"
	}

	err = mySSH.sshHost.RequestPty(term, 25, 80, modes)
	if err != nil {
		return
	}

	err = mySSH.sshHost.Shell()
	if err != nil {
		return
	}

	protocol.ConstructMessage(sMessage, sshResheader, sshResSuccMess, false)
	sMessage.SendMessage()

	sshExitHeader := &protocol.Header{
		Sender:      global.G_Component.UUID,
		Accepter:    protocol.ADMIN_UUID,
		MessageType: protocol.SSHEXIT,
		RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))), // No need to set route when agent send mess to admin
		Route:       protocol.TEMP_ROUTE,
	}

	sshExitMess := &protocol.SSHExit{
		OK: 1,
	}

	buffer := make([]byte, 4096)
	for {
		length, err := mySSH.stdout.Read(buffer)

		if err != nil {
			protocol.ConstructMessage(sMessage, sshExitHeader, sshExitMess, false)
			sMessage.SendMessage()
			return
		}

		sshResultMess := &protocol.SSHResult{
			ResultLen: uint64(length),
			Result:    string(buffer[:length]),
		}

		protocol.ConstructMessage(sMessage, sshResultheader, sshResultMess, false)
		sMessage.SendMessage()
	}
}

// WriteCommand 写入command
func (mySSH *SSH) input(command string) {
	mySSH.stdin.Write([]byte(command))
}

func DispatchSSHMess(mgr *manager.Manager) {
	var mySSH *SSH

	for {
		message := <-mgr.SSHManager.SSHMessChan

		switch mess := message.(type) {
		case *protocol.SSHReq:
			mySSH = newSSH()
			mySSH.Addr = mess.Addr
			mySSH.Method = int(mess.Method)
			mySSH.Username = mess.Username
			mySSH.Password = mess.Password
			mySSH.Certificate = mess.Certificate
			go mySSH.start()
		case *protocol.SSHCommand:
			mySSH.input(mess.Command)
		}
	}
}
