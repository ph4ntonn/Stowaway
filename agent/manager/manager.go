/*
 * @Author: ph4ntom
 * @Date: 2021-03-23 19:01:26
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-04-02 17:23:16
 */
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
