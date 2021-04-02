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

type Manager struct {
	File           *share.MyFile
	SocksManager   *socksManager
	ForwardManager *forwardManager
}

func NewManager(file *share.MyFile) *Manager {
	manager := new(Manager)
	manager.File = file
	manager.SocksManager = newSocksManager()
	manager.ForwardManager = newForwardManager()
	return manager
}

func (manager *Manager) Run() {
	go manager.SocksManager.run()
	go manager.ForwardManager.run()
}
