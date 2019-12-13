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
	respCommand, _ := common.ConstructCommand("ADMINOFFLINE", "", 1)
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

			{ v0.1  Author:ph4ntom }
`)
}
