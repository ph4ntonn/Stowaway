package admin

import (
	"Stowaway/config"
	"fmt"
)

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

func ShowMainHelp() {
	fmt.Println(`
	help                                     		Show Help information.
	exit                                     		Exit.
	detail                                  		Display connected node detail
	tree                                     		Display nodes's topology
	use        [id]                          		Select the target node you want to use.
  `)
}

func ShowNodeHelp() {
	fmt.Println(`
	help                                     		Show Help information.
	addnote    [string]                      		Add note for this node
	delnote                                  		Delete note of this node
	ssh        [ip:port] [username] [pass]   		Start SSH through selected node.
	shell                                    		Start an interactive shell on selected node.
	socks      [lport] [username] [pass]     		Start a socks5 server.(username and pass are optional)
	connect    [ip:port]                     		Connect to new node
	sshtunnel  [ip:sshport] [agent port]    		Use sshtunnel to add the node into the whole network
	stopsocks                                		Shut down corresponding socks service
	upload     [filename]                    		Upload file to current agent node
	download   [filename]                   		Download file from current agent node
	forward    [lport] [ip:port]             		Forward local port to remote (eg:forward 8888 192.168.0.100:22)
	reflect    [rport] [lport]               		Reflect remote port to local port (eg:reflect 22 80)
	exit                                     		Back to upper panel
  `)
}
