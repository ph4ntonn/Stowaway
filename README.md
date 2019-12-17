# Stowaway

[English](README_EN.md)

Stowaway是一个利用go语言编写的多级代理工具

用户可使用此程序将外部流量通过多个节点代理至内网，并实现管理功能

PS:demo文件夹下为其雏形demo，亦可使用，详见demo文件下的readme文件

PPS:大佬们轻喷，后续会继续优化代码！

> 此工具仅限于安全研究和教学，用户承担因使用此工具而导致的所有法律和相关责任！ 作者不承担任何法律和相关责任！

## 特性

- 一目了然的节点管理
- 多级socks5流量代理转发
- ssh流量代理
- 远程交互式shell
- 节点间流量以AES-256(CBC模式)进行加密

## Usage

Stowaway分为admin端和agent端两种形式，集成在一个程序中以不同参数控制


不想编译的盆油可以直接用release下编译完成的程序

简单示例：
```
  Admin端：./stowaway admin -l 9999 -s 123
  
  命令解析：
  
  admin代表以admin模式启动
  
  -l 参数代表监听端口

  -s 参数代表节点通信加密密钥(admin端与agent端必须一致!)
  
  暂时就这两个参数！！！！！！
 
```
```
  agent端： ./stowaway agent -m 127.0.0.1:9999 -l 10000 --startnode -s 123
  
  命令解析：
  
  agent代表以agent端模式启动
  
  -m 代表上一级节点的地址
  
  -l 代表监听端口

  -s 参数代表节点通信加密密钥(admin端与agent端必须一致!)

  --startnode 代表此节点是agent端的第一个节点（若无--startnode表示为普通节点，命令与startnode一致）

  暂时就这四个参数！！！！！！
  
```

## Example

一个简单的例子(以一个admin端三个agent端为例）：

Admin端：

![admin](https://github.com/ph4ntonn/Stowaway/blob/master/img/admin.png)

Startnode端：

![startnode](https://github.com/ph4ntonn/Stowaway/blob/master/img/startnode.png)

第一个普通Node：

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/node1.png)


第二个普通Node：

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/node2.png)

连入完成后，admin端查看节点：

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/chain.png)

此时在admin端操控第二个普通node节点：

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/manipulate.png)

打开远程shell：

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/shell.png)

此时就可以在admin端操纵第二个普通节点的shell

打开socks5代理：

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/socks5.png)

此时用户即可以将admin端的7777端口作为socks5代理端口，将流量代理至第二个普通node节点(当想关闭socks服务时，在节点模式下输入stopsocks即可关闭与此节点相关的socks代理服务)

打开ssh：

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/ssh.png)

此时就可以在admin端将ssh流量代理至第二个普通节点，由第二个普通节点来访问指定的ssh服务

PS: 在ssh模式下，你可以用pwd来判断自己所处的文件夹（好吧，其实就是没法把banner传回来。。）

```
具体命令使用可输入help查询
```
## TODO

- [x] 节点间通信流量加密
- [x] 关闭代理与端口转发
- [ ] 重连功能
- [ ] 清理代码，优化逻辑
- [ ] 增加cc功能

### 注意事项

- 此程序仅是闲暇时开发学习，结构及代码结构不够严谨，功能可能存在bug，请多多谅解
- 当admin端掉线，所有后续连接的agent端都会退出
- 当多个agent端中有一个掉线，后续的agent端都会掉线
- 在admin启动后，必须有节点连入才可操作
- 需要自己编译的童鞋可在根目录利用：```go build -ldflags="-w -s"```自行编译以适配多种平台
- 暂时不支持windows

### 致谢

- [rootkiter#Termite](https://github.com/rootkiter/Termite)
- [Venom](https://github.com/Dliv3/Venom)