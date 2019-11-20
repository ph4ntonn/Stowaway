# Stowaway

Stowaway是一个利用go语言编写的简单的ssh转发工具

此程序旨在使用户可以在不安全的网络中安全地访问位于安全网络中的另一主机所提供的受限制的ssh服务

认证过程采用ecc加密

PS:转发HTTP流量亦可，但尚未优化，先留一个坑:blush:

# Usage

Stowaway分为服务端及客户端两部分，集合在一个程序中，由不同的参数控制

不想编译的盆油可以直接用release下编译完成的程序

简单示例：
```
  服务端：./stowaway listen -s 1234 -p 9291
  
  命令解析：
  
  listen代表以服务端模式启动
  
  -s 参数代表本次通讯所使用的认证密钥
  
  -p 参数代表服务端监听的服务端口
```
```
  客户端： ./stowaway connect -s 1234 -t "127.0.0.1:9291|9999|22"
  
  命令解析：
  
  connect代表以客户端模式启动
  
  -s 如上
  
  -t 代表本次通讯所使用的隧道参数，格式为：
  
  远程主机ip:服务端口|本地欲转发的端口|远程服务器接受转发的端口
  
  简单来说，上面的命令可解释为：
  
  利用secret 1234 与位于127.0.0.1:9291上的服务端程序进行认证，请求将本地的9999端口转发至服务端程序所在主机的22号端口
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