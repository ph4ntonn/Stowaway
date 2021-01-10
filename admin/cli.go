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
	help                                     		Show Help information
	exit                                     		Exit
	detail                                  		Display connected node detail
	tree                                     		Display nodes's topology
	use        [id]                          		Select the target node you want to use
  `)
}

// ShowNodeHelp 打印node模式下的帮助
func ShowNodeHelp() {
	fmt.Println(`
	help                                     		Show Help information
	listen     [ip:]<port>                  		Start port listening on selected node
	addnote    <string>                      		Add note for this node
	delnote                                  		Delete note of this node
	ssh        <ip:port>    		                Start SSH through selected node
	shell                                    		Start an interactive shell on selected node
	socks      <lport> [username] [pass]     		Start a socks5 server.(username and pass are optional)
	connect    <ip:port>                     		Connect to new node
	sshtunnel  <ip:sshport> <agent port>    		Use sshtunnel to add the node into the whole network
	stopsocks                                		Shut down all socks services
	upload     <filename>                    		Upload file to current agent node
	download   <filename>                   		Download file from current agent node
	forward    <lport> <ip:port>             		Forward local port to specific remote ip:port (eg:forward 8888 192.168.0.100:22)
	stopforward                                		Shut down all forward services
	reflect    <rport> <lport>               		Reflect remote port(agent) to local port(admin) (eg:reflect 22 80)
	stopreflect                                		Shut down all reflect services
	offline                                 		Terminate current node
	exit                                     		Back to upper panel
  `)
}
