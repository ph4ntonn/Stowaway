# Stowaway

Stowaway是一个利用go语言编写的简单的ssh转发工具

此程序旨在使用户可以在不安全的网络中安全地访问位于安全网络中的另一主机所提供的受限制的ssh服务

认证过程采用ecc加密

PS:也可用作SOCKS5服务端程序

# Usage

当你将Stowaway作为ssh转发工具时，其分为服务端及客户端两部分，集合在一个程序中，由不同的参数控制

不想编译的盆油可以直接用release下编译完成的程序

简单示例：
```
  服务端：./stowaway listen -s 1234 -p 9291  --heartbeat --replay --duration 5
  
  命令解析：
  
  listen代表以服务端模式启动
  
  -s 参数代表本次通讯所使用的认证密钥
  
  -p 参数代表服务端监听的服务端口

  --heartbeat 代表打开心跳包功能（可选,需要客户端同样开启此功能）

  --replay 代表开启反重放机制(可选,需要与--duration选项一起使用)

  --duration 代表反重放机制的超时时间(即将超时多久的认证包视为无效包,在例子中，表示超时5s以上的认证包无效,可选,需要与--replay选项一起使用)
```
```
  客户端： ./stowaway connect -s 1234 -t "127.0.0.1:9291|9999|22" --heartbeat
  
  命令解析：
  
  connect代表以客户端模式启动
  
  -s 如上
  
  -t 代表本次通讯所使用的隧道参数，格式为：
  
  远程主机ip:服务端口|本地欲转发的端口|远程服务器接受转发的端口
  
  简单来说，上面的命令可解释为：
  
  利用secret 1234 与位于127.0.0.1:9291上的服务端程序进行认证，请求将本地的9999端口转发至服务端程序所在主机的22号端口

 --heartbeat 代表打开心跳包功能（可选，需要服务端同样开启此功能） 
```

当你将Stowaway作为socks5服务程序使用时,示例如下：
```
SOCKS5:  ./stowaway socks5 -u 123 -s 321 -p 43690

命令解析：

socks5代表以socks5服务端模式启动

-u 代表认证所用的用户名(可选)

-s 代表认证所用的密码(可选)

（当-u和-s选项都不指定时，即代表无需认证）

-p 代表sock5服务器监听的端口
```
#  Example

一个简单的例子：

服务端：

![server](https://github.com/ph4ntonn/Stowaway/blob/master/img/server.png)

客户端：

![client](https://github.com/ph4ntonn/Stowaway/blob/master/img/client.png)

连接：

![connect](https://github.com/ph4ntonn/Stowaway/blob/master/img/connect.png)

```
此时连接127.0.0.1的9999端口即相当于访问了22号端口
```
# TODO
```
优化转发http流的功能

优化代码

and more more ........
```
```
(这个程序只是写着玩的hh，小声bb
```
