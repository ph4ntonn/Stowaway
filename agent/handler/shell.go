package handler

import (
	"Stowaway/agent/initial"
	"Stowaway/pkg/util"
	"Stowaway/protocol"
	"io"
	"os/exec"
	"runtime"

	"Stowaway/agent/manager"
	"Stowaway/global"
	"Stowaway/utils"
)

type Shell struct {
	stdin   io.Writer
	stdout  io.Reader
	charset string
}

func newShell(charset string) *Shell {
	s := new(Shell)
	s.charset = charset
	return s
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

		// 执行结果和系统编码一致，需要转换成UTF-8，这样发送给admin时才是正常的。
		result := string(buffer[:count])
		if shell.charset == "GBK" {
			result = util.ConvertGBK2Str(result)
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
	// 这里将admin输入的UNICODE字符串转换成系统编码，这样执行才不会报错
	if shell.charset == "GBK" {
		command = util.ConvertStr2GBK(command)
	}
	shell.stdin.Write([]byte(command))
}

func DispatchShellMess(mgr *manager.Manager, options *initial.Options) {
	shell := newShell(options.Charset)

	for {
		message := <-mgr.ShellManager.ShellMessChan

		switch mess := message.(type) {
		case *protocol.ShellReq:
			go shell.start()
		case *protocol.ShellCommand:
			shell.input(mess.Command)
		}
	}
}
