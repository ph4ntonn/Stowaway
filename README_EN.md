# Stowaway

Stowaway is Multi-hop proxy tool for security researchers and pentesters

Users can easily proxy their network traffic to intranet nodes (multi-layer)

PS: Thanks for everyone's star, i'm just an amateur, and the code still need be optimized,so if you find anything wrong or bugs, feel free to tell me, and i'll fix it :kissing_heart:. 
> This tool is limited to security research and teaching, and the user bears all legal and related responsibilities caused by the use of this tool! The author does not assume any legal and related responsibilities!

## Features

- Obvious node topology
- Active/passive mode
- Can be used on multiple platforms
- Multi-hop socks5 traffic proxy
- Multi-hop ssh traffic proxy
- Remote interactive shell
- Upload/download functions
- Port Mapping(local to remote/remote to local)
- Network traffic encryption with AES-256(CBC mode)


## Usage

Stowaway can be excuted as two kinds of mode: admin && agent(including startnode && simple node,startnode means the first node that connecting to admin,it's different from simple nodes)


If you don't want to compile the project by yourself, you can check the [release](https://github.com/ph4ntonn/Stowaway/releases) tag to get ONE!（Compressed and Uncompressed files are provided, choose the one you like)

Example 1：
```
Admin mode (passive,waiting the connection from startnode)：./stowaway admin -l 9999 -s 123
  
  Meaning：
  
  admin  It means Stowaway is started as admin mode
  
  -l     It means Stowaway is listening on port 9999 and waiting for incoming connection

  -s     It means Stowaway has used 123 as the encrypt key during the communication
  
  Be aware! -s option's value must be as same as the agents' 
 
startnode： ./stowaway agent -m 127.0.0.1:9999 -l 10000 --startnode -s 123 --reconnect 5

  Meaning：
  
  agent It means Stowaway is started as agent mode 
  
  -m    It means Stowaway's monitor node's address (In this case,it's the node we started above)
  
  -l    It means Stowaway is listening on port 10000 and waiting for incoming connection 

  -s    It means Stowaway has used 123 as the encrypt key during the communication 

  --startnode  It means Stowaway is started as FIRST agent node(if the node is the first one , you MUST add this option!!! And there are two submode of agent mode,if you want to start the second, third one....., just remove this option)

  --reconnect It means the startnode will automatically try to reconnect to admin node at 5 second intervals(in this example).PS:
  if you want to start the reconnect function, just add this option when you start the STARTNODE , there is no need to add this option when you start the other simple nodes.

And now, if you want to start the following simple node passively(means waiting for the connection from upper node instead of connecting to upper node actively)

Then,the command of startnode should be changed to : ./stowaway agent -m 127.0.0.1:9999 --startnode -s 123 --reconnect 5
  
The following simple node: ./stowaway agent -l 10001 -s 123 -r

  -r It means you want to start the node in passive mode(For instance: you can add node 2 into the net via node 1 actively connect to node 2, instead of node 1 just waiting for the connection from node 2 )

And now, you can use admin,type in 'use 1' ---> 'connect 127.0.0.1:10001' to add this simple node into network

But,if you want to start the following simple node actively(means connecting to upper node actively instead of waiting for the connection from upper node)

Then, the command of startnode should not be changed

The following simple node: ./stowaway agent -m 127.0.0.1:10000 -l 10001 -s 123

And now ,you can add this simple node into network
```

Example 2：
```
Admin node(connecting to startnode actively): ./stowaway admin -s 123 -c 127.0.0.1:9999

  Meaning:

  -c  It means startnode's address
    
startnode: ./stowaway agent -l 9999 -s 123 --startnode --reconnect 0 -r --single --activeconnect

  Meaning:

  --reconnect In this example,if you want to start startnode passively and also sustain the reconnect function of admin node, then this option must be set, and the value MUST be 0! Otherwise,this option must be deleted.

  -r   It means startnode is started passively

  --single  If this option is set,it means the whole network including just admin node and startnode(no following simple node),if there are still some simple nodes that need to be added into the network,DO NOT set this option

  ----activeconnect If this option is set,it means the second node(aka the FIRST SIMPLE node) will be started in passive mode,otherwise,this option must be deleted

The following simple node can be started as Example 1's description

And if you do not set the --single option, then you should start the startnode first and admin node successively (mention the sequence! startnode -> admin -> other nodes),then add the following simple nodes into the network.when all the things before are done,you can let the admin node offline(or just keep it online, it's totally up to you)

The next time you want to reconnect to the startnode,just start the admin node like : ./stowaway admin -s 123 -c 127.0.0.1:9999

Then, the whole network will be rebuilt.

But, if you set the --single option,it means you only need to start startnode and admin node, So when they are started,you can also choose to keep the admin node online or just let it offline.

The next time you want to reconnect to the startnode,just start the admin node like : ./stowaway admin -s 123 -c 127.0.0.1:9999 

Then, the whole network will be rebuilt,too.
```

**Some points you should know:**

**1.Every node(including startnode and simple nodes), cannot be actively connected by the following node if it was connected before,so if the following node is down and still want to reconnect, what you can do is starting the following node passively and waiting the previous node connect to it actively or just rebuild the whole network**

**2.When a node offline(for instance,A's following node B offline),it will force all the socks,reflect,forward services down,even the services are not associate to node B,so if you still want to use some of these services,you should restart them manually**

**3.When a node offline(for instance,A's following node B offline),then if you reconnect the node B,before you manipulate node B,you should enter the node A,and type in the command: recover ,this command will help you to recover the node A for the reconnection of node B. After you do that,you can manipulate node B instantly**


## Example

For instance(one admin;one startnode;two simple nodes）：

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

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/chain.png)

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

Open ssh：

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/ssh.png)

And it can make the second simple node do its work as ssh cilent to start a ssh connection to 127.0.0.1:22(in this example)

PS: In this function,you can type in ```pwd``` to check where you currently are

Upload/Download file:

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/upload.png)

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/download.png)

If you want to upload/download any files,use upload/download + (filepath) under the node mode(after using command "use xxx"),and then you can upload specific file to selected agent/download specific file from selected agent XD (Be aware! You can just transfer only ONE file at the same time,if you want to transfer more,please wait for the previous one complete.)

Port mapping：

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/portforward.png)
![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/connect.png)

Now you can connect to 127.0.0.1:8888 like really connecting to 127.0.0.1:22(forward local port to remote port)

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/portreflect.png)
![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/connectreflect.png)

Now anyone who connect to 127.0.0.1:80 will connect to 127.0.0.1:22 in fact(forward remote port to local port)

```
For more detail, just type help to get further informations
```
## TODO

- [x] Network traffic encryption
- [x] Method to turn off socks5 proxy
- [x] Reconnection
- [x] Port mapping
- [ ] Clean codes, optimize logic
- [ ] Add cc function
- [x] Add reverse connect mode
- [ ] Support port reuse(seems not very essential,so maybe add it later)

### Attention

- This porject is coding just for fun , the logic structure and code structure are not strict enough, please don't be so serious about it
- When the admin offline, all agent nodes will be offline too(only when startnode isn't under reconnect mode(passive or active))
- When one of the agents offline, the agent nodes after it will offline
- Once the admin started, you need to connect at least one agent node to it before you do any operations
- If you want to compile this project from source code,you can run build_admin.sh/build_agent.sh（Be Mentioned!!!!!!!!!! The default compile result is AGENT mode and please run build_agent.sh. But if you want to compile ADMIN mode,please see the main.go file and FOLLOW THE INSTRUCTION, and next you can run build_admin.sh to get admin mode program.)

### Thanks

- [rootkiter#Termite](https://github.com/rootkiter/Termite)
- [Venom](https://github.com/Dliv3/Venom)