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

func StartSSH(controlConnToAdmin net.Conn, info string, nodeid uint32) {
	spiltedinfo := strings.Split(info, "::")
	host := spiltedinfo[0]
	username := spiltedinfo[1]
	password := spiltedinfo[2]

	sshdial, err := ssh.Dial("tcp", host, &ssh.ClientConfig{
		User:            username,
		Auth:            []ssh.AuthMethod{ssh.Password(password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	})
	Sshhost, err = sshdial.NewSession()

	if err != nil {
		sshMess, _ := common.ConstructCommand("SSHRESP", "FAILED", nodeid)
		controlConnToAdmin.Write(sshMess)
		return
	}
	Stdout, err = Sshhost.StdoutPipe()
	if err != nil {
		fmt.Println(err)
	}
	Stdin, err = Sshhost.StdinPipe()
	if err != nil {
		fmt.Println(err)
	}
	Sshhost.Stderr = Sshhost.Stdout
	Sshhost.Shell()
	sshMess, _ := common.ConstructCommand("SSHRESP", "SUCCESS", nodeid)
	controlConnToAdmin.Write(sshMess)
	if err != nil {
		fmt.Println(err)
	}
}

func WriteCommand(command string) {
	fmt.Println("write in", command)
	Stdin.Write([]byte(command))
}

func ReadCommand() {
	buffer := make([]byte, 40960)
	for {
		len, err := Stdout.Read(buffer)
		if err != nil {
			fmt.Println("err from sshhost")
			break
		}
		sshRespMess, _ := common.ConstructDataResult(0, "1", "SSHMESS", string(buffer[:len]))
		cmdResult <- sshRespMess
	}
}
