package manager

type consoleManager struct {
	OK chan bool
}

func newConsoleManager() *consoleManager {
	manager := new(consoleManager)
	manager.OK = make(chan bool)
	return manager
}
