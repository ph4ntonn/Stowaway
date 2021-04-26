package manager

import (
	"Stowaway/share"
)

// Manager is used to maintain all status and keep different parts connected
type Manager struct {
	ChildrenManager  *childrenManager
	FileManager      *fileManager
	SocksManager     *socksManager
	ForwardManager   *forwardManager
	BackwardManager  *backwardManager
	SSHManager       *sshManager
	SSHTunnelManager *sshTunnelManager
	ShellManager     *shellManager
	ListenManager    *listenManager
	ConnectManager   *connectManager
	OfflineManager   *offlineManager
}

func NewManager(file *share.MyFile) *Manager {
	manager := new(Manager)
	manager.ChildrenManager = newChildrenManager()
	manager.FileManager = newFileManager(file)
	manager.SocksManager = newSocksManager()
	manager.ForwardManager = newForwardManager()
	manager.BackwardManager = newBackwardManager()
	manager.SSHManager = newSSHManager()
	manager.SSHTunnelManager = newSSHTunnelManager()
	manager.ShellManager = newShellManager()
	manager.ListenManager = newListenManager()
	manager.ConnectManager = newConnectManager()
	manager.OfflineManager = newOfflineManager()
	return manager
}

func (manager *Manager) Run() {
	go manager.ChildrenManager.run()
	go manager.SocksManager.run()
	go manager.ForwardManager.run()
	go manager.BackwardManager.run()
}
