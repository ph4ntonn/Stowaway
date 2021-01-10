package admin

import (
	"fmt"

	"Stowaway/config"
)

// Banner 程序图标
func Banner() {
	fmt.Printf(`
▄▀▀▀▀▄  ▄▀▀▀█▀▀▄  ▄▀▀▀▀▄   ▄▀▀▄    ▄▀▀▄  ▄▀▀█▄   ▄▀▀▄    ▄▀▀▄  ▄▀▀█▄   ▄▀▀▄ ▀▀▄ 
█ █   ▐ █    █  ▐ █      █ █   █    ▐  █ ▐ ▄▀ ▀▄ █   █    ▐  █ ▐ ▄▀ ▀▄ █   ▀▄ ▄▀ 
   ▀▄   ▐   █     █      █ ▐  █        █   █▄▄▄█ ▐  █        █   █▄▄▄█ ▐     █   
▀▄   █     █      ▀▄    ▄▀   █   ▄    █   ▄▀   █   █   ▄    █   ▄▀   █       █   
 █▀▀▀    ▄▀         ▀▀▀▀      ▀▄▀ ▀▄ ▄▀  █   ▄▀     ▀▄▀ ▀▄ ▄▀  █   ▄▀      ▄▀    
 ▐      █                           ▀    ▐   ▐            ▀    ▐   ▐       █     
       ▐                                                                  ▐    

			{ v%s  Author:ph4ntom }
`, config.VERSION)
}

// ShowMainHelp 打印admin模式下的帮助
func ShowMainHelp() {
	fmt.Println(`
	help                                     		Show help information
	detail                                  		Display connected nodes' detail
	tree                                     		Display nodes' topology
	use        <id>                          		Select the target node you want to use
	exit                                     		Exit
  `)
}

// ShowNodeHelp 打印node模式下的帮助
func ShowNodeHelp() {
	fmt.Println(`
	help                                     		Show help information
	listen     [ip:]<port>                  		Start port listening on current node
	addnote    <string>                      		Add note for current node
	delnote                                  		Delete note of current node
	ssh        <ip:port>    		                Start SSH through current node
	shell                                    		Start an interactive shell on current node
	socks      <lport> [username] [pass]     		Start a socks5 server.
	connect    <ip:port>                     		Connect to a new node
	sshtunnel  <ip:sshport> <agent port>    		Use sshtunnel to add the node into our topology
	stopsocks                                		Shut down all socks services
	upload     <filename>                    		Upload file to current node
	download   <filename>                   		Download file from current node
	forward    <lport> <ip:port>             		Forward local port to specific remote ip:port (eg:forward 8888 192.168.0.100:22)
	stopforward                                		Shut down all forward services
	reflect    <rport> <lport>               		Reflect remote port(agent) to local port(admin) (eg:reflect 22 80)
	stopreflect                                		Shut down all reflect services
	offline                                 		Terminate current node
	exit                                     		Back to upper panel
  `)
}
