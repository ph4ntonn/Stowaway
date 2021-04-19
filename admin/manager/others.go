package manager

import (
	"Stowaway/share"
)

type consoleManager struct {
	OK chan bool
}

func newConsoleManager() *consoleManager {
	manager := new(consoleManager)
	manager.OK = make(chan bool)
	return manager
}

type fileManager struct {
	File *share.MyFile

	FileMessChan chan interface{}
}

func newFileManager(file *share.MyFile) *fileManager {
	manager := new(fileManager)
	manager.File = file
	manager.FileMessChan = make(chan interface{}, 5)
	return manager
}

type sshManager struct {
	SSHMessChan chan interface{}
}

func newSSHManager() *sshManager {
	manager := new(sshManager)
	manager.SSHMessChan = make(chan interface{}, 5)
	return manager
}

type sshTunnelManager struct {
	SSHTunnelMessChan chan interface{}
}

func newSSHTunnelManager() *sshTunnelManager {
	manager := new(sshTunnelManager)
	manager.SSHTunnelMessChan = make(chan interface{}, 5)
	return manager
}

type shellManager struct {
	ShellMessChan chan interface{}
}

func newShellManager() *shellManager {
	manager := new(shellManager)
	manager.ShellMessChan = make(chan interface{}, 5)
	return manager
}

type infoManager struct {
	InfoMessChan chan interface{}
}

func newInfoManager() *infoManager {
	manager := new(infoManager)
	manager.InfoMessChan = make(chan interface{}, 5)
	return manager
}

type listenManager struct {
	ListenMessChan chan interface{}
	ListenReady    chan bool
}

func newListenManager() *listenManager {
	manager := new(listenManager)
	manager.ListenMessChan = make(chan interface{}, 5)
	manager.ListenReady = make(chan bool)
	return manager
}

type connectManager struct {
	ConnectMessChan chan interface{}
	ConnectReady    chan bool
}

func newConnectManager() *connectManager {
	manager := new(connectManager)
	manager.ConnectMessChan = make(chan interface{}, 5)
	manager.ConnectReady = make(chan bool)
	return manager
}
