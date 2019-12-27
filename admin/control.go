package admin

import (
	"Stowaway/common"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
)

func SendOffLineToStartNode(startNodeControlConn net.Conn) {
	respCommand, _ := common.ConstructCommand("ADMINOFFLINE", "", 1, AESKey)
	_, err := startNodeControlConn.Write(respCommand)
	if err != nil {
		logrus.Error(err)
		return
	}
}

func MonitorCtrlC(startNodeControlConn net.Conn) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan
	SendOffLineToStartNode(startNodeControlConn)
}

func Banner() {
	fmt.Print(`
▄▀▀▀▀▄  ▄▀▀▀█▀▀▄  ▄▀▀▀▀▄   ▄▀▀▄    ▄▀▀▄  ▄▀▀█▄   ▄▀▀▄    ▄▀▀▄  ▄▀▀█▄   ▄▀▀▄ ▀▀▄ 
█ █   ▐ █    █  ▐ █      █ █   █    ▐  █ ▐ ▄▀ ▀▄ █   █    ▐  █ ▐ ▄▀ ▀▄ █   ▀▄ ▄▀ 
   ▀▄   ▐   █     █      █ ▐  █        █   █▄▄▄█ ▐  █        █   █▄▄▄█ ▐     █   
▀▄   █     █      ▀▄    ▄▀   █   ▄    █   ▄▀   █   █   ▄    █   ▄▀   █       █   
 █▀▀▀    ▄▀         ▀▀▀▀      ▀▄▀ ▀▄ ▄▀  █   ▄▀     ▀▄▀ ▀▄ ▄▀  █   ▄▀      ▄▀    
 ▐      █                           ▀    ▐   ▐            ▀    ▐   ▐       █     
		▐                                                                  ▐    

			{ v1.1  Author:ph4ntom }
`)
}

func ShowMainHelp() {
	fmt.Println(`
	help                                     Show Help information.
	exit                                     Exit.
	chain                                    Display connected node information
	use        [id]                          Select the target node you want to use.
  `)
}

func ShowNodeHelp() {
	fmt.Println(`
	help                                     Show Help information.
	exit                                     Exit.
	ssh        [ip:port] [username] [pass]   Start SSH through selected node.
	shell                                    Start an interactive shell on selected node.
	socks      [lport]                       Start a socks5 server.
	stopsocks                                Shut down corresponding socks service
  `)
}
