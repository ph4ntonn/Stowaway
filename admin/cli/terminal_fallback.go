package cli

import (
	"fmt"
	"strings"
	"time"

	"Stowaway/admin/handler"
	"Stowaway/admin/printer"
)

func (console *Console) fallbackLegacyShell(route string, uuid string) bool {
	printer.Warning("\r\n[*] Terminal Stream unavailable, falling back to legacy shell.....")
	status := console.status
	console.status = ""
	ok := console.startLegacyShell(route, uuid)
	console.status = status
	if ok {
		fmt.Print("\r\n")
		fmt.Print(console.status)
	}
	return ok
}

func (console *Console) startLegacyShell(route string, uuid string) bool {
	drainConsoleExit(console)
	handler.LetShellStart(route, uuid)
	ok, timedOut := console.waitConsoleOK(10 * time.Second)
	if timedOut {
		printer.Fail("\r\n[*] Legacy shell start timed out!")
		console.ready <- true
		return false
	}
	if !ok {
		printer.Fail("\r\n[*] Legacy shell cannot be started!")
		console.ready <- true
		return false
	}
	console.handleShellPanelCommand(route, uuid)
	return true
}

func drainConsoleExit(console *Console) {
	for {
		select {
		case <-console.mgr.ConsoleManager.Exit:
		default:
			return
		}
	}
}

func (console *Console) waitConsoleOK(timeout time.Duration) (bool, bool) {
	select {
	case ok := <-console.mgr.ConsoleManager.OK:
		return ok, false
	case <-time.After(timeout):
		return false, true
	}
}

func isLegacyShellExit(command string) bool {
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return false
	}
	return fields[0] == "exit" || fields[0] == "logout"
}
