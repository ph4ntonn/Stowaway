# Stowaway

Stowaway is Multi-hop proxy tool for security researchers and pentesters

Users can easily proxy their network traffic to intranet nodes (multi-layer)

PS: Thanks for everyone's star, i'm just an amateur, and the code still need be optimized,so if you find anything wrong or bugs, feel free to tell me, and i'll fix it :kissing_heart:. 
> This tool is limited to security research and teaching, and the user bears all legal and related responsibilities caused by the use of this tool! The author does not assume any legal and related responsibilities!

## Features

- Obvious node topology
- Active/passive mode
- Ssh tunnel mode
- Can be used on multiple platforms
- Multi-hop socks5 traffic proxy
- Multi-hop ssh traffic proxy
- Remote interactive shell
- Upload/download functions
- Port Mapping(local to remote/remote to local)
- Port Reuse
- Network traffic encryption with AES-256(CBC mode)


## Usage

Stowaway can be excuted as two kinds of mode: admin && agent(including startnode && simple node,startnode means the first node that connecting to admin,it's different from simple nodes)


If you don't want to compile the project by yourself, you can check the [release](https://github.com/ph4ntonn/Stowaway/releases) tag to get ONE!（Compressed and Uncompressed files are provided, choose the one you like)

Example 1：Admin node waiting the connection from startnode
```
Admin node：./stowaway_admin -l 9999 -s 123
  
  Meaning：
  
  -l     It means Stowaway is listening on port 9999 and waiting for incoming connection

  -s     It means Stowaway has used 123 as the encrypt key during the communication
  
  Be aware! -s option's value must be as same as the agents' 
 
startnode： ./stowaway_agent -m 127.0.0.1:9999 -l 10000 --startnode -s 123 --reconnect 5

  Meaning：
  
  -m    It means Stowaway's monitor node's address (In this case,it's the node we started above)
  
  -l    It means Stowaway is listening on port 10000 and waiting for incoming connection (optional)

  -s    It means Stowaway has used 123 as the encrypt key during the communication 

  --startnode  It means Stowaway is started as FIRST agent node(if the node is the first one , you MUST add this option!!! And there are two submode of agent mode,if you want to start the second, third one....., just remove this option)

  --reconnect It means the startnode will automatically try to reconnect to admin node at 5 second intervals(in this example).PS:
  if you want to start the reconnect function, just add this option when you start the STARTNODE , there is no need to add this option when you start the other simple nodes.

And now, if you want to start the following simple node passively(means waiting for the connection from upper node instead of connecting to upper node actively)

Then,the command of startnode should be changed to : ./stowaway_agent -m 127.0.0.1:9999 --startnode -s 123 --reconnect 5
  
The following simple node: ./stowaway_agent -l 10001 -s 123 -r

  -r It means you want to start the node in passive mode(For instance: you can add node 2 into the net via node 1 actively connect to node 2, instead of node 1 just waiting for the connection from node 2 )

  -l    It means Stowaway is listening on port 10000 and waiting for incoming connection (optional)

And now, you can use admin,type in 'use 1' ---> 'connect 127.0.0.1:10001' to add this simple node into network

But,if you want to start the following simple node actively(means connecting to upper node actively instead of waiting for the connection from upper node)

Then, the command of startnode will still : ./stowaway_agent -m 127.0.0.1:9999 -l 10000 --startnode -s 123 --reconnect 5  

The following simple node: ./stowaway_agent -m 127.0.0.1:10000 -l 10001 -s 123

And now ,you can add this simple node into network
```

Example 2： Admin node connecting to startnode actively
```
Admin node: ./stowaway_admin -s 123 -c 127.0.0.1:9999

  Meaning:

  -c  It means startnode's address
    
startnode: ./stowaway_agent -l 9999 -s 123 --startnode -r

  Meaning:

  -l,-s,--startnode is as the same as Example 1

  -r   It means startnode is started passively

The following simple node can be started as Example 1's description

The next time you want to reconnect to the startnode and rebuild the whole network,just start the admin node like : ./stowaway_admin -s 123 -c 127.0.0.1:9999,and then whole network will be rebuilt

```

```
Port Reuse:

  Now Stowaway provide the port reuse functions based on SO_REUSEPORT和SO_REUSEADDR features and iptable rules(startnode and simple node can both use this function)

  In Linux environment, it can reuse most ports

  In Windows environment,it cannot reuse service port like IIS,RDP,can reuse Mysql,Apache and so on

  Nginx is not supported under default setting

SO_REUSEPORT/SO_REUSEADDR mode's example:(startnode is reusing port 80)

  Mainly support windows operation system

  Admin: ./stowaway_admin -c 192.168.0.105:80 -s 123 --rhostreuse

    -c/-s same as i mentioned before

    --rhostreuse it means the node that admin want to connect is under port reusing mode(This option MUST be set if the node you want to connect is reusing port)

  Startnode: ./stowaway_agent -s 123 --startnode --report 80 --rehost 192.168.0.105

    -s/--startnode the same as i mentioned before 

    --report it means the port you want to reuse

    --rehost it means the ip address you want to listen on(DO NOT set 0.0.0.0,it will make the reuse funtion lose its effect)

Now if there is a simple node followed by startnode and want to connect to startnode,the command can be like this: ./stowaway_agent -s 123 -m 192.168.0.105:80 --rhostreuse

All options's meanings are the same as i mentioned before

Iptables mode's example:(startnode is reusing port 22)

  Mainly support Linux,needs root privilege

  And this kind of reusing method depend on using iptables rule to redirect traffics to the port(-l option) before they reach the reuse port(--report option).

  Startnode: ./stowaway_agent -s 123 --startnode --report 22 -l 10000

    --startnode/-s same as i mentioned before 

    --report it means the port you want to reuse

    -l it means the port that will accept all redirect traffics(node will listen on it)

when the startnode started,you can use the reuse.py in bolder "script"

Open the reusing function: python port_reuse.py --start --rhost 192.168.0.105 --rport 22

And now admin can connect to startnode: ./stowaway_admin -c 192.168.0.105:22 -s 123 --rhostreuse

  -c/-s/--rhostreuse same as i mentioned before

Attention! :If node is killed by ctrl-c or command "kill",it will clean up the iptables rules automatically,but if it is killed by command "kill -9",then it can't do that and it will lead to the service originally run on the reusing port cannot be reached,so in order to avoid this situation ,the reuse.py provide the function that can stop the "port reusing" function.

If you want to stop "port reusing",just run reuse.py like this: python reuse.py --stop --rhost 192.168.0.105 --rport 22

And then the "port reusing" will be closed,and the service originally run on the reusing port can be reached again

Now if there is a simple node followed by startnode and want to connect to startnode,the command can be like this: ./stowaway_agent -s 123 -m 192.168.0.105:22 --rhostreuse

All options's meanings are the same as i mentioned before
```

**Some points you should know:**

**1.Every node(except for the admin node),can be connected by several nodes to build the network tree**

**2.When a node offline,it will force all the socks,reflect,forward services that related to this node down**

**3.If one of the branches is disconnected due to network fluctuations or disconnection of intermediate nodes (for example, admin is followed by startnode, and startnode's branch is divided into two branches, one is startnode-> node 2-> node 3-> node 4,another is startnode-> node 5-> node 6), then if node2 goes offline, node3 and node4 will not go offline, but will continue to survive. At this time, if the user wants to rejoin node3 and node4 to the network, the user has two options. One is that if startnode can directly access node3, then the user can reconnect the node3 with the connect or sshtunnel command at startnode at any time (remember, even if startnode can also access node4 at the same time, please DO NOT directly connect to node4, please connect the head node node3 of the entire missing chain (node3-> node4), so that you can rejoin node3 and node4 to the network; another option is when startnode cannot directly access node3 (that means you must pass through node2), please restart node2 and join it into the network first, and then use connect or sshtunnel command on node2 to connect to node3, so that node3 and node4 will rejoin to the network (As mentioned before, even if node2 can directly access node4, please do not connect to node4, just connect to node3)**

## Example

For instance：

Admin：

![admin](https://github.com/ph4ntonn/Stowaway/blob/master/img/admin.png)

Startnode：

![startnode](https://github.com/ph4ntonn/Stowaway/blob/master/img/startnode.png)

First simple Node(setting as reverse mode）：

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/node1.png)

Now, use admin node and type in "use 1" -> "connect 127.0.0.1:10001" ,then you can add node 1 into the net

Second simple Node：

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/node2.png)

When all agent nodes connected，check the topology in admin：

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/tree.png)

When all agent nodes connected，check all nodes's detail：

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/detail.png)

Set note for this node：

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/addnote.png)

Delete note of this node：

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/delnote.png)

Now we manipulate the second simple node through admin：

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/manipulate.png)

Open the remote interactive shell：

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/shell.png)

Now you can use interactive shell (the second simple node's) through admin

Start socks5 proxy service：

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/socks5.png)


Now you can use the admin's port 7777 as the socks5 proxy service

And it can proxy your traffic to the second simple node and the second simple node will do its work as socks server（ When you want to shut down this socks5 service, just type in "stopsocks" under this mode to turn off it)

If you want to set username/password for socks5 service(Firefox support this function, Chrome doesn't), For instance, if you want to set the username as ph4ntom and password as 11235,just change the command to : socks 7777 ph4ntom 11235 (Be aware,do not use colon(:) in either username or password)

Shutdown the socks5 proxy service：

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/shutdownsocks5.png)

Open ssh：

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/ssh.png)

And it can make the second simple node do its work as ssh cilent to start a ssh connection to 127.0.0.1:22(in this example)

PS: In this function,you can type in ```pwd``` to check where you currently are

Now if you want to add another node into the network, you can use sshtunnel command：

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/waiting.png)
![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/sshtunnel.png)


And I suggest to use the "sshtunnel" command to add the node into network when the firewall has stricted all the traffics expect for SSH(In general,you can just use "connect" command,it also works)

Upload/Download file:

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/upload.png)

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/download.png)

If you want to upload/download any files,use upload/download + (filepath) under the node mode(after using command "use xxx"),and then you can upload specific file to selected agent/download specific file from selected agent XD (Be aware! You can just transfer only ONE file at the same time,if you want to transfer more,please wait for the previous one complete.)

Port mapping：

Mapping local port to remote port:
![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/portforward.png)
![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/connectforward.png)

Now you can connect to 127.0.0.1:8888 like really connecting to 127.0.0.1:22(forward local port to remote port)

If you want to shutdown the forward service:

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/shutdownforward.png)

Mapping remote port to local port:

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/portreflect.png)
![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/connectreflect.png)

Now anyone who connect to 127.0.0.1:80 will connect to 127.0.0.1:22 in fact(forward remote port to local port)

If you want to shutdown the reflect service:

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/shutdownreflect.png)

If you want to open a new listener on node 3 for accepting the following nodes connection,you can use "listen" command(PS:If you use the -l option when you start the node 3,then the listener created by -l option will not be killed because of "listen" command,you can also use it too)

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/listen.png)
```
For more detail, just type help to get further informations
```
## TODO

- [x] Network traffic encryption
- [x] Method to turn off socks5 proxy
- [x] Reconnection
- [x] Port mapping
- [ ] Clean codes, optimize logic
- [x] Add reverse connect mode
- [x] Support port reuse

### Attention

- This porject is coding just for fun , the logic structure and code structure are not strict enough, please don't be so serious about it
- This program will be slightly bigger than usual after compiled, but actually through my test , it just 1 MB more than usual,Maybe slightly big on IOT platform(1MB maybe not a big deal lol),so if you got any problem when you are using it on IOT platform,just tell me, and i will try my best to decrease the size.
- Admin node MUST be online when new node is being added into the network
- If you want to compile this project from source code,you can run build_admin.sh/build_agent.sh（Be Mentioned!!!!!!!!!! The default compile result is AGENT mode and please run build_agent.sh. But if you want to compile ADMIN mode,please see the main.go file and FOLLOW THE INSTRUCTION, and next you can run build_admin.sh to get admin mode program.)

### Thanks

- [rootkiter#Termite](https://github.com/rootkiter/Termite)
- [Venom](https://github.com/Dliv3/Venom)