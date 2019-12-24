package agent

import (
	"Stowaway/common"
	"fmt"
	"io"
	"os/exec"
	"runtime"
)

func CreatInteractiveShell() (*exec.Cmd, io.Reader, io.Writer) {
	cmd := exec.Command("/bin/sh", "-i")
	if runtime.GOARCH == "386" || runtime.GOARCH == "amd64" {
		cmd = exec.Command("/bin/bash", "-i")
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Println(err)
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		fmt.Println(err)
	}
	cmd.Stderr = cmd.Stdout //将stderr连接至stdout
	cmd.Start()
	return cmd, stdout, stdin
}

func StartShell(command string, cmd *exec.Cmd, stdin io.Writer, stdout io.Reader, currentid uint32) {
	success := "1"
	dataType := "SHELLRESP"

	buf := make([]byte, 1024)
	stdin.Write([]byte(command))
	for {
		count, err := stdout.Read(buf)
		if err != nil {
			fmt.Println("error: ", err)
			return
		}
		respShell, err := common.ConstructDataResult(0, 0, success, dataType, string(buf[:count]), AESKey, currentid)
		cmdResult <- respShell
	}
}
