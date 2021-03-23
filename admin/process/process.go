/*
 * @Author: ph4ntom
 * @Date: 2021-03-16 16:10:23
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-03-23 19:03:43
 */
package process

import (
	"Stowaway/admin/cli"
	"Stowaway/admin/initial"
	"Stowaway/admin/topology"
	"Stowaway/crypto"
	"Stowaway/protocol"
	"Stowaway/share"
	"fmt"
	"log"
	"net"
	"os"
)

type Admin struct {
	UUID         string
	Conn         net.Conn
	CryptoSecret []byte
	Topology     *topology.Topology
	UserOptions  *initial.Options
	// manager that needs to be shared with console
	mgr *share.Manager
}

func NewAdmin(options *initial.Options) *Admin {
	admin := new(Admin)
	admin.UUID = protocol.ADMIN_UUID
	admin.CryptoSecret, _ = crypto.KeyPadding([]byte(options.Secret))
	admin.UserOptions = options
	return admin
}

func (admin *Admin) Run() {
	file := share.NewFile()

	admin.mgr = share.NewManager(file)
	go admin.mgr.Run()

	console := cli.NewConsole()
	console.Init(admin.Topology, admin.mgr, admin.Conn, admin.UUID, admin.UserOptions.Secret, admin.CryptoSecret)

	go admin.handleDataFromDownstream(console)

	console.Run() // start interactive panel
}

func (admin *Admin) handleDataFromDownstream(console *cli.Console) {
	rMessage := protocol.PrepareAndDecideWhichRProtoFromUpper(admin.Conn, admin.UserOptions.Secret, protocol.ADMIN_UUID)
	for {
		fHeader, fMessage, err := protocol.DestructMessage(rMessage)
		if err != nil {
			log.Print("\n[*]Peer node seems offline!")
			os.Exit(0)
		}
		switch fHeader.MessageType {
		case protocol.MYINFO:
			message := fMessage.(*protocol.MyInfo)
			task := &topology.TopoTask{
				Mode:     topology.UPDATEDETAIL,
				UUID:     fHeader.Sender,
				UserName: message.Username,
				HostName: message.Hostname,
			}
			admin.Topology.TaskChan <- task
		case protocol.SHELLRES:
			message := fMessage.(*protocol.ShellRes)
			if message.OK == 1 {
				fmt.Print("\r\n[*]Shell is started successfully!\r\n")
				console.OK <- true
			} else {
				fmt.Print("\r\n[*]Shell cannot be started!")
				console.OK <- false
			}
		case protocol.SHELLRESULT:
			message := fMessage.(*protocol.ShellResult)
			fmt.Print(message.Result)
		case protocol.LISTENRES:
			message := fMessage.(*protocol.ListenRes)
			if message.OK == 1 {
				fmt.Print("\n[*]Listen successfully!")
			} else {
				fmt.Print("\n[*]Listen failed!")
			}
		case protocol.SSHRES:
			message := fMessage.(*protocol.SSHRes)
			if message.OK == 1 {
				fmt.Print("\r\n[*]Connect to target host via ssh successfully!")
				console.OK <- true
			} else {
				fmt.Print("\r\n[*]Fail to connect to target host via ssh!")
				console.OK <- false
			}
		case protocol.SSHRESULT:
			message := fMessage.(*protocol.SSHResult)
			fmt.Printf("\033[u\033[K\r%s", message.Result)
			fmt.Printf("\r\n%s", console.Status)
		case protocol.FILESTATREQ:
			message := fMessage.(*protocol.FileStatReq)
			admin.mgr.File.FileSize = int64(message.FileSize)
			admin.mgr.File.SliceNum = message.SliceNum
			console.OK <- true
		case protocol.FILEDOWNRES:
			// no need to check mess
			fmt.Print("\r\n[*]Unable to download file!")
			console.OK <- false
		case protocol.FILESTATRES:
			message := fMessage.(*protocol.FileStatRes)
			if message.OK == 1 {
				console.OK <- true
			} else {
				fmt.Print("\r\n[*]Fail to upload file!")
				admin.mgr.File.Handler.Close()
				console.OK <- false
			}
		case protocol.FILEDATA:
			message := fMessage.(*protocol.FileData)
			admin.mgr.File.DataChan <- message.Data
		case protocol.FILEERR:
			// no need to check mess
			admin.mgr.File.ErrChan <- true
		default:
			log.Print("\n[*]Unknown Message!")
		}
	}
}
