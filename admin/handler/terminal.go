package handler

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"Stowaway/admin/manager"
	"Stowaway/global"
	"Stowaway/protocol"
)

var terminalSession = struct {
	sync.RWMutex
	id uint64
}{}

var terminalSessionCounter uint64

type TerminalWaiter struct {
	Session uint64
	Ready   chan bool
	Exit    chan struct{}
}

var terminalWaiters = struct {
	sync.Mutex
	items map[uint64]*TerminalWaiter
}{
	items: make(map[uint64]*TerminalWaiter),
}

func NewTerminalSession() uint64 {
	session := uint64(time.Now().UnixNano()) ^ atomic.AddUint64(&terminalSessionCounter, 1)
	if session == 0 {
		session = atomic.AddUint64(&terminalSessionCounter, 1)
	}
	return session
}

func SetActiveTerminalSession(session uint64) {
	terminalSession.Lock()
	terminalSession.id = session
	terminalSession.Unlock()
}

func ClearActiveTerminalSession(session uint64) {
	terminalSession.Lock()
	if terminalSession.id == session {
		terminalSession.id = 0
	}
	terminalSession.Unlock()
}

func isActiveTerminalSession(session uint64) bool {
	terminalSession.RLock()
	defer terminalSession.RUnlock()
	return terminalSession.id == session
}

func RegisterTerminalWaiter(session uint64) *TerminalWaiter {
	waiter := &TerminalWaiter{
		Session: session,
		Ready:   make(chan bool, 1),
		Exit:    make(chan struct{}, 1),
	}
	terminalWaiters.Lock()
	terminalWaiters.items[session] = waiter
	terminalWaiters.Unlock()
	return waiter
}

func UnregisterTerminalWaiter(session uint64) {
	terminalWaiters.Lock()
	delete(terminalWaiters.items, session)
	terminalWaiters.Unlock()
}

func (waiter *TerminalWaiter) WaitReady(timeout time.Duration) (bool, bool) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case ok := <-waiter.Ready:
		return ok, false
	case <-timer.C:
		return false, true
	}
}

func LetTerminalStart(route string, uuid string, session uint64, cols, rows uint16) {
	sMessage := protocol.NewDownMsg(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)

	header := &protocol.Header{
		Sender:      protocol.ADMIN_UUID,
		Accepter:    uuid,
		MessageType: protocol.TERMINALSTART,
		RouteLen:    uint32(len([]byte(route))),
		Route:       route,
	}

	startMess := &protocol.TerminalStart{
		Start:   1,
		Session: session,
		Cols:    cols,
		Rows:    rows,
	}

	protocol.ConstructMessage(sMessage, header, startMess, false)
	sMessage.SendMessage()
}

func SendTerminalData(route string, uuid string, session uint64, data []byte) {
	if len(data) == 0 {
		return
	}

	sMessage := protocol.NewDownMsg(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)

	header := &protocol.Header{
		Sender:      protocol.ADMIN_UUID,
		Accepter:    uuid,
		MessageType: protocol.TERMINALDATA,
		RouteLen:    uint32(len([]byte(route))),
		Route:       route,
	}

	dataMess := &protocol.TerminalData{
		Session: session,
		DataLen: uint64(len(data)),
		Data:    data,
	}

	protocol.ConstructMessage(sMessage, header, dataMess, false)
	sMessage.SendMessage()
}

func SendTerminalResize(route string, uuid string, session uint64, cols, rows uint16) {
	if cols == 0 || rows == 0 {
		return
	}

	sMessage := protocol.NewDownMsg(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)

	header := &protocol.Header{
		Sender:      protocol.ADMIN_UUID,
		Accepter:    uuid,
		MessageType: protocol.TERMINALRESIZE,
		RouteLen:    uint32(len([]byte(route))),
		Route:       route,
	}

	resizeMess := &protocol.TerminalResize{
		Session: session,
		Cols:    cols,
		Rows:    rows,
	}

	protocol.ConstructMessage(sMessage, header, resizeMess, false)
	sMessage.SendMessage()
}

func SendTerminalExit(route string, uuid string, session uint64) {
	sMessage := protocol.NewDownMsg(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)

	header := &protocol.Header{
		Sender:      protocol.ADMIN_UUID,
		Accepter:    uuid,
		MessageType: protocol.TERMINALEXIT,
		RouteLen:    uint32(len([]byte(route))),
		Route:       route,
	}

	exitMess := &protocol.TerminalExit{Session: session, OK: 1}
	protocol.ConstructMessage(sMessage, header, exitMess, false)
	sMessage.SendMessage()
}

func notifyTerminalReady(ready *protocol.TerminalReady) {
	terminalWaiters.Lock()
	waiter := terminalWaiters.items[ready.Session]
	terminalWaiters.Unlock()
	if waiter == nil {
		return
	}

	select {
	case waiter.Ready <- ready.OK == 1:
	default:
	}
}

func notifyTerminalExit(session uint64) {
	terminalWaiters.Lock()
	waiter := terminalWaiters.items[session]
	terminalWaiters.Unlock()
	if waiter == nil {
		return
	}

	select {
	case waiter.Exit <- struct{}{}:
	default:
	}
}

func DispatchTerminalMess(mgr *manager.Manager) {
	for {
		message := <-mgr.TerminalManager.TerminalMessChan

		switch mess := message.(type) {
		case *protocol.TerminalReady:
			if !isActiveTerminalSession(mess.Session) {
				continue
			}
			if mess.OK == 1 {
				notifyTerminalReady(mess)
			} else {
				if mess.Error != "" {
					fmt.Fprintf(os.Stderr, "\r\n[*] Terminal error: %s\r\n", mess.Error)
				}
				notifyTerminalReady(mess)
			}
		case *protocol.TerminalData:
			if !isActiveTerminalSession(mess.Session) {
				continue
			}
			_, _ = os.Stdout.Write(mess.Data)
		case *protocol.TerminalExit:
			if !isActiveTerminalSession(mess.Session) {
				continue
			}
			notifyTerminalExit(mess.Session)
		}
	}
}
