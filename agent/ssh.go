package agent

import (
	"Stowaway/common"
	"fmt"
	"io"
	"net"
	"strings"

	"golang.org/x/crypto/ssh"
)

var Stdin io.Writer
var Stdout io.Reader
var Sshhost *ssh.Session

func StartSSH(controlConnToAdmin net.Conn, info string, nodeid uint32) error {
	spiltedinfo := strings.Split(info, "::")
	host := spiltedinfo[0]
	username := spiltedinfo[1]
	password := spiltedinfo[2]

	sshdial, err := ssh.Dial("tcp", host, &ssh.ClientConfig{
		User:            username,
		Auth:            []ssh.AuthMethod{ssh.Password(password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	})
	if err != nil {
		sshMess, _ := common.ConstructCommand("SSHRESP", "FAILED", nodeid, AESKey)
		controlConnToAdmin.Write(sshMess)
		return err
	}
	Sshhost, err = sshdial.NewSession()

	if err != nil {
		sshMess, _ := common.ConstructCommand("SSHRESP", "FAILED", nodeid, AESKey)
		controlConnToAdmin.Write(sshMess)
		return err
	}
	Stdout, err = Sshhost.StdoutPipe()
	if err != nil {
		fmt.Println(err)
		return err
	}
	Stdin, err = Sshhost.StdinPipe()
	if err != nil {
		fmt.Println(err)
		return err
	}
	Sshhost.Stderr = Sshhost.Stdout
	Sshhost.Shell()
	sshMess, _ := common.ConstructCommand("SSHRESP", "SUCCESS", nodeid, AESKey)
	controlConnToAdmin.Write(sshMess)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

func WriteCommand(command string) {
	Stdin.Write([]byte(command))
}

func ReadCommand() {
	buffer := make([]byte, 40960)
	for {
		len, err := Stdout.Read(buffer)
		if err != nil {
			break
		}
		sshRespMess, _ := common.ConstructDataResult(0, 0, "1", "SSHMESS", string(buffer[:len]), AESKey)
		cmdResult <- sshRespMess
	}
}
