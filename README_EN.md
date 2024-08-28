![stowaway.png](https://github.com/ph4ntonn/Stowaway/blob/master/img/logo.png)

# Stowaway

[![GitHub issues](https://img.shields.io/github/issues/ph4ntonn/Stowaway)](https://github.com/ph4ntonn/Stowaway/issues)
[![GitHub forks](https://img.shields.io/github/forks/ph4ntonn/Stowaway)](https://github.com/ph4ntonn/Stowaway/network)
[![GitHub stars](https://img.shields.io/github/stars/ph4ntonn/Stowaway)](https://github.com/ph4ntonn/Stowaway/stargazers)
[![GitHub license](https://img.shields.io/github/license/ph4ntonn/Stowaway)](https://github.com/ph4ntonn/Stowaway/blob/master/LICENSE)

Stowaway is a Multi-hop proxy tool for security researchers and pentesters

Users can use this program to proxy external traffic through multiple nodes to the core internal network, breaking through internal network access restrictions, constructing a tree-like node network, and easily realizing management functions.

Thank you everyone for the stars, and also welcome everyone to raise questions && bugs after use :kissing_heart: 

**And please be sure to read the usage method and the notes at the end before using**

> This tool is limited to security research and teaching, and the user bears all legal and related responsibilities caused by the use of this tool! The author does not assume any legal and related responsibilities!

## Features

- More user-friendly interaction, support command auto-completion/search history
- Obvious node topology
- Clear information display of nodes
- Active/Passive connection between nodes
- Support reconnection between nodes
- Nodes can be connected through socks5/http proxy
- Nodes can be connected through ssh tunnel
- TCP/HTTP/WS can be selected for inter-node traffic
- Multi-hop socks5 traffic proxy forwarding, support UDP/TCP, IPV4/IPV6
- Nodes can access arbitrary host via ssh
- Remote shell
- Upload/download files
- Port local/remote mapping
- Port Reuse
- Open/Close all the services arbitrarily
- Authenicate each other between nodes
- Traffic encryption with TLS/AES-256-GCM
- Compared with v1.0, the file size is reduced by 25%
- Multiple platforms support(Linux/Mac/Windows/MIPS/ARM)

## Build and Demo

- Use `make` to directly compile programs for multiple platforms, or refer to the Makefile for compiling specific programs.
- If you prefer not to compile, you can directly use the programs available here [release](https://github.com/ph4ntonn/Stowaway/releases) 
- Demo video: [Youtube](https://www.youtube.com/watch?v=Lh5Q0RPWKMU&list=PLkbGxnHFIhA_g5XZtKzN4u-JXRq41L2g-)

## Usage

### Character

Stowaway has two kinds of characters: 
- `admin`  The master used by the penetration tester
- `agent`  The slave deployed by the penetration tester

### Noun definition

- Node: This refers to admin || agent
- Active mode: This refers to the situation where the node being currently operated actively establishes a connection to another node
- Passive mode: This refers to the situation where the node currently being operated is listening on a specific port, waiting for another node to connect.
- Upstream: This refers to the traffic between the node currently being operated and its parent node
- Downstream: This refers to the traffic between the node currently being operated and **all ** its child nodes

### Quick start

The following command can quickly start the simplest stowaway instance

- admin: `./stowaway_admin -l 9999`
- agent: `./stowaway_agent -c <stowaway_admin's IP>:9999`

### Parameter analysis

- admin

```
Parameter:
-l Listening address in passive mode [ip]:<port>
-s Node communication encryption key, all nodes (admin&&agent) must be consistent
-c Target node address under active mode
--socks5-proxy SOCKS5 proxy server address
--socks5-proxyu SOCKS5 proxy server username
--socks5-proxyp SOCKS5 proxy server password
--http-proxy HTTP proxy server address
--down Downstream protocol type, default is raw TCP traffic, optional HTTP/WS
--tls-enable Enable TLS for node communication, after enabling TLS, AES encryption will be disabled
--domain Specify the TLS SNI/WebSocket domain name. If it is empty, it defaults to the target node address
--heartbeat Enable heartbeat 
```

- agent

```
Parameter:
-l Listening address in passive mode [ip]:<port>
-s Node communication encryption key
-c Target node address under active mode
--socks5-proxy SOCKS5 proxy server address
--socks5-proxyu SOCKS5 proxy server username (optional)
--socks5-proxyp SOCKS5 proxy server password (optional)
--http-proxy HTTP proxy server address
--reconnect Reconnect time interval
--rehost The IP address to be reused
--report The Port number to be reused
--up Upstream protocol type, default is raw TCP traffic, optional HTTP/WS
--down Downstream protocol type, default is raw TCP traffic, optional HTTP/WS
--cs Platform's console encoding type,default is utf-8，optional gbk
--tls-enable Enable TLS for node communication, after enabling TLS, AES encryption will be disabled
--domain Specify the TLS SNI/Websocket domain name. If it is empty, it defaults to the target node address.
```

### Parameter usage

#### -l

This parameter can be used on admin&&agent, under passive mode 

If you do not specify an IP address, it will default to listening on `0.0.0.0`

- admin:  `./stowaway_admin -l 9999` or `./stowaway_admin -l 127.0.0.1:9999`

- agent:  `./stowaway_agent -l 9999`  or `./stowaway_agent -l 127.0.0.1:9999`

#### -s

This parameter can be used on admin&&agent, under both active && passive mode

This parameter is optional. If it is left blank, it means that the communication will not be encrypted. Conversely, if a key is provided by the user, the communication will be encrypted based on that key.

- admin:  `./stowaway_admin -l 9999 -s 123`

- agent:  `./stowaway_agent -l 9999 -s 123`

#### -c

This parameter can be used on admin&&agent, under active mode 

It represents the address of the node you wish to connect to

- admin:  `./stowaway_admin -c 127.0.0.1:9999`

- agent:  `./stowaway_agent -c 127.0.0.1:9999` 

#### --socks5-proxy/--socks5-proxyu/--socks5-proxyp/--http-proxy

These four parameters can be used on admin&&agent , under active mode

`--socks5-proxy` represents the address of the socks5 proxy server, `--socks5-proxyu` and `--socks5-proxyp` are optional

`--http-proxy` represents the address of the http-proxy server, the usage is as same as socks5

No username and password：

- admin:  `./stowaway_admin -c 127.0.0.1:9999 --socks5-proxy xxx.xxx.xxx.xxx`

- agent:  `./stowaway_agent -c 127.0.0.1:9999 --socks5-proxy xxx.xxx.xxx.xxx`

Require username and password:

- admin:  `./stowaway_admin -c 127.0.0.1:9999 --socks5-proxy xxx.xxx.xxx.xxx --socks5-proxyu xxx --socks5-proxyp xxx`

- agent:  `./stowaway_agent -c 127.0.0.1:9999 --socks5-proxy xxx.xxx.xxx.xxx --socks5-proxyu xxx --socks5-proxyp xxx`

#### --up/--down

These two parameter can be used on admin&&agent, under active && passive mode

However, note that there is no `--up` parameter on the admin

These two parameters are optional. If left empty, it signifies that the upstream/downstream traffic will be in the form of raw TCP traffic

If you wish for the upstream/downstream traffic to be HTTP/WS traffic, simply set these two parameters to `http` or `ws`

- admin:  `./stowaway_admin -c 127.0.0.1:9999 --down ws` 

- agent:  `./stowaway_agent -c 127.0.0.1:9999 --up ws`  or `./stowaway_agent -c 127.0.0.1:9999 --up ws --down ws`

There are two other points to note:

First, once you set the upstream/downstream traffic of a particular node to TCP/HTTP/WS, the downstream/upstream traffic of its connected parent/child node must be set consistently

Like this:

- admin:  `./stowaway_admin -c 127.0.0.1:9999 --down ws`

- agent:  `./stowaway_agent -l 9999 --up ws`

In the above case, the agent must set `--up` to ws, otherwise it will cause network errors

The rules between admin<-->agent is as same as agent<-->agent

Assuming agent-1 is waiting for the connection of child nodes on the port `127.0.0.1:10000` and has set `--down ws`

Then, agent-2 must also set `--up` to `ws`, otherwise, it would lead to network errors

- agent-2:  `./stowaway_agent -c 127.0.0.1:10000 --up ws`

Second, since HTTP is a half-duplex protocol, it is not very suitable for the full-duplex communication nature of Stowaway. Therefore, the HTTP protocol here only implements the HTTP message format, not a fully functional HTTP workflow. So you can still use this protocol, but the traffic between Stowaway cannot be forwarded by nginx when choosing to transmit in the HTTP message format. This part of the code and function is retained on the one hand for the use of the HTTP protocol in some special cases, and on the other hand to provide a template for custom traffic, which is convenient for users to use as a reference when customizing other protocols.

If you need to use reverse proxy services such as nginx, please use the Websocket (ws) protocol for communication(it would be better if it can be used with tls).

#### --reconnect

This parameter can be used on agent, under active mode

This parameter is optional. If not set, it means that the node will not automatically attempt to reconnect after a network disconnection. If set, it indicates that the node will try to reconnect to the parent node every x seconds (the number of seconds you set)

- admin:  `./stowaway_admin -l 9999`

- agent:  `./stowaway_agent -c 127.0.0.1:9999 --reconnect 10`

In the scenario described above, it means that if the connection between the agent and the admin is interrupted, the agent will attempt to reconnect to the admin every ten seconds

The rules between admin<-->agent is as same as agent<-->agent

Additionally, the `--reconnect` parameter can be used in conjunction with `--socks5-proxy`, `--socks5-proxyu`, `--socks5-proxyp`, or `--http-proxy`. In such cases, the agent will attempt to reconnect through the proxy according to the settings specified at startup

#### --rehost/--report

These two parameters are quite unique and are used exclusively on the agent side. For more details, please refer to the information below on the port reuse mechanism

#### --cs

This parameter can be used on agent, under active && passive mode

This is primarily aimed at resolving the issue of garbled text with the 'shell' function. When the agent is operated on a platform where the console encoding is set to GBK (such as is commonly the case with Windows) and concurrently, the admin is run on a platform with UTF-8 console encoding, it is crucial to set this parameter to `gbk`

- Windows: `./stowaway_agent -c 127.0.0.1:9999 -s 123 --cs gbk`

#### --tls-enable

These two parameter can be used on admin&&agent, under active && passive mode

By setting this option, traffic between nodes can be encrypted with TLS

- admin: `./stowaway_admin -l 10000 --tls-enable -s 123`
- agent: `./stowaway_agent -c localhost:10000 --tls-enable -s 123`

Please note that when this parameter is enabled, AES encryption will be disabled by default. The `-s` parameter will then be used solely for mutual verification between nodes & port reuse functionality

Additionally, when this parameter is enabled, **ensure that every node in the network (including the admin) has this parameter enabled**

#### --domain

These two parameter can be used on admin&&agent, under active mode

By setting this option, you can specify the SNI option for TLS negotiation or WebSocket target host for the current node

- admin: `./stowaway_admin -l 10000 --tls-enable -s 123`
- agent: `./stowaway_agent -c xxx.xxx.xxx.xxx:10000 --tls-enable -s 123 --domain xxx.com`

#### --heartbeat

This parameter can be used on admin, under active && passive mode

By setting this option, it allows the admin to continuously send heartbeat packets to the first node, thus maintaining a persistent connection even in the presence of a reverse proxy in between.

Assuming there are reverse proxy devices similar to NGINX between the admin and agent, proxying port 8080 to port 8000, an example is as follows:
- admin: `./stowaway_admin -l 8000 --tls-enable -s 123 --down ws --heartbeat`
- agent: `./stowaway_agent -c xxx.xxx.xxx.xxx:8080 --tls-enable -s 123 --domain xxx.com --up ws`

## Port reuse

Stowaway currently supports port reuse functionality based on the SO_REUSEPORT and SO_REUSEADDR features, as well as port reuse functionality based on IPTABLES

- In Linux environment, stowaway can reuse most ports
- In Windows environment, it cannot reuse service port like IIS/RDP, but can reuse Mysql/Apache and so on

### How To?

- SO_REUSEPORT/SO_REUSEADDR

  Assuming that the agent uses the port reuse mechanism to reuse port 80

  In order to do this, the agent must set the `--rehost`&&`--report`&&`-s` parameter at startup

  - `--rehost` This represents the desired IP address for reuse, which cannot be `0.0.0.0` and generally should be the external address of the network card

  - `--report` This represents the desired port for reuse

  - `-s` This represents communication key

  **This mode mainly supports windows and mac, linux is also possible, but there are more restrictions**

  - admin：`./stowaway_admin -c 192.168.0.105:80 -s 123`
  - agent： `./stowaway_agent  --report 80 --rehost 192.168.0.105 -s 123`

- IPTABLES

  Assuming that the agent uses the port reuse mechanism to reuse port 22

  In order to do this, the agent must set the `-l`&&`--report`&&`-s` parameter at startup

  - `-l` This represents the port that cannot be accessed normally, meaning the port you actually want the agent to listen on and accept connections

  - `--report` This represents the desired port for reuse

  - `-s` This represents communication key

  **This method only support linux, agent will manipulate IPTABLES automatically,root permission is required**

  - agent： `./stowaway_agent --report 22 -l 10000 -s 123`

    After the agent has started, please use the `reuse.py` script located in the `script` directory first

    Set the value of SECRET (the value of SECRET is the communication key, aka -s option)

    Then execute：`python reuse.py --start --rhost xxx.xxx.xxx.xxx --rport 22`

    - `--rhost` This represents the address of the agent

    - `--rport` This represents the port being reused, which in this example should be 22.

  - At this time, the admin can connect this agent：`./stowaway_admin -c xxx.xxx.xxx.xxx:22 -s 123`

### Notice

- The scenarios mentioned above only enumerate the connection between the admin and the agent. The connection between agents also follows the same principles and has no differences.

- If the agent is terminated using `ctrl-c` or the `kill` command, the program will automatically clean up the iptables rules. However, if it is terminated using `kill -9`, it cannot be automatically cleaned up. Therefore, to prevent the iptables rules from not being cleaned up when the agent exits abnormally, resulting in the inability to access the reused service. So, when you need to shut down "port reusing", you should run: `python reuse.py --stop --rhost xxx.xxx.xxx.xxx --rport xxx`. This will disable the forwarding rules, allowing the original service to be accessed normally.

- If using the IPTABLES mode for port reuse, it will forcefully listen on `0.0.0.0`, and cannot be specified using the `-l` parameter.

## How to build a multi-level network?

From the example above, it can be observed that only the admin and one agent are involved

However, constructing a multi-level network is the core functionality of Stowaway

In Stowaway, building a multi-level network requires the use of commands such as `listen`, `connect`, and `sshtunnel` within the admin interface

Here is an example

- admin: `./stowaway_admin -l 9999 -s 123` 

At this point, agent-1 has already connected to the admin

- agent-1:  `./stowaway_agent -c 127.0.0.1:9999 -s 123`

And, the user also wishes to connect to agent-2, as follows:

- agent-2:  `./stowaway_agent -l 10000 -s 123`

Then, the user can enter `use 0` -> `connect agent-2's IP:10000` via admin to add agent-2 to the network and become a child node of agent-1

After that,If the user wishes to connect to another node, agent-3, but cannot access it through agent-1

Then, in order to solve this problem, the user can enter `use 0` -> `listen` via admin -> select `1.Normal Passive` -> enter `10001` to have agent-1 listen on port 10001 and await connections from agent-3

After the admin operation is completed, agent-3 can be started as follows

- agent-3: `./stowaway_agent -c 127.0.0.1:10001 -s 123`

At this point, agent-3 has become another child node of agent-1 and joined the network.

For a detailed explanation of the `listen` and `sshtunnel` commands, please refer to the command analysis below.

## How to reconnect?

Stowaway currently supports various methods of reconnection, summarized briefly as follows:

Firstly, when the parent node goes offline, only one type of node will actively disconnect, namely the nodes that were started in active mode and have not set up reconnection.

If reconnection is set up, the node will attempt to reconnect at specified time intervals.

Additionally, all nodes started in passive mode will not actively disconnect. Instead, they will re-listen on the specified port based on the parameters set at startup. Users can still use `connect` and `sshtunnel` to reconnect these nodes to the network.

## Some points you should know

1. **If a branch disconnects due to network fluctuations or a middle node going offline, when attempting to reconnect, it is essential to connect to the head node of the missing branch.** For example, after the admin, there is node1. Node1 branches into two: one branch is node1 -> node 2 -> node 3 -> node 4, and the other branch is node1 -> node 5 -> node 6. If node2 goes offline, node3 and node4 will remain active. If the user wishes to reconnect node3 and node4 to the network, there are two options available. Firstly, if node1 can directly access node3, the user can reconnect node3 to the network at any time by using the `connect` or `sshtunnel` commands on node1. It's important to note that even if node1 can also access node4 directly, please do not connect to node4 directly. Instead, connect to the head node of the missing chain (node3->node4), which is node3. This way, both node3 and node4 can be reconnected to the network. Alternatively, if node1 cannot directly access node3 (i.e., it must go through node2), please restart node2 and join it to the network first. Then, on node2, use the `connect` or `sshtunnel` commands to connect to node3, thus reconnecting both node3 and node4 to the network.
2. **When a node is offline, all `socks`, `backward`, and `forward` services related to this node and its child nodes will be forcibly stopped**

## Command analysis

In the admin console, users can utilize the tab key for command completion and the arrow keys (up, down, left, right) to navigate through command history or move the cursor

The admin console is divided into two levels. The first level is the main panel, which includes the following commands:

- `help`: Display help information for the main panel

```
(admin) >> help
  help                                     		Show help information
  detail                                  		Display connected nodes' detail
  topo                                     		Display nodes' topology
  use        <id>                          		Select the target node you want to use
  exit                                     		Exit Stowaway
```

- `detail`: Display detailed information about online nodes

```
(admin) >> detail
Node[0] -> IP: 127.0.0.1:10000  Hostname: ph4ntoms-MBP.lan  User: ph4ntom
Memo:
```

- `topo`: Display the parent-child relationships of online nodes

```
(admin) >> topo
Node[0]'s children ->
Node[1]

Node[1]'s children ->
```

- `use`: Select an node

```
(admin) >> use 0
(node 0) >>
```

- `exit`: Exit stowaway

```
(admin) >> exit
[*] Do you really want to exit stowaway?(y/n): y
[*] BYE!
```

When user selects an node via `use` command, admin will enter the second level: node panel, it includes the following commands:

- `help`: Display the help information for the node panel

```
(node 0) >> help
  help                                            Show help information
  listen                                          Start port listening on current node
  addmemo    <string>                             Add memo for current node
  delmemo                                         Delete memo of current node
  ssh        <ip:port>                            Start SSH through current node
  shell                                           Start an interactive shell on current node
  socks      <lport> [username] [pass]            Start a socks5 server
  stopsocks                                       Shut down socks services
  connect    <ip:port>                            Connect to a new node
  sshtunnel  <ip:sshport> <agent port>            Use sshtunnel to add the node into our topology
  upload     <local filename> <remote filename>   Upload file to current node
  download   <remote filename> <local filename>   Download file from current node
  forward    <lport> <ip:port>                    Forward local port to specific remote ip:port
  stopforward                                     Shut down forward services
  backward    <rport> <lport>                     Backward remote port(agent) to local port(admin)
  stopbackward                                    Shut down backward services
  shutdwon                                        Terminate current node
  back                                            Back to parent panel
  exit                                            Exit Stowaway 
```

- `listen`: Instruct the node to listen on a specific port and wait for connection from child node

```
(node 0) >> listen
[*] MENTION! If you choose IPTables Reuse or SOReuse,you MUST CONFIRM that the node was initially started in the corresponding way!
[*] When you choose IPTables Reuse or SOReuse, the node will use the initial config(when node started) to reuse port!
[*] Please choose the mode(1.Normal passive / 2.IPTables Reuse / 3.SOReuse): 1
[*] Please input the [ip:]<port> : 10001
[*] Waiting for response......
[*] Node is listening on 10001
```

Note that `listen` is a special command. As you can see, the `listen` command has three modes

1. `Normal passive`: This option implies that the agent will listen on the target port in a normal way and wait for child nodes to connect.
2. `IPTables Reuse`：This option implies that the agent will reuse the port using IPTables and wait for child nodes to connect.
3. `SOReuse`：This option implies that the agent will reuse the port using SOReuse and wait for child nodes to connect.

The first mode is the most commonly used. If the parent node is listening in this way, child nodes only need to use `-c parent_node_ip:port` to join the network.

The second and third modes are rather unique. If the user selects the second or third mode, they must ensure that the node they are currently operating on has been started using port reuse. Otherwise, these two modes cannot be used.

In the second and third modes, users won't need to input any information. The node will automatically reuse the port using the parameters set at its own startup and prepare to accept connections from child nodes.

Furthermore, the `listen` command can only accept one child node connection at a time. If multiple child nodes need to connect, please execute the `listen` command the corresponding number of times.

- `addmemo`: Add a memo for the current node

```
(node 0) >> addmemo test
[*] Memo added!
(node 0) >> exit
(admin) >> detail
Node[0] -> IP: 127.0.0.1:10000  Hostname: ph4ntoms-MBP.lan  User: ph4ntom
Memo:  test
```

- `delmemo`: Delete the memo of the current node

```
(node 0) >> delmemo
[*] Memo deleted!
(node 0) >> exit
(admin) >> detail
Node[0] -> IP: 127.0.0.1:10000  Hostname: ph4ntoms-MBP.lan  User: ph4ntom
Memo:
```

- `ssh`: Instruct the node to connect to the target host via SSH.

```
(node 0) >> ssh 127.0.0.1:22
[*] Please choose the auth method(1.username&&password / 2.certificate): 1
[*] Please enter the username: ph4ntom
[*] Please enter the password: *****
[*] Waiting for response.....
[*] Connect to target host via ssh successfully!
 # ph4ntom @ ph4ntoms-MBP in ~ 👑 [17:03:56]
$ whoami
ph4ntom
 # ph4ntom @ ph4ntoms-MBP in ~ 👑 [17:04:16]
$
```

Under this mode, the tab key will be disabled

- `shell`: Get the shell of the current node

```
(node 0) >> shell
[*] Waiting for response.....
[*] Shell is started successfully!

bash: no job control in this shell

The default interactive shell is now zsh.
To update your account to use zsh, please run `chsh -s /bin/zsh`.
For more details, please visit https://support.apple.com/kb/HT208050.
bash-3.2$ whoami
ph4ntom
bash-3.2$
```

Under this mode, the tab key will be disabled

- `socks`：Start the socks5 service on the current node

```
(node 0) >> socks 7777
[*] Trying to listen on 0.0.0.0:7777......
[*] Waiting for response......
[*] Socks start successfully!
(node 0) >>
```

Please note that the port 7777 is not opened on the agent, but rather on the admin

If you need to set a username and password, you can modify the above command to `socks 7777 <your username> <your password>`

If you need to specify the interface to listen on, you can modify the above command to `socks xxx.xxx.xxx.xxx:7777`

- `stopsocks`: Stop the SOCKS5 service on the current node

```
(node 0) >> stopsocks
Socks Info ---> ListenAddr: 0.0.0.0:7777    Username: <null>    Password: <null>
[*] Do you really want to shutdown socks?(yes/no): yes
[*] Closing......
[*] Socks service has been closed successfully!
(node 0) >>
```

- `connect`: Instruct the current node to connect to another child node

```
agent-1: ./stowaway_agent -l 10002
```

```
(node 0) >> connect 127.0.0.1:10002
[*] Waiting for response......
[*] New node come! Node id is 1

(node 0) >>
```

- `sshtunnel`: Instruct the current node to connect to another child node via ssh tunnel

```
agent-2: ./stowaway_agent -l 10003
```

```
(node 0) >> sshtunnel 127.0.0.1:22 10003
[*] Please choose the auth method(1.username&&password / 2.certificate): 1
[*] Please enter the username: ph4ntom
[*] Please enter the password: ******
[*] Waiting for response.....
[*] New node come! Node id is 2

(node 0) >>
```

In highly restricted network environments, Stowaway can utilize SSH tunneling to disguise its traffic as SSH traffic, thereby circumventing firewall restrictions

- `upload`: Upload file to the current node

```
(node 0) >> upload test.7z test.xxx
[*] File transmitting, please wait...
136.07 KiB / 136.07 KiB [-----------------------------------------------------------------------------------] 100.00% ? p/s 0s
```

- `download`: Download file from the current node

```
(node 0) >> download test.xxx test.xxxx
[*] File transmitting, please wait...
136.07 KiB / 136.07 KiB [-----------------------------------------------------------------------------------] 100.00% ? p/s 0s
```

- `forward`: Map port on the admin to remote port

```
(node 0) >> forward 9000 127.0.0.1:22
[*] Trying to listen on 0.0.0.0:9000......
[*] Waiting for response......
[*] Forward start successfully!
(node 0) >>
```

```
$ ssh 127.0.0.1 -p 9000
Password:
 # ph4ntom @ ph4ntoms-MBP in ~ 👑 [17:19:51]
$
```

- `stopforward`: Close the remote mapping on the admin

```
(node 0) >> stopforward
[0] All
[1] Listening Addr : [::]:9000 , Remote Addr : 127.0.0.1:22 , Current Active Connnections : 1
[*] Do you really want to shutdown forward?(yes/no): yes
[*] Please choose one to close: 1
[*] Closing......
[*] Forward service has been closed successfully!
```

- `backward`: Reverse map the port on the current agent to the local port on the admin

```
(node 0) >> backward 9001 22
[*] Trying to ask node to listen on 0.0.0.0:9001......
[*] Waiting for response......
[*] Backward start successfully!
(node 0) >>
```

```
$ ssh 127.0.0.1 -p 9001
Password:
 # ph4ntom @ ph4ntoms-MBP in ~ 🌈 [17:22:14]
$
```

- `stopbackward`: Close the reverse mapping on the current node

```
(node 0) >> stopbackward
[0] All
[1] Remote Port : 9001 , Local Port : 22 , Current Active Connnections : 1
[*] Do you really want to shutdown backward?(yes/no): yes
[*] Please choose one to close: 1
[*] Closing......
[*] Backward service has been closed successfully!
```

- `shutdown`: Shutdown the current node 

```
(node 1) >> shutdown
(node 1) >>
[*] Node 1 is offline!
```

- `back`: Return to main panel

```
(node 1) >> back
(admin) >>
```

- `exit`: Exit Stowaway 

```
(node 1) >> exit
[*] Do you really want to exit stowaway?(y/n): y
[*] BYE!
```

## TODO

- [x] Fix the bug that may exists
- [x] Support TLS
- [ ] Support multi startnode

### Attention

- This porject is coding just for fun , the logic structure and code structure are not strict enough, please don't be so serious about it
- Admin node MUST be online if you want to add a new node into the network
- The admin only supports one directly connected agent node, but the agent node has no such restriction
- If users use the admin on windows, please download [ansicon](https://github.com/adoxa/ansicon/releases) first, or download from [here](), and then enter the folder corresponding to the number of system bits. Execute `ansicon.exe -i`, otherwise garbled characters will appear on the admin
- This program only supports standard `UDP ASSOCIATE` described in [RFC1928](https://www.ietf.org/rfc/rfc1928.txt). Please check the programs(such as scanners, etc.) you are using, make sure if the packet construction method comply with the standard [RFC1928](https://www.ietf.org/rfc/rfc1928.txt). Besides, the packet loss situation also needs to be handled by yourself.

## 404Starlink

<img src="https://github.com/knownsec/404StarLink/raw/master/Images/logo.png" width="30%">

Stowaway has joined [404Starlink](https://github.com/knownsec/404StarLink)

## Acknowledgement

Thanks to the following developers and projects for their help during the development of Stowaway
- [lz520520](https://github.com/lz520520)
- [SignorMercurio](https://github.com/SignorMercurio)
- [MM0x00](https://github.com/MM0x00)
- [r0ck3rt](https://github.com/r0ck3rt)
- [Termite](https://github.com/rootkiter/Termite)
- [Venom](https://github.com/Dliv3/Venom)