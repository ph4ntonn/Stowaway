/*
 * @Author: ph4ntom
 * @Date: 2021-03-08 14:44:07
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-03-18 15:59:02
 */
package cli

import (
	"fmt"
)

const STOWAWAY_VERSION = "v2.0"

// Banner 程序图标
func Banner() {
	fmt.Printf(`
    .-')    .-') _                  ('\ .-') /'  ('-.      ('\ .-') /'  ('-.                 
   ( OO ). (  OO) )                  '.( OO ),' ( OO ).-.   '.( OO ),' ( OO ).-.             
   (_)---\_)/     '._  .-'),-----. ,--./  .--.   / . --. /,--./  .--.   / . --. /  ,--.   ,--.
   /    _ | |'--...__)( OO'  .-.  '|      |  |   | \-.  \ |      |  |   | \-.  \    \  '.'  / 
   \  :' '. '--.  .--'/   |  | |  ||  |   |  |,.-'-'  |  ||  |   |  |,.-'-'  |  | .-')     /  
    '..'''.)   |  |   \_) |  |\|  ||  |.'.|  |_)\| |_.'  ||  |.'.|  |_)\| |_.'  |(OO  \   /   
   .-._)   \   |  |     \ |  | |  ||         |   |  .-.  ||         |   |  .-.  | |   /  /\_  
   \       /   |  |      ''  '-'  '|   ,'.   |   |  | |  ||   ,'.   |   |  | |  | '-./  /.__) 
    '-----'    '--'        '-----' '--'   '--'   '--' '--''--'   '--'   '--' '--'   '--'      
			            { %s  Author:ph4ntom }
`, STOWAWAY_VERSION)
}

// ShowMainHelp 打印admin模式下的帮助
func ShowMainHelp() {
	fmt.Print(`
	help                                     		Show help information
	detail                                  		Display connected nodes' detail
	tree                                     		Display nodes' topology
	use        <id>                          		Select the target node you want to use
	exit                                     		Exit
  `)
}

// ShowNodeHelp 打印node模式下的帮助
func ShowNodeHelp() {
	fmt.Print(`
	help                                     		Show help information
	listen     [ip:]<port>                  		Start port listening on current node
	addmemo    <string>                      		Add memo for current node
	delmemo                                  		Delete memo of current node
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
	backward    <rport> <lport>               		Reflect remote port(agent) to local port(admin) (eg:reflect 22 80)
	stopbackward                                	Shut down all reflect services
	offline                                 		Terminate current node
	exit                                     		Back to upper panel
  `)
}
