![stowaway.png](https://github.com/ph4ntonn/Stowaway/blob/master/img/logo.png)

# Stowaway

[![GitHub issues](https://img.shields.io/github/issues/ph4ntonn/Stowaway)](https://github.com/ph4ntonn/Stowaway/issues)
[![GitHub forks](https://img.shields.io/github/forks/ph4ntonn/Stowaway)](https://github.com/ph4ntonn/Stowaway/network)
[![GitHub stars](https://img.shields.io/github/stars/ph4ntonn/Stowaway)](https://github.com/ph4ntonn/Stowaway/stargazers)
[![GitHub license](https://img.shields.io/github/license/ph4ntonn/Stowaway)](https://github.com/ph4ntonn/Stowaway/blob/master/LICENSE)

Stowaway is Multi-hop proxy tool for security researchers and pentesters

Users can easily proxy their network traffic to intranet nodes (multi-layer),break the restrction and manipulate all the nodes that under your control XD

PS: Thanks for everyone's star, i'm just an amateur, and the code still need be optimized,so if you find anything wrong or bugs, feel free to tell me, prs and issues are welcome :kissing_heart:. 

PPS: Please read the usage method and the precautions at the end of the article before use!

> This tool is limited to security research and teaching, and the user bears all legal and related responsibilities caused by the use of this tool! The author does not assume any legal and related responsibilities!

## Features

- Obvious node topology
- Clear information display of nodes and keep them permanently
- Active/passive connect mode between nodes
- Reverse connection between nodes through socks5 proxy
- Ssh tunnel mode
- Can be used on multiple platforms
- Multi-hop socks5 traffic proxy(Support UDP/TCP,IPV4/IPV6)
- Multi-hop ssh traffic proxy
- Remote interactive shell
- Upload/download functions
- Port Mapping(local to remote/remote to local)
- Port Reuse
- Open or Close all the services arbitrarily
- Authenicate each other between nodes
- Network traffic encryption with AES-256(CBC mode)

## Download and Demo

- Check the [release](https://github.com/ph4ntonn/Stowaway/releases) to get one.And if you want the Uncompressed collection，check [Uncompressed](https://github.com/ph4ntonn/Stowaway/releases/download/v1.6.2/Uncompress_By_Upx.7z) or you can choose the Compressed collection(much more easier for u to upload agent to target server),check [Compressed](https://github.com/ph4ntonn/Stowaway/releases/download/v1.6.2/Compressed_By_Upx.tar)

- Demo video: [Youtube](https://www.youtube.com/watch?v=O3DHQ1ESMhw)

## Usage

### Character
Stowaway has three kinds of characters: 
- ```admin```  The node that the pentester can use as a console
- ```startnode```  startnode means the first node that connecting to admin,it's different from simple nodes
- ```simple node```  node that can be manipulated by admin

```startnode``` and ```simple node``` can be named as AGENT mode, they are almost identical except for some tiny differences(You can see the differences in the following content)

### Command

- Example 1：Admin node waiting the connection from startnode
  - **Admin node：./stowaway_admin -l 9999 -s 123**
  ```
    Meaning：
  
       -l     It means Stowaway is listening on port 9999 and waiting for incoming connection(default listening on 0.0.0.0)

       -s     It means Stowaway has used 123 as the encrypt key during the communication
  
       Be aware! -s option's value must be identical when you start every node
  ```

  - **startnode： ./stowaway_agent -c 127.0.0.1:9999 -l 10000 --startnode -s 123 --reconnect 5**
  ```
    Meaning：
  
       -c    It means Stowaway's monitor node's listening address
  
       -l    It means Stowaway is listening on port 10000 and waiting for incoming connection (You can also specify the listening ip at the same time, like 192.168.0.1:9999, if you don’t specify the ip, the default listening is 0.0.0.0; And besides，you can also open the listening port via "listen" command)

       -s    It means Stowaway has used 123 as the AES encrypt key during the communication 

       --startnode  It means Stowaway is started as startnode(if the node is the first node that connect to admin, you MUST add this option!!! otherwise, just remove this option)

       --reconnect It means the startnode will automatically try to reconnect to admin node every 5 seconds (in this example).

        PS:if you want to start the reconnect function,add --reconnect JUST when you start the STARTNODE , there is no need to add this option when you start the other simple nodes.
  ```

  - **Then, if you want to start the following simple node passively**

    The following simple node: ```./stowaway_agent -l 10001 -s 123 -r```

  ```
    Meaning：

       -r   It means you want to start the node in passive mode

       -l   It means Stowaway is listening on port 10001 and waiting for incoming connection
  ```
    And now, you can use admin,type in ```use 1```---> ```connect 127.0.0.1:10001``` to add this simple node into network

  - **But,if you want to start the following simple node actively**

    The following simple node：```./stowaway_agent -c 127.0.0.1:10000 -s 123```

    And now ,you can add this simple node into network

- Example 2： Admin node connecting to startnode actively

  - **Admin node: ./stowaway_admin -s 123 -c 127.0.0.1:9999**
  ```
    Meaning:

       -c  It means startnode's address
  ```  
  - **startnode: ./stowaway_agent -l 9999 -s 123 --startnode -r**
  ```
    Meaning:

       -l,-s,--startnode As the same as Example 1

       -r   It means startnode is started passively
  ```
  The following simple nodes can be started as Example 1's description

  The next time you want to reconnect to the startnode and rebuild the whole network,just start the admin node like ```./stowaway_admin -s 123 -c 127.0.0.1:9999```,and then whole network will be rebuilt

## SOCKS5 proxy connection

  Stowaway can perform reverse connections between nodes through socks5 proxy

  That needs following three params ```--proxy```,```--proxyu```, ```--proxyp```

### Example 

  Suppose there is a socks5 server A, ip is 6.6.6.6, proxy port is 1080, username is ph4ntom, password is just4fun 

  startnode needs to be connected to admin via server A, admin is deployed on 7.7.7.7, and the internal network address of startnode is 192.168.0.200 

  admin: ```./stowaway_admin -l 9999 -s 123```

  startnode: ```./stowaway_agent -c 7.7.7.7:9999 --startnode -s 123 -l 10000 --proxy 6.6.6.6:1080 --proxyu ph4ntom --proxyp just4fun```

  At this time, if there is an another socks5 server B in the intranet, the ip is 192.168.0.2, the proxy port is 1080, and there is no username and password.

  And there is a new child node that wants to connect to the startnode node via B,Then run command below

  node: ```./stowaway_agent -c 192.168.0.200:10000 -s 123 --proxy 192.168.0.2:1080```

  That's all :)

## Port Reuse

  Now Stowaway provide the port reuse functions based on SO_REUSEPORT和SO_REUSEADDR features and iptable rules(startnode and simple node can both use this function)

- In Linux environment, it can reuse most ports

- In Windows environment,it cannot reuse service port like IIS,RDP,can reuse Mysql,Apache and so on

- Nginx is not supported under default setting

### Method
- SO_REUSEPORT/SO_REUSEADDR mode's example:(startnode is reusing port 80)

  **Mainly support windows、mac operation system,also can be used on linux,but there are some restrictions on linux platform**

  - **Admin: ./stowaway_admin -c 192.168.0.105:80 -s 123 --rhostreuse**
  ```
    Meaning:

       -c/-s As the same as i mentioned before

       --rhostreuse It means the node that you want to connect is under port reusing mode(This option MUST be set if the node you want to connect is reusing port)
  ```
  - **Startnode: ./stowaway_agent -s 123 --startnode --report 80 --rehost 192.168.0.105**
  ```
    Meaning:

       -s/--startnode As the same as i mentioned before 

       --report It means the port you want to reuse

       --rehost It means the ip address you want to listen on(DO NOT set 0.0.0.0,it will make the reuse funtion lose its effect)
  ```
  Now if there is a simple node followed by startnode and want to connect to startnode,the command can be like this: ```./stowaway_agent -s 123 -c 192.168.0.105:80 --rhostreuse```

  All options's meanings are the same as i mentioned before

- Iptables mode's example:(startnode is reusing port 22)

  **Only support Linux,needs root privilege**

  And this kind of reusing method depend on using iptables rule to redirect traffics to the port(-l option) before they reach the reuse port(--report option).

  - **Startnode: ./stowaway_agent -s 123 --startnode --report 22 -l 10000**
  ```
    Meaning:

       --startnode/-s As the same as i mentioned before 

       --report It means the port you want to reuse

       -l It means the port that will accept all redirect traffics(node will listen on it)
  ```
  when the startnode started,you can use the reuse.py in folder "script"

  Set the value of SECRET(the value of SECRET is the value of -s option when you start the nodes)

  And open the reusing function: ```python reuse.py --start --rhost 192.168.0.105 --rport 22```

  - **And after using the script,admin can connect to startnode: ./stowaway_admin -c 192.168.0.105:22 -s 123 --rhostreuse** 
  ```
    Meaning:

       -c/-s/--rhostreuse same as i mentioned before
  ```

  Now if there is a simple node followed by startnode and want to connect to startnode,the command can be like this:
```./stowaway_agent -s 123 -c 192.168.0.105:22 --rhostreuse```

### Attention
- If node is killed by ctrl-c or command ```kill```,it will clean up the iptables rules automatically,but if it is killed by command ```kill -9```,then it can't do that and it will lead to the service originally run on the reusing port cannot be reached,so in order to avoid this situation ,the reuse.py provide the function that can stop the "port reusing" function.

  If you want to stop "port reusing",just run reuse.py like this: ```python reuse.py --stop --rhost 192.168.0.105 --rport 22```

  And then the "port reusing" will be closed,and the service originally run on the reusing port can be reached again

- If you use the port reusing mode via IPTABLES , the agent will be forced to monitor at 0.0.0.0, and you cannot specify ip+port by the ```-l``` option

## Some points you should know

1. **Every node(except for the admin node),can be connected by several nodes to build the network tree**

2. **When a node offline,it will force all the socks,reflect,forward services that related to this node down**

3. **If one of the branches is disconnected due to network fluctuations or disconnection of intermediate nodes (for example, admin is followed by startnode, and startnode's branch is divided into two branches, one is startnode-> node 2-> node 3-> node 4,another is startnode-> node 5-> node 6).**

**Then if node2 goes offline, node3 and node4 will keep survive.**
 
**At this time, if the user wants to rejoin node3 and node4 to the network, the user has two options. One is that if startnode can directly access node3, then the user can reconnect the node3 with the connect or sshtunnel command at startnode at any time (remember, even if startnode can also access node4 at the same time, please DO NOT directly connect to node4, please connect the head node node3 of the entire missing chain (node3-> node4), so that you can rejoin node3 and node4 to the network**

**Another option is when startnode cannot directly access node3 (that means you must pass through node2), please restart node2 and join it into the network first, and then use connect or sshtunnel command on node2 to connect to node3, so that node3 and node4 will rejoin to the network (As mentioned before, even if node2 can directly access node4, please do not connect to node4, just connect to node3)**

## Example

For instance：

- Admin：

![admin](https://github.com/ph4ntonn/Stowaway/blob/master/img/admin.png)

- Startnode：

![startnode](https://github.com/ph4ntonn/Stowaway/blob/master/img/startnode.png)

- First simple Node(setting as reverse mode）：

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/node1.png)

Now, use admin node and type in ```use 1``` -> ```connect 127.0.0.1:10001``` ,then you can add node 1 into the net

- Let startnode to listen on the port and accept connections from subsequent nodes 

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/listen.png)

```listen```command enables the current node to listen to the specified port and accept subsequent node connections on this port,and the format is ```listen <ip:>port```, if ip is not specified, the default listening is ```0.0.0.0```

- Second simple Node：

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/node2.png)

- Check the topology in admin：

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/tree.png)

- Check all nodes's detail：

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/detail.png)

- Set note for this node：

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/addnote.png)

Once the node's note is set,unless the node down,the note can be recovered even when admin offline follow by reconnecting to the whole network.So you don't need to be worried about losing the notes.

- Delete note of this node：

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/delnote.png)

- Now we manipulate the second simple node through admin：

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/manipulate.png)

- Open the remote interactive shell：

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/shell.png)

Now you can use interactive shell (the second simple node's) through admin

- Start socks5 proxy service：

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/socks5.png)


Now you can use the admin's port 7777 as the socks5 proxy service

And it can proxy your traffic to the second simple node and the second simple node will do its work like a socks5 server

If you want to set username/password for socks5 service:For instance, if you want to set the username as ph4ntom and password as 11235,just change the command to ```socks 7777 ph4ntom 11235``` (Be aware,do not use colon(:) in either username or password)

- Shutdown the socks5 proxy service：

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/shutdownsocks5.png)

- Open ssh：

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/ssh.png)

And it can make the second simple node do its work as ssh cilent to start a ssh connection to 127.0.0.1:22(arm and mipsel agent doesn't support this function)

PS: In this function,you can type in ```pwd``` to check where you currently are

- Now if you want to add another node into the network, you can choose ```sshtunnel``` command：

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/waiting.png)
![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/sshtunnel.png)


And I suggest to use the ```sshtunnel``` command to add the node into network **ONLY** when the firewall has stricted all the traffics expect for SSH.In general,you can just use ```connect``` command,it also works(arm and mipsel agent doesn't support this function)

- Upload/Download file:

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/upload.png)

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/download.png)

If you want to ```upload/download``` any files,use ```upload/download + (filepath)```,and then you can upload specific file to selected node/download specific file from selected node XD (Be aware! You can just transfer only ONE file at the same time,if you want to transfer more,please wait for the previous one complete.)

- Mapping local port to remote port:

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/portforward.png)
![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/connectforward.png)

Now you can connect to 127.0.0.1:8888 like really connecting to 127.0.0.1:22(forward local port to remote port)

- If you want to shutdown the forward service:

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/shutdownforward.png)

- Mapping remote port to local port:

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/portreflect.png)
![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/connectreflect.png)

Now anyone who connect to 127.0.0.1:80 will connect to 127.0.0.1:22 in fact(forward remote port to local port)

- If you want to shutdown the reflect service:

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/shutdownreflect.png)

> For more detail, just type ```help``` to get more informations

## TODO

- [x] Network traffic encryption
- [x] Enhance the robustness of the whole program
- [x] Method to turn on/off most functions given
- [x] Automatic reconnection
- [x] Port mapping
- [ ] Clean codes, optimize logic
- [x] Add reverse connect mode
- [x] Support port reuse
- [x] Let stowaway avoid the memory leak problem that happens on similar programs 

### Attention

- This porject is coding just for fun , the logic structure and code structure are not strict enough, please don't be so serious about it
- This program will be slightly bigger than usual after compiled, but actually through my test , it just 1 MB more than usual,Maybe slightly big on IOT platform(1MB maybe not a big deal lol),so if you got any problem when you are using it on IOT platform,just tell me, and i will try my best to decrease the size.
- The executable file after upx compress seems much more smaller than original one and it really makes upload stowaway to target server easily,but actually ,although it can make uploading stuff easier,it will occupy slightly more memory than original one,so pick the suitable version(upx or non-upx) depends on the target you want to use stowaway on.
- Admin node MUST be online when new node is being added into the network
- This program only supports standard UDP ASSOCIATE (that supports UDP proxy) described in [RFC1928](https://www.ietf.org/rfc/rfc1928.txt), please pay attention to what you are using when using socks5 udp proxy Programs (such as scanners, etc.), the packet construction method must comply with the standard [RFC1928] (https://www.ietf.org/rfc/rfc1928.txt), and the packet loss situation needs to be handled by yourself.
- If you want to compile this project from source code,you can run build_admin.sh/build_agent.sh（Be Mentioned!!!!!!!!!! The default compile result is AGENT mode and please run build_agent.sh. But if you want to compile ADMIN mode,please see the main.go file and FOLLOW THE INSTRUCTION, and next you can run build_admin.sh to get admin mode program.)

### Thanks

- [rootkiter#Termite](https://github.com/rootkiter/Termite)
- [Venom](https://github.com/Dliv3/Venom)