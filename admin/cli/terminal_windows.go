//go:build windows
// +build windows

package cli

import (
	"time"

	"Stowaway/admin/handler"
	"Stowaway/admin/printer"

	"github.com/eiannone/keyboard"
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

	handler.LetTerminalStart(route, uuid, session, 80, 24)

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

	keysEvents, err := keyboard.GetKeys(10)
	if err != nil {
		printer.Fail("\r\n[*] Unable to read local keyboard: %s", err.Error())
		handler.SendTerminalExit(route, uuid, session)
		finish()
		return
	}

	for {
		select {
		case event := <-keysEvents:
			if event.Err != nil {
				continue
			}
			if event.Key == keyboard.KeyCtrlBackslash {
				handler.SendTerminalExit(route, uuid, session)
				finish()
				return
			}
			data := windowsKeyEventBytes(event)
			if len(data) > 0 {
				handler.SendTerminalData(route, uuid, session, data)
			}
		case <-waiter.Exit:
			finish()
			return
		}
	}
}

func windowsKeyEventBytes(event keyboard.KeyEvent) []byte {
	if event.Rune != 0 {
		return []byte(string(event.Rune))
	}

	switch event.Key {
	case keyboard.KeyEnter:
		return []byte{'\r'}
	case keyboard.KeySpace:
		return []byte{' '}
	case keyboard.KeyTab:
		return []byte{'\t'}
	case keyboard.KeyBackspace, keyboard.KeyBackspace2:
		return []byte{0x08}
	case keyboard.KeyEsc:
		return []byte{0x1b}
	case keyboard.KeyArrowUp:
		return []byte{0x1b, '[', 'A'}
	case keyboard.KeyArrowDown:
		return []byte{0x1b, '[', 'B'}
	case keyboard.KeyArrowRight:
		return []byte{0x1b, '[', 'C'}
	case keyboard.KeyArrowLeft:
		return []byte{0x1b, '[', 'D'}
	case keyboard.KeyHome:
		return []byte{0x1b, '[', 'H'}
	case keyboard.KeyEnd:
		return []byte{0x1b, '[', 'F'}
	case keyboard.KeyDelete:
		return []byte{0x1b, '[', '3', '~'}
	case keyboard.KeyPgup:
		return []byte{0x1b, '[', '5', '~'}
	case keyboard.KeyPgdn:
		return []byte{0x1b, '[', '6', '~'}
	}

	if event.Key > 0 && event.Key < keyboard.KeySpace {
		return []byte{byte(event.Key)}
	}
	if event.Key == keyboard.KeyCtrlSpace {
		return []byte{0}
	}

	return nil
}
