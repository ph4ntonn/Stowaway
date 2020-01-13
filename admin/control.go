package admin

import (
	"Stowaway/common"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	time.Sleep(2 * time.Second)
	os.Exit(1)
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

			{ v1.2  Author:ph4ntom }
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
	connect    [ip:port]					 Connect to new node
	stopsocks                                Shut down corresponding socks service
  `)
}
