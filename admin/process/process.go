/*
 * @Author: ph4ntom
 * @Date: 2021-03-16 16:10:23
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-04-01 15:32:59
 */
package process

import (
	"Stowaway/admin/cli"
	"Stowaway/admin/handler"
	"Stowaway/admin/initial"
	"Stowaway/admin/manager"
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
	Conn         net.Conn
	CryptoSecret []byte
	Topology     *topology.Topology
	UserOptions  *initial.Options

	BufferChan chan *BufferData
	// manager that needs to be shared with console
	mgr *manager.Manager
}

type BufferData struct {
	fHeader  *protocol.Header
	fMessage interface{}
}

func NewAdmin(options *initial.Options) *Admin {
	admin := new(Admin)
	admin.CryptoSecret, _ = crypto.KeyPadding([]byte(options.Secret))
	admin.UserOptions = options
	admin.BufferChan = make(chan *BufferData, 10)
	return admin
}

func (admin *Admin) Run() {
	file := share.NewFile()

	admin.mgr = manager.NewManager(file)
	go admin.mgr.Run()

	console := cli.NewConsole()
	console.Init(admin.Topology, admin.mgr, admin.Conn, admin.UserOptions.Secret, admin.CryptoSecret)

	go admin.handleConnFromDownstream(console)
	go admin.handleDataFromDownstream(console)

	console.Run() // start interactive panel
}

func (admin *Admin) handleConnFromDownstream(console *cli.Console) {
	rMessage := protocol.PrepareAndDecideWhichRProtoFromUpper(admin.Conn, admin.UserOptions.Secret, protocol.ADMIN_UUID)

	for {
		fHeader, fMessage, err := protocol.DestructMessage(rMessage)
		if err != nil {
			log.Print("\n[*]Peer node seems offline!")
			os.Exit(0)
		}

		admin.BufferChan <- &BufferData{fHeader: fHeader, fMessage: fMessage}
	}

}

func (admin *Admin) handleDataFromDownstream(console *cli.Console) {
	for {
		data := <-admin.BufferChan

		switch data.fHeader.MessageType {
		case protocol.MYINFO:
			message := data.fMessage.(*protocol.MyInfo)
			// register new node
			task := &topology.TopoTask{
				Mode:     topology.UPDATEDETAIL,
				UUID:     data.fHeader.Sender,
				UserName: message.Username,
				HostName: message.Hostname,
			}
			admin.Topology.TaskChan <- task
		case protocol.SHELLRES:
			message := data.fMessage.(*protocol.ShellRes)
			if message.OK == 1 {
				fmt.Print("\r\n[*]Shell is started successfully!\r\n")
				console.OK <- true
			} else {
				fmt.Print("\r\n[*]Shell cannot be started!")
				console.OK <- false
			}
		case protocol.SHELLRESULT:
			message := data.fMessage.(*protocol.ShellResult)
			fmt.Print(message.Result)
		case protocol.LISTENRES:
			message := data.fMessage.(*protocol.ListenRes)
			if message.OK == 1 {
				fmt.Print("\n[*]Listen successfully!")
			} else {
				fmt.Print("\n[*]Listen failed!")
			}
		case protocol.SSHRES:
			message := data.fMessage.(*protocol.SSHRes)
			if message.OK == 1 {
				fmt.Print("\r\n[*]Connect to target host via ssh successfully!")
				console.OK <- true
			} else {
				fmt.Print("\r\n[*]Fail to connect to target host via ssh!")
				console.OK <- false
			}
		case protocol.SSHRESULT:
			message := data.fMessage.(*protocol.SSHResult)
			fmt.Printf("\r\033[K%s", message.Result)
			fmt.Printf("\r\n%s", console.Status)
		case protocol.FILESTATREQ:
			message := data.fMessage.(*protocol.FileStatReq)
			admin.mgr.File.FileSize = int64(message.FileSize)
			admin.mgr.File.SliceNum = message.SliceNum
			console.OK <- true
		case protocol.FILEDOWNRES:
			// no need to check mess
			fmt.Print("\r\n[*]Unable to download file!")
			console.OK <- false
		case protocol.FILESTATRES:
			message := data.fMessage.(*protocol.FileStatRes)
			if message.OK == 1 {
				console.OK <- true
			} else {
				fmt.Print("\r\n[*]Fail to upload file!")
				admin.mgr.File.Handler.Close()
				console.OK <- false
			}
		case protocol.FILEDATA:
			message := data.fMessage.(*protocol.FileData)
			admin.mgr.File.DataChan <- message.Data
		case protocol.FILEERR:
			// no need to check mess
			admin.mgr.File.ErrChan <- true
		case protocol.SOCKSTCPDATA:
			message := data.fMessage.(*protocol.SocksTCPData)
			admin.mgr.SocksTCPDataChan <- message
		case protocol.SOCKSTCPFIN:
			message := data.fMessage.(*protocol.SocksTCPFin)
			admin.mgr.SocksTCPDataChan <- message
		case protocol.UDPASSSTART:
			message := data.fMessage.(*protocol.UDPAssStart)
			go handler.StartUDPAss(admin.mgr, admin.Topology, admin.Conn, admin.UserOptions.Secret, message.Seq)
		case protocol.SOCKSUDPDATA:
			message := data.fMessage.(*protocol.SocksUDPData)
			admin.mgr.SocksUDPDataChan <- message
		default:
			log.Print("\n[*]Unknown Message!")
		}
	}
}
