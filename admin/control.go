package admin

import (
	"Stowaway/common"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func SendOffLineToStartNode(startNodeControlConn net.Conn) {
	respCommand, _ := common.ConstructCommand("ADMINOFFLINE", "", 1, AESKey)
	_, err := startNodeControlConn.Write(respCommand)
	if err != nil {
		fmt.Println("Startnode seems offline!Message cannot be transmitted")
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

			{ v1.3  Author:ph4ntom }
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
	upload     [filename]                    Upload file to current agent node
	download   [filename]                    Download file from current agent node
  `)
}
