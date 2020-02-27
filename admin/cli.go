package admin

import (
	"fmt"
)

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
	socks      [lport] [username] [pass]     Start a socks5 server.(username and pass are optional)
	connect    [ip:port]					 Connect to new node
	stopsocks                                Shut down corresponding socks service
	upload     [filename]                    Upload file to current agent node
	download   [filename]                    Download file from current agent node
  `)
}
