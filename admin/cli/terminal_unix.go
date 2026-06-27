//go:build !windows
// +build !windows

package cli

import (
	"errors"
	"os"
	"os/signal"
	"syscall"
	"time"

	"Stowaway/admin/handler"
	"Stowaway/admin/printer"

	"github.com/nsf/termbox-go"
	"golang.org/x/sys/unix"
)

func (console *Console) handleTerminalPanelCommand(route string, uuid string) {
	session := handler.NewTerminalSession()
	drainConsoleExit(console)
	handler.SetActiveTerminalSession(session)
	waiter := handler.RegisterTerminalWaiter(session)
	finish := func() {
		handler.ClearActiveTerminalSession(session)
		handler.UnregisterTerminalWaiter(session)
		console.ready <- true
	}

	cols, rows := currentTerminalSize()
	handler.LetTerminalStart(route, uuid, session, cols, rows)

	ok, timedOut := waiter.WaitReady(10 * time.Second)
	if timedOut {
		printer.Fail("\r\n[*] Terminal start timed out!")
		handler.SendTerminalExit(route, uuid, session)
		finish()
		return
	}
	if !ok {
		handler.ClearActiveTerminalSession(session)
		handler.UnregisterTerminalWaiter(session)
		console.fallbackLegacyShell(route, uuid)
		return
	}

	termbox.Close()

	restore, err := makeRaw(int(os.Stdin.Fd()))
	if err != nil {
		printer.Fail("\r\n[*] Unable to switch local terminal to raw mode: %s", err.Error())
		handler.SendTerminalExit(route, uuid, session)
		restoreTermbox()
		finish()
		return
	}

	done := make(chan struct{})
	localExit := make(chan struct{}, 1)
	go forwardLocalInput(route, uuid, session, done, localExit)
	go forwardTerminalResize(route, uuid, session, done)

	select {
	case <-waiter.Exit:
	case <-localExit:
		handler.SendTerminalExit(route, uuid, session)
	}
	close(done)
	restore()
	restoreTermbox()
	finish()
}

func forwardLocalInput(route string, uuid string, session uint64, done <-chan struct{}, localExit chan<- struct{}) {
	buf := make([]byte, 4096)
	for {
		select {
		case <-done:
			return
		default:
		}

		n, err := os.Stdin.Read(buf)
		if n > 0 {
			data := append([]byte(nil), buf[:n]...)
			if pos := controlBackslashIndex(data); pos >= 0 {
				if pos > 0 {
					handler.SendTerminalData(route, uuid, session, data[:pos])
				}
				select {
				case localExit <- struct{}{}:
				default:
				}
				return
			}
			handler.SendTerminalData(route, uuid, session, data)
		}
		if errors.Is(err, unix.EAGAIN) || errors.Is(err, unix.EWOULDBLOCK) {
			time.Sleep(10 * time.Millisecond)
			continue
		}
		if err != nil {
			select {
			case localExit <- struct{}{}:
			default:
			}
			return
		}
	}
}

func forwardTerminalResize(route string, uuid string, session uint64, done <-chan struct{}) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGWINCH)
	defer signal.Stop(sigCh)

	cols, rows := currentTerminalSize()
	handler.SendTerminalResize(route, uuid, session, cols, rows)
	for {
		select {
		case <-done:
			return
		case <-sigCh:
			cols, rows := currentTerminalSize()
			handler.SendTerminalResize(route, uuid, session, cols, rows)
		}
	}
}

func controlBackslashIndex(data []byte) int {
	for i, b := range data {
		if b == 0x1c {
			return i
		}
	}
	return -1
}

func makeRaw(fd int) (func(), error) {
	oldState, err := unix.IoctlGetTermios(fd, ioctlReadTermios())
	if err != nil {
		return nil, err
	}
	oldFlags, err := unix.FcntlInt(uintptr(fd), unix.F_GETFL, 0)
	if err != nil {
		return nil, err
	}

	newState := *oldState
	newState.Iflag &^= unix.BRKINT | unix.ICRNL | unix.INPCK | unix.ISTRIP | unix.IXON
	newState.Oflag &^= unix.OPOST
	newState.Cflag |= unix.CS8
	newState.Lflag &^= unix.ECHO | unix.ICANON | unix.IEXTEN | unix.ISIG
	newState.Cc[unix.VMIN] = 1
	newState.Cc[unix.VTIME] = 0

	if err := unix.IoctlSetTermios(fd, ioctlWriteTermios(), &newState); err != nil {
		return nil, err
	}
	if _, err := unix.FcntlInt(uintptr(fd), unix.F_SETFL, oldFlags|unix.O_NONBLOCK); err != nil {
		_ = unix.IoctlSetTermios(fd, ioctlWriteTermios(), oldState)
		return nil, err
	}

	return func() {
		_, _ = unix.FcntlInt(uintptr(fd), unix.F_SETFL, oldFlags)
		_ = unix.IoctlSetTermios(fd, ioctlWriteTermios(), oldState)
	}, nil
}

func currentTerminalSize() (uint16, uint16) {
	ws, err := unix.IoctlGetWinsize(int(os.Stdout.Fd()), unix.TIOCGWINSZ)
	if err != nil || ws.Col == 0 || ws.Row == 0 {
		return 80, 24
	}
	return ws.Col, ws.Row
}

func restoreTermbox() {
	if err := termbox.Init(); err != nil {
		return
	}
	termbox.SetCursor(0, 0)
	termbox.Flush()
}
