![stowaway.png](https://github.com/ph4ntonn/Stowaway/blob/master/img/logo.png)

# Stowaway

[![GitHub issues](https://img.shields.io/github/issues/ph4ntonn/Stowaway)](https://github.com/ph4ntonn/Stowaway/issues)
[![GitHub forks](https://img.shields.io/github/forks/ph4ntonn/Stowaway)](https://github.com/ph4ntonn/Stowaway/network)
[![GitHub stars](https://img.shields.io/github/stars/ph4ntonn/Stowaway)](https://github.com/ph4ntonn/Stowaway/stargazers)
[![GitHub license](https://img.shields.io/github/license/ph4ntonn/Stowaway)](https://github.com/ph4ntonn/Stowaway/blob/master/LICENSE)

Stowaway is Multi-hop proxy tool for security researchers and pentesters

Users can easily proxy their network traffic to intranet nodes (multi-layer),break the restrction and manipulate all the nodes that under your control XD

PS: Thanks for everyone's star, i'm just an amateur, and the code still need be optimized,so if you find anything wrong or bugs, feel free to tell me, prs and issues are welcome :kissing_heart:. 

PPS: **Please read the usage method and the precautions at the end of the article before use!**

> This tool is limited to security research and teaching, and the user bears all legal and related responsibilities caused by the use of this tool! The author does not assume any legal and related responsibilities!

## Features

- More user-friendly interaction on the admin side, support command completion/search history
- Obvious node topology
- Clear information display of nodes
- Forward/reverse connection between nodes
- Support reconnection between nodes
- The nodes can be connected through the socks5 proxy
- The nodes can be connected through an ssh tunnel
- TCP/HTTP can be selected for inter-node traffic
- Multi-level socks5 traffic proxy forwarding, support UDP/TCP, IPV4/IPV6
- Multi-hop ssh traffic proxy
- Remote interactive shell
- Upload/download functions
- Port local/remote mapping
- Port Reuse
- Open or Close all the services arbitrarily
- Authenicate each other between nodes
- Network traffic encryption with AES-256(CBC)
- Compared with version 1.0, the file size is reduced by 25%
- Can be used on multiple platforms

## Download and Demo

- Check the [release](https://github.com/ph4ntonn/Stowaway/releases) to get one.And if you want the Uncompressed collection，check [Uncompressed](https://github.com/ph4ntonn/Stowaway/releases/download/v2.0/Uncompress_By_Upx.7z) or you can choose the Compressed collection(much more easier for u to upload agent to target server),check [Compressed](https://github.com/ph4ntonn/Stowaway/releases/download/v2.0/Compressed_By_Upx.tar)

- Demo video: Still working~

## Usage

### Character
Stowaway has two kinds of characters: 
- ```admin```  The control panel used by the penetration tester
- ```agent```  The controlled end deployed by the penetration tester

### Noun definition

- Node: refers to admin || agent
- Active mode: Refers to the currently operating node actively connecting to another node
- Passive mode: Refers to the currently operating node listening to a port and waiting for another node to connect
- Upstream: Refers to the traffic between the currently operating node and its parent node
- Downstream: Refers to the traffic between the currently operating node and **all ** child nodes

### Parameter analysis

- admin

```
parameter:
-l Listening address in passive mode [ip]:<port>
-s node communication encryption key, all nodes (admin&&agent) must be consistent
-c target node address in active mode
--proxy socks5 proxy server address
--proxyu socks5 proxy server username (optional)
--proxyp socks5 proxy server password (optional)
--down downstream protocol type, default is bare TCP traffic, optional HTTP
```

- agent

```
parameter:
-l Listening address in passive mode [ip]:<port>
-s node communication encryption key
-c target node address in active mode
--proxy socks5 proxy server address
--proxyu socks5 proxy server username (optional)
--proxyp socks5 proxy server password (optional)
--reconnect reconnect time interval
--rehost the IP address to be reused
--report the Port number to be reused
--up upstream protocol type, default is bare TCP traffic, optional HTTP
--down downstream protocol type, default is bare TCP traffic, optional HTTP
```

### Parameter usage

#### -l

This parameter can be used on admin&&agent, only can be used in passive mode 

If you do not specify an IP address, the default monitoring will be on ```0.0.0.0```

- admin:  ```./stowaway_admin -l 9999``` or ```./stowaway_admin -l 127.0.0.1:9999```

- agent:  ```./stowaway_agent -l 9999```  or ```./stowaway_agent -l 127.0.0.1:9999```

#### -s

This parameter can be used on admin&&agent, can be used in active && passive mode

Optional, if it is empty, it means that the communication is not encrypted, otherwise, the communication is encrypted based on the key given by the user

- admin:  ```./stowaway_admin -l 9999 -s 123``` 

- agent:  ```./stowaway_agent -l 9999 -s 123``` 

#### -c

This parameter  can be used on admin&&agent, only can be used in active mode 

Represents the address of the node you want to connect to

- admin:  ```./stowaway_admin -c 127.0.0.1:9999``` 

- agent:  ```./stowaway_agent -c 127.0.0.1:9999``` 

#### --proxy/--proxyu/--proxyp

These three parameters can be used on admin&&agent , only can be used in active mode

```--proxy``` represents the address of the socks5 proxy server, ```--proxyu``` and ```--proxyp``` are optional

No username and password：

- admin:  ```./stowaway_admin -c 127.0.0.1:9999 --proxy xxx.xxx.xxx.xxx```

- agent:  ```./stowaway_agent -c 127.0.0.1:9999 --proxy xxx.xxx.xxx.xxx``` 

Username and password:

- admin:  ```./stowaway_admin -c 127.0.0.1:9999 --proxy xxx.xxx.xxx.xxx --proxyu xxx --proxyp xxx```

- agent:  ```./stowaway_agent -c 127.0.0.1:9999 --proxy xxx.xxx.xxx.xxx--proxyu xxx --proxyp xxx``` 

#### --up/--down

These two parameter can be used on admin&&agent, can be used in active && passive mode

But note that there is no ```--up``` parameter on admin

These two parameters are optional. If they are empty, it means that the upstream/downstream traffic is bare TCP traffic.

If you want the upstream/downstream traffic to be HTTP traffic, just set these two parameters

- admin:  ```./stowaway_admin -c 127.0.0.1:9999 --down http``` 

- agent:  ```./stowaway_agent -c 127.0.0.1:9999 --up http```  or ```./stowaway_agent -c 127.0.0.1:9999 --up http --down http```

**Note that when you set the upstream/downstream of a node to TCP/HTTP traffic, the downstream/upstream traffic of the connected parent/child node must be set to be consistent! ! ! **

Like this:

- admin:  ```./stowaway_admin -c 127.0.0.1:9999 --down http``` 

- agent:  ```./stowaway_agent -l 9999 --up http```

In the above case, the agent must set ```--up``` to http, otherwise it will cause network errors

The same between agents

Assume agent-1 is waiting for the connection of the child node on the port ```127.0.0.1:10000```, and ```--down http``` is set

Then, agent-2 must also set ```--up``` to http, otherwise it will cause network errors

- agent-2:  ```./stowaway_agent -c 127.0.0.1:10000 --up http```

#### --reconnect

This parameter can be used on agent , only can be used in active mode

The parameter is optional. If not set, it means that the node will not actively reconnect after the network connection is disconnected. If set, it means that the node will try to reconnect to the parent node every x (the number of seconds you set) seconds.

- admin:  ```./stowaway_admin -l 9999``` 

- agent:  ```./stowaway_agent -c 127.0.0.1:9999 --reconnect 10```

In the above case, it means that if the connection between the agent and the admin is disconnected, the agent will try to reconnect back to the admin every ten seconds.

The same between agents

And ```--reconnect``` parameter can be used together with ```--proxy```/```--proxyu```/```--proxyp```. The agent will refer to its own settings at startup and try to reconnect through the proxy

#### --rehost/--report

These two parameters are quite special and can be only used on the agent side. For details, please refer to the port reuse mechanism below

## Port reuse

  Now Stowaway provide the port reuse functions based on SO_REUSEPORT和SO_REUSEADDR features and iptables rules

- In Linux environment, it can reuse most ports
- In Windows environment,it cannot reuse service port like IIS,RDP,can reuse Mysql,Apache and so on

### 复用方式

- SO_REUSEPORT/SO_REUSEADDR mode's example

  Assuming that the agent side uses port reuse mechanism to reuse port 80

  At this time, the agent must set the ```--rehost```&&```--report```&&```-s``` parameter

  - --rehost represents the IP address that you want to reuse, it cannot be 0.0.0.0, it should generally be the external address of the network card

  - --report represents the port that you want to reuse

  - -s represents communication key

  **Mainly supports port reusing on windows and mac, linux is also possible, but there are more restrictions**

  - admin：```./stowaway_admin -c 192.168.0.105:80 -s 123```
  - agent： ```./stowaway_agent  --report 80 --rehost 192.168.0.105 -s 123```


- IPTABLES mode's example

  Assuming that the agent side uses port reuse mechanism to reuse port 22

  At this time, the agent must set the ```-l```&&```--report```&&```-s``` parameter

  - -l represents the port that cannot be accessed normally, that is, the port you really want the agent to listen to and accept connections

  - --report represents the port that you want to reuse

  - -s represents communication key

  **Only supports port reusing on linux, root permission is required**

  - agent： ```./stowaway_agent --report 22 -l 10000 -s 123```

    After the agent is started, please use ```reuse.py``` in the ```script``` directory

    Set the value of SECRET (the value of SECRET is the communication key set when each node is started)

    Then execute：```python reuse.py --start --rhost xxx.xxx.xxx.xxx --rport xxx```

    - --rhost represents the address of the agent

    - --rport represents the port to be reused, in this case it should be 22

  - At this time, the admin can connect this agent：```./stowaway_admin -c 192.168.0.105:22 -s 123```

### Notice

- The above situation only lists the connection between the admin and the agent, the connection between the agent and the agent is also the same, there is no difference

- If node is killed by ctrl-c or command `kill`,it will clean up the iptables rules automatically,but if it is killed by command `kill -9`,then it can't do that and it will lead to the service originally run on the reusing port cannot be reached,so in order to avoid this situation ,the reuse.py provide the function that can stop the "port reusing" function.

  If you want to stop "port reusing",just run reuse.py like this: `python reuse.py --stop --rhost 192.168.0.105 --rport 22`

  And then the "port reusing" will be closed,and the service originally run on the reusing port can be reached again

- If you use the port reusing mode via IPTABLES , the agent will be forced to monitor at 0.0.0.0, and you cannot specify ip+port by the `-l` option

## How to get a multi-level network?

As you can see from the above example, only admin and one agent are present

Bur the multi-level network is the core of stowaway

In stowaway, the formation of a multi-level network requires the help of ```listen```, ```connect```, ```sshtunnel``` commands in admin to achieve

Here is a simple example

- admin: ```./stowaway_admin -l 9999 -s 123``` 

At this time agent-1 has connected to admin

- agent-1:  ```./stowaway_agent -c 127.0.0.1:9999 -s 123```

 If the user also wants to connect to agent-2 as follows

- agent-2:  ```./stowaway_agent -l 10000 -s 123```

Then, at this time, the user can enter ```use 0``` -> ```connect agent-2's IP:10000``` through admin to add agent-2 to the network and become a child node of agent-1

After that,If the user still wants to connect to another node agent-3 at this time too, but cannot access agent-3 through agent-1

Then, at this time, the user can enter ```use 0``` -> ```listen``` through admin -> select ```1.Normal Passive``` -> enter ```10001``` So that agent-1 listens on port 10001 and waits for the connection of child node

After the admin operation is completed, agent-3 can be started as follows

- agent-3: ```./stowaway_agent -c 127.0.0.1:10001 -s 123```

Then agent-3 can be added to the network as another child node of agent-1

For a detailed introduction of ```listen``` and ```sshtunnel```, please refer to the command analysis below

## How to reconnect?

Stowaway currently supports multiple ways of reconnection, briefly summarized as follows

First of all, when the parent node goes offline, only one kind of node will voluntarily exit, that is, the node that is in active mode at startup and has no reconnection settings.

If reconnection is set, the node will try to reconnect in the specified time interval

In addition, all nodes started in passive mode will not actively exit, but will re-monitor on the specified port based on the parameters at startup. At this time, the user can still use ```connect```, ```sshtunnel``` To connect these nodes back to the network

## Some points you should know

1. **If a branch is disconnected due to network fluctuations or an intermediate node is disconnected, please be sure to connect to the head node of the missing chain when actively reconnecting **
   **For example, node1 is followed by admin, and node1 is divided into two branches, one is node1->node 2 -> node 3 -> node 4, and the other is node1->node 5 ->node 6, then if If node2 is offline, node3 and node4 will not be offline, but will continue to survive. At this time, if the user wants to rejoin node3 and node4 to the network, the user has two choices. One is that if node1 can directly access node3, then the user can rejoin the network at any time using the connect or sshtunnel command on node1 (remember, even if node1 can also access node4 at the same time, please do not directly connect to node4, please connect the head node node3 of the entire missing chain (node3->node4), so that you can rejoin node3 and node4 to the network; another option is when node1 cannot access node3 directly (that means, you must go through node2), then please restart node2 and join the network, and then use the connect or sshtunnel command on node2 to connect to node3, thereby adding node3 and node4 to the network. **
2. **When a node is offline, all ```socks```, ```backward```, and ```forward``` services related to this node and its child nodes will be forcibly stopped**

## Command analysis

In the admin console, users can use tabs to complete commands, and use the arrow keys to search history/move the cursor

The admin console is divided into two levels, the first level is the main panel, and the commands included are as follows

- help: Display the help information of the main panel

```
(admin) >> help
  help                                     		Show help information
  detail                                  		Display connected nodes' detail
  topo                                     		Display nodes' topology
  use        <id>                          		Select the target node you want to use
  exit                                     		Exit
```

- detail: Show detailed information of online nodes

```
(admin) >> detail
Node[0] -> IP: 127.0.0.1:10000  Hostname: ph4ntoms-MBP.lan  User: ph4ntom
Memo:
```

- topo: Show the parent-child relationship of online nodes

```
(admin) >> topo
Node[0]'s children ->
Node[1]

Node[1]'s children ->
```

- use: Use an agent

```
(admin) >> use 0
(node 0) >>
```

- exit: Exit stowaway

```
(admin) >> exit
[*] BYE!
```

When the user selects an agent using the ```use``` command, he enters the second layer: node panel, which contains the following commands

- help: Display the help information of the node panel

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
  offline                                         Terminate current node
  exit                                            Back to upper panel
```

- listen: Ask the agent to monitor a certain port and wait for the connection of the child node

```
(node 0) >> listen
[*] MENTION! If you choose IPTables Reuse or SOReuse,you MUST CONFIRM that the node was initially started in the corresponding way!
[*] When you choose IPTables Reuse or SOReuse, the node will use the initial config(when node started) to reuse port!
[*] Please choose the mode(1.Normal passive / 2.IPTables Reuse / 3.SOReuse): 1
[*] Please input the [ip:]<port> : 10001
[*] Waiting for response......
[*] Node is listening on 10001
```

Note that listen is a special command. As you can see, the listen command has three modes

1. Normal passive: This option means that the agent will listen on the target port in a normal way and wait for the child node to connect
2. IPTables Reuse：This option means that the agent will reuse the port in the way of IPTables Reuse and wait for the child node to connect
3. SOReuse：This option means that the agent will reuse the port in SOReuse mode and wait for the child node to connect

The first mode is the most commonly used. If the parent node listens in this way, then the child node only needs ```-c parent node ip:port`'' to join the network

The second and third modes are quite special. If the user chooses the second or third mode, the user must ensure that the currently operating node itself is started by port reusing, otherwise these two cannot be used

The second and third modes will not require the user to input any information, the node will automatically use its own startup parameters to reuse the port, and prepare to accept the connection of the child node

In addition, listen can only accept the connection of one child node at a time. If multiple child nodes are required to connect, please execute the listen command for the corresponding number of times

- addmemo: Add a memo for the current node

```
(node 0) >> addmemo test
[*] Memo added!
(node 0) >> exit
(admin) >> detail
Node[0] -> IP: 127.0.0.1:10000  Hostname: ph4ntoms-MBP.lan  User: ph4ntom
Memo:  test
```

- delmemo: Delete the memo of the current node

```
(node 0) >> delmemo
[*] Memo deleted!
(node 0) >> exit
(admin) >> detail
Node[0] -> IP: 127.0.0.1:10000  Hostname: ph4ntoms-MBP.lan  User: ph4ntom
Memo:
```

- ssh: Ask the node to connect to the target machine in ssh mode and open the terminal

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

In this mode, the tab key will be available and the arrow keys will be forbidden

- shell: Get the shell of the current node

```
(node 0) >> shell
[*] Waiting for response.....
[*] MENTION!UNDER SHELL MODE, ARROW UP/DOWN/LEFT/RIGHT ARE ALL ABANDONED!
[*] Shell is started successfully!

bash: no job control in this shell

The default interactive shell is now zsh.
To update your account to use zsh, please run `chsh -s /bin/zsh`.
For more details, please visit https://support.apple.com/kb/HT208050.
bash-3.2$ whoami
ph4ntom
bash-3.2$
```

In this mode, the tab key will be available and the arrow keys will be forbidden

- socks：Start the socks5 service on the current node

```
(node 0) >> socks 7777
[*] Trying to listen on 0.0.0.0:7777......
[*] Waiting for response......
[*] Socks start successfully!
(node 0) >>
```

Note that the port 7777 here is not opened on the agent, but on the admin

In addition, if you need to set a username and password, you can change the above command to ```socks 7777 <your username> <your password>```

- stopsocks: Stop the socks5 service on the current node

```
(node 0) >> stopsocks
Socks Info ---> ListenAddr: 0.0.0.0:7777    Username: <null>    Password: <null>
[*] Do you really want to shutdown socks?(yes/no): yes
[*] Closing......
[*] Socks service has been closed successfully!
(node 0) >>
```

- connect: Ask the current node to connect to another child node

```
agent-1: ./stowaway_agent -l 10002
```

```
(node 0) >> connect 127.0.0.1:10002
[*] Waiting for response......
[*] New node come! Node id is 1

(node 0) >>
```

- sshtunnel: Ask the current node to connect to another child node through an ssh tunnel

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

In a strictly restricted network environment, ssh tunnels can be used to disguise stowaway traffic as ssh traffic to avoid firewall restrictions

- upload: Upload files to the current node

```
(node 0) >> upload test.7z test.xxx
[*] File transmitting, please wait...
136.07 KiB / 136.07 KiB [-----------------------------------------------------------------------------------] 100.00% ? p/s 0s
```

- download: Download files from the current node

```
(node 0) >> download test.xxx test.xxxx
[*] File transmitting, please wait...
136.07 KiB / 136.07 KiB [-----------------------------------------------------------------------------------] 100.00% ? p/s 0s
```

- forward: Map the port on the admin to the remote port

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

- stopforward: Close the remote mapping of the current node

```
(node 0) >> stopforward
[0] All
[1] Listening Addr : [::]:9000 , Remote Addr : 127.0.0.1:22 , Current Active Connnections : 1
[*] Do you really want to shutdown forward?(yes/no): yes
[*] Please choose one to close: 1
[*] Closing......
[*] Forward service has been closed successfully!
```

- backward: Reverse mapping the port on the current agent to the local port of the admin

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

- stopbackward: Turn off the reverse mapping of the current node

```
(node 0) >> stopbackward
[0] All
[1] Remote Port : 9001 , Local Port : 22 , Current Active Connnections : 1
[*] Do you really want to shutdown backward?(yes/no): yes
[*] Please choose one to close: 1
[*] Closing......
[*] Backward service has been closed successfully!
```

- offline: Ask the current node to go offline

```
(node 1) >> offline
(node 1) >>
[*] Node 1 is offline!
(node 1) >>
```

- exit: Return to the main panel

```
(node 1) >> exit
(admin) >>
```

## TODO

- [ ] Enhance the robustness of the program

### 注意事项

- This porject is coding just for fun , the logic structure and code structure are not strict enough, please don't be so serious about it
- Admin node MUST be online when new node is being added into the network
- The admin only supports one directly connected agent node, and the agent node has no such restriction
- If users use the admin on windows, please download [ansicon](https://github.com/adoxa/ansicon/releases) first, or download from [here](), and then enter the folder corresponding to the number of system bits. Execute ```ansicon.exe -i```, otherwise garbled characters will appear on the admin
- This program only supports standard UDP ASSOCIATE (that supports UDP proxy) described in [RFC1928](https://www.ietf.org/rfc/rfc1928.txt), please pay attention to what you are using when using socks5 udp proxy Programs (such as scanners, etc.), the packet construction method must comply with the standard [RFC1928](https://www.ietf.org/rfc/rfc1928.txt), and the packet loss situation needs to be handled by yourself.

### Thanks

- [rootkiter#Termite](https://github.com/rootkiter/Termite)
- [Venom](https://github.com/Dliv3/Venom)