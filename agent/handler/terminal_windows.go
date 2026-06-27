//go:build windows
// +build windows

package handler

import (
	"errors"
	"os"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"unsafe"

	"Stowaway/agent/manager"
	"Stowaway/global"
	"Stowaway/protocol"

	"golang.org/x/sys/windows"
)

const (
	procThreadAttributePseudoConsole = 0x00020016
)

var (
	kernel32                          = syscall.NewLazyDLL("kernel32.dll")
	procCreatePseudoConsole           = kernel32.NewProc("CreatePseudoConsole")
	procResizePseudoConsole           = kernel32.NewProc("ResizePseudoConsole")
	procClosePseudoConsole            = kernel32.NewProc("ClosePseudoConsole")
	procInitializeProcThreadAttrList  = kernel32.NewProc("InitializeProcThreadAttributeList")
	procUpdateProcThreadAttribute     = kernel32.NewProc("UpdateProcThreadAttribute")
	procDeleteProcThreadAttributeList = kernel32.NewProc("DeleteProcThreadAttributeList")
)

type coord struct {
	X int16
	Y int16
}

type startupInfoEx struct {
	windows.StartupInfo
	AttributeList *byte
}

type windowsTerminal struct {
	session    uint64
	hpc        uintptr
	inputWrite windows.Handle
	outputRead windows.Handle
	process    windows.ProcessInformation
	sendMu     sync.Mutex
	writeMu    sync.Mutex
	closeMu    sync.Mutex
	closed     bool
}

func newWindowsTerminal(session uint64) *windowsTerminal {
	return &windowsTerminal{session: session}
}

func (terminal *windowsTerminal) start(cols, rows uint16) {
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

	send := func(header *protocol.Header, mess interface{}) {
		terminal.sendMu.Lock()
		defer terminal.sendMu.Unlock()
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
			terminal.close()
		}
	}()

	err = terminal.open(cols, rows)
	if err != nil {
		return
	}
	if terminal.isClosed() {
		terminal.close()
		return
	}

	cmdline, err := windows.UTF16PtrFromString(defaultWindowsShell())
	if err != nil {
		return
	}

	attrList, attrListPtr, err := newPseudoConsoleAttributeList(terminal.hpc)
	if err != nil {
		return
	}
	defer deleteProcThreadAttributeList(attrListPtr)

	si := &startupInfoEx{AttributeList: attrListPtr}
	si.StartupInfo.Cb = uint32(unsafe.Sizeof(*si))

	err = windows.CreateProcess(
		nil,
		cmdline,
		nil,
		nil,
		false,
		windows.EXTENDED_STARTUPINFO_PRESENT|windows.CREATE_NO_WINDOW,
		nil,
		nil,
		(*windows.StartupInfo)(unsafe.Pointer(si)),
		&terminal.process,
	)
	runtime.KeepAlive(attrList)
	if err != nil {
		return
	}

	terminal.closeMu.Lock()
	if terminal.closed {
		process := terminal.process.Process
		thread := terminal.process.Thread
		terminal.process.Process = 0
		terminal.process.Thread = 0
		terminal.closeMu.Unlock()
		if thread != 0 {
			_ = windows.CloseHandle(thread)
		}
		if process != 0 {
			_ = windows.TerminateProcess(process, 1)
			_ = windows.CloseHandle(process)
		}
		terminal.close()
		return
	}
	process := terminal.process.Process
	thread := terminal.process.Thread
	terminal.process.Thread = 0
	terminal.closeMu.Unlock()

	if thread != 0 {
		_ = windows.CloseHandle(thread)
	}

	ready := &protocol.TerminalReady{Session: terminal.session, OK: 1}
	send(readyHeader, ready)

	go func() {
		_, _ = windows.WaitForSingleObject(process, windows.INFINITE)
		terminal.close()
		exit := &protocol.TerminalExit{Session: terminal.session, OK: 1}
		send(exitHeader, exit)
	}()

	buf := make([]byte, 4096)
	for {
		outputRead := terminal.outputReadHandle()
		if outputRead == 0 {
			return
		}
		var done uint32
		readErr := windows.ReadFile(outputRead, buf, &done, nil)
		if done > 0 {
			data := append([]byte(nil), buf[:done]...)
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

func (terminal *windowsTerminal) open(cols, rows uint16) error {
	if err := ensureConPTYAvailable(); err != nil {
		return err
	}
	if cols == 0 {
		cols = 80
	}
	if rows == 0 {
		rows = 24
	}

	var inputRead, outputWrite windows.Handle
	var inputWrite, outputRead windows.Handle
	err := windows.CreatePipe(&inputRead, &inputWrite, nil, 0)
	if err != nil {
		return err
	}
	err = windows.CreatePipe(&outputRead, &outputWrite, nil, 0)
	if err != nil {
		_ = windows.CloseHandle(inputRead)
		_ = windows.CloseHandle(inputWrite)
		return err
	}

	var hpc uintptr
	size := coord{X: int16(cols), Y: int16(rows)}
	err = createPseudoConsole(size, inputRead, outputWrite, &hpc)
	_ = windows.CloseHandle(inputRead)
	_ = windows.CloseHandle(outputWrite)
	if err != nil {
		_ = windows.CloseHandle(inputWrite)
		_ = windows.CloseHandle(outputRead)
		return err
	}

	terminal.closeMu.Lock()
	if terminal.closed {
		terminal.closeMu.Unlock()
		_ = windows.CloseHandle(inputWrite)
		_ = windows.CloseHandle(outputRead)
		closePseudoConsole(hpc)
		return errors.New("terminal closed")
	}
	terminal.inputWrite = inputWrite
	terminal.outputRead = outputRead
	terminal.hpc = hpc
	terminal.closeMu.Unlock()

	return nil
}

func ensureConPTYAvailable() error {
	if err := procCreatePseudoConsole.Find(); err != nil {
		return errors.New("ConPTY unavailable: CreatePseudoConsole not found")
	}
	if err := procResizePseudoConsole.Find(); err != nil {
		return errors.New("ConPTY unavailable: ResizePseudoConsole not found")
	}
	if err := procClosePseudoConsole.Find(); err != nil {
		return errors.New("ConPTY unavailable: ClosePseudoConsole not found")
	}
	return nil
}

func defaultWindowsShell() string {
	if shell := os.Getenv("ComSpec"); shell != "" {
		return shell
	}
	return `C:\Windows\System32\cmd.exe`
}

func (terminal *windowsTerminal) input(data []byte) {
	terminal.writeMu.Lock()
	defer terminal.writeMu.Unlock()
	inputWrite := terminal.inputWriteHandle()
	if inputWrite == 0 || len(data) == 0 {
		return
	}
	var done uint32
	_ = windows.WriteFile(inputWrite, data, &done, nil)
}

func (terminal *windowsTerminal) resize(cols, rows uint16) {
	hpc := terminal.pseudoConsoleHandle()
	if hpc == 0 || cols == 0 || rows == 0 {
		return
	}
	_ = resizePseudoConsole(hpc, coord{X: int16(cols), Y: int16(rows)})
}

func (terminal *windowsTerminal) close() {
	terminal.closeMu.Lock()
	terminal.closed = true
	inputWrite := terminal.inputWrite
	outputRead := terminal.outputRead
	process := terminal.process.Process
	thread := terminal.process.Thread
	hpc := terminal.hpc
	terminal.inputWrite = 0
	terminal.outputRead = 0
	terminal.process.Process = 0
	terminal.process.Thread = 0
	terminal.hpc = 0
	terminal.closeMu.Unlock()

	if process != 0 {
		_ = windows.TerminateProcess(process, 1)
	}
	if inputWrite != 0 {
		_ = windows.CloseHandle(inputWrite)
	}
	if outputRead != 0 {
		_ = windows.CloseHandle(outputRead)
	}
	if process != 0 {
		_ = windows.CloseHandle(process)
	}
	if thread != 0 {
		_ = windows.CloseHandle(thread)
	}
	if hpc != 0 {
		closePseudoConsole(hpc)
	}
}

func (terminal *windowsTerminal) isClosed() bool {
	terminal.closeMu.Lock()
	defer terminal.closeMu.Unlock()
	return terminal.closed
}

func (terminal *windowsTerminal) inputWriteHandle() windows.Handle {
	terminal.closeMu.Lock()
	defer terminal.closeMu.Unlock()
	if terminal.closed {
		return 0
	}
	return terminal.inputWrite
}

func (terminal *windowsTerminal) outputReadHandle() windows.Handle {
	terminal.closeMu.Lock()
	defer terminal.closeMu.Unlock()
	if terminal.closed {
		return 0
	}
	return terminal.outputRead
}

func (terminal *windowsTerminal) pseudoConsoleHandle() uintptr {
	terminal.closeMu.Lock()
	defer terminal.closeMu.Unlock()
	if terminal.closed {
		return 0
	}
	return terminal.hpc
}

func createPseudoConsole(size coord, input, output windows.Handle, hpc *uintptr) error {
	r1, _, e1 := procCreatePseudoConsole.Call(
		uintptr(*(*uint32)(unsafe.Pointer(&size))),
		uintptr(input),
		uintptr(output),
		0,
		uintptr(unsafe.Pointer(hpc)),
	)
	if r1 != 0 {
		if e1 != syscall.Errno(0) {
			return e1
		}
		return syscall.Errno(r1)
	}
	return nil
}

func resizePseudoConsole(hpc uintptr, size coord) error {
	r1, _, e1 := procResizePseudoConsole.Call(
		hpc,
		uintptr(*(*uint32)(unsafe.Pointer(&size))),
	)
	if r1 != 0 {
		if e1 != syscall.Errno(0) {
			return e1
		}
		return syscall.Errno(r1)
	}
	return nil
}

func closePseudoConsole(hpc uintptr) {
	procClosePseudoConsole.Call(hpc)
}

func newPseudoConsoleAttributeList(hpc uintptr) ([]byte, *byte, error) {
	var size uintptr
	r1, _, e1 := procInitializeProcThreadAttrList.Call(0, 1, 0, uintptr(unsafe.Pointer(&size)))
	if r1 == 0 && size == 0 && e1 != syscall.Errno(0) {
		return nil, nil, e1
	}
	if size == 0 {
		return nil, nil, errors.New("empty proc thread attribute list")
	}

	attrList := make([]byte, size)
	listPtr := &attrList[0]
	r1, _, e1 = procInitializeProcThreadAttrList.Call(uintptr(unsafe.Pointer(listPtr)), 1, 0, uintptr(unsafe.Pointer(&size)))
	if r1 == 0 {
		return nil, nil, e1
	}

	r1, _, e1 = procUpdateProcThreadAttribute.Call(
		uintptr(unsafe.Pointer(listPtr)),
		0,
		procThreadAttributePseudoConsole,
		hpc,
		unsafe.Sizeof(hpc),
		0,
		0,
	)
	if r1 == 0 {
		deleteProcThreadAttributeList(listPtr)
		return nil, nil, e1
	}
	return attrList, listPtr, nil
}

func deleteProcThreadAttributeList(attrList *byte) {
	if attrList != nil {
		procDeleteProcThreadAttributeList.Call(uintptr(unsafe.Pointer(attrList)))
	}
}

func DispatchTerminalMess(mgr *manager.Manager) {
	var terminal *windowsTerminal

	for {
		message := <-mgr.TerminalManager.TerminalMessChan

		switch mess := message.(type) {
		case *protocol.TerminalStart:
			if terminal != nil {
				terminal.close()
			}
			terminal = newWindowsTerminal(mess.Session)
			go terminal.start(mess.Cols, mess.Rows)
		case *protocol.TerminalData:
			if terminal != nil && terminal.session == mess.Session {
				terminal.input(normalizeWindowsTerminalInput(mess.Data))
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

func normalizeWindowsTerminalInput(data []byte) []byte {
	return []byte(strings.ReplaceAll(string(data), "\n", "\r"))
}
