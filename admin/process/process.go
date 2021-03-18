/*
 * @Author: ph4ntom
 * @Date: 2021-03-16 16:10:23
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-03-18 18:47:07
 */
package process

import (
	"Stowaway/admin/cli"
	"Stowaway/admin/initial"
	"Stowaway/admin/topology"
	"Stowaway/crypto"
	"Stowaway/protocol"
	"fmt"
	"log"
	"net"
)

type Admin struct {
	ID           string
	Conn         net.Conn
	CryptoSecret []byte
	Topology     *topology.Topology
	Route        *topology.Route
	UserOptions  *initial.Options
}

func NewAdmin(options *initial.Options) *Admin {
	admin := new(Admin)
	admin.ID = protocol.ADMIN_UUID
	admin.CryptoSecret, _ = crypto.KeyPadding([]byte(options.Secret))
	admin.UserOptions = options
	return admin
}

func (admin *Admin) Run() {
	task := &topology.RouteTask{
		Mode: topology.CALCULATE,
	}
	admin.Route.TaskChan <- task
	result := <-admin.Route.ResultChan

	console := cli.NewConsole()
	console.Init(admin.Topology, admin.Route, admin.Conn, admin.ID, admin.UserOptions.Secret, admin.CryptoSecret)

	go admin.handleDataFromDownstream(console, result.RouteInfo)

	console.Run() // start interactive panel
}

func (admin *Admin) handleDataFromDownstream(console *cli.Console, routeMap map[int]string) {
	rMessage := protocol.PrepareAndDecideWhichRProto(admin.Conn, admin.UserOptions.Secret, protocol.ADMIN_UUID)
	for {
		fHeader, fMessage, err := protocol.DestructMessage(rMessage)
		if err != nil {
			log.Print("\n[*]Peer node seems offline!")
			break
		}
		switch fHeader.MessageType {
		case protocol.MYINFO:
			message := fMessage.(*protocol.MyInfo)
			task := &topology.TopoTask{
				Mode:     topology.UPDATEDETAIL,
				ID:       fHeader.Sender,
				UserName: message.Username,
				HostName: message.Hostname,
			}
			admin.Topology.TaskChan <- task
		case protocol.SHELLRES:
			message := fMessage.(*protocol.ShellRes)
			if message.OK == 1 {
				console.OK <- true
				fmt.Print("\n[*]Shell is started successfully!")
			} else {
				console.OK <- false
				fmt.Print("\n[*]Shell cannot be started!")
			}
		case protocol.SHELLRESULT:
			message := fMessage.(*protocol.ShellResult)
			if message.OK == 1 {
				fmt.Print(message.Result)
			} else {
				fmt.Print("\n[*]Command cannot be executed!")
			}
		case protocol.LISTENRES:
			message := fMessage.(*protocol.ListenRes)
			if message.OK == 1 {
				fmt.Print("\n[*]Listen successfully!")
			} else {
				fmt.Print("\n[*]Listen failed!")
			}
		default:
			log.Print("\n[*]Unknown Message!")
		}
	}
}
