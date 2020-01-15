# Stowaway

Stowaway is Multi-hop proxy tool for security researchers and pentesters

Users can easily proxy their network traffic to intranet nodes (multi-layer)

PS: The files under demo folder are Stowaway's beta version,it's still functional, you can check the detail by README.md file under the demo folder

PPS: Thanks for everyone's star, i'm just an amateur, and the code still need be optimized,so if you find anything wrong or bugs, feel free to tell me, and i'll fix it :kissing_heart:. 
> This tool is limited to security research and teaching, and the user bears all legal and related responsibilities caused by the use of this tool! The author does not assume any legal and related responsibilities!

## Features

- obvious node topology
- can be used on multiple platforms
- multi-hop socks5 traffic proxy
- multi-hop ssh traffic proxy
- remote interactive shell
- upload/download functions
- network traffic encryption with AES-256(CBC mode)


## Usage

Stowaway can be excuted as two kinds of mode: admin && agent


If you don't want to compile the project by yourself, you can check the release tag to get ONE!

Simple example：
```
  Admin mode：./stowaway admin -l 9999 -s 123
  
  Meaning：
  
  admin  It means Stowaway is started as admin mode
  
  -l     It means Stowaway is listening on port 9999 and waiting for incoming connection

  -s     It means Stowaway has used 123 as the encrypt key during the communication
  
  Be aware! -s option's value must be as same as the agents' 

  For now, there are only three options above are supported!
 
```
```
  agent mode： ./stowaway agent -m 127.0.0.1:9999 -l 10000 --startnode -s 123 -r
  
  Meaning：
  
  agent It means Stowaway is started as agent mode 
  
  -m    It means Stowaway's monitor node's address (In this case,it's the node we started above)
  
  -l    It means Stowaway is listening on port 10000 and waiting for incoming connection 

  -s    It means Stowaway has used 123 as the encrypt key during the communication 

  --startnode  It means Stowaway is started as FIRST agent node(if the node is the first one , you MUST add this option!!! And there are two submode of agent mode,if you want to start the second, third one....., just remove this option)

  -r It means you want to start the node in reverse mode(For instance: you can add node 2 into the net via node 1 actively connect to node 2, instead of node 1 just waiting for the connection from node 2 )

  Be aware! -s option's value must be as same as the agents' 

 For now, there are only five options above are supported!
  
```

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

Open ssh：

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/ssh.png)

And it can proxy your ssh traffic to the second simple node and the second simple node will do its work as ssh cilent

PS: In this function,you can type in ```pwd``` to check where you currently are

Upload/Download file:

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/upload.png)

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/download.png)

If you want to upload/download any files,use upload/download + (filepath) under the node mode(after use command "use xxx"),and then you can upload specific file to selected agent/download specific file from selected agent XD

```
For more detail, just type help to get further informations
```
## TODO

- [x] Network traffic encryption
- [x] Method to turn off socks5 proxy
- [ ] Reconnection
- [ ] Clean codes, optimize logic
- [ ] Add cc function
- [x] Add reverse connect mode

### Attention

- This porject is coding just for fun , the logic structure and code structure are not strict enough, please don't be so serious about it
- When the admin offline, all agent nodes will offline too(maybe it will be changed in future)
- When one of the agents offline, the agent nodes after it will offline
- Once the admin started, you need to connect at least one agent node to it before you do any operations
- If you want to compile this project for supporting more platform, you can use ```go build -ldflags="-w -s"``` to do that

### Thanks

- [rootkiter#Termite](https://github.com/rootkiter/Termite)
- [Venom](https://github.com/Dliv3/Venom)