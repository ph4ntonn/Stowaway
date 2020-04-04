# Stowaway

[English](README_EN.md)

Stowaway是一个利用go语言编写的多级代理工具

用户可使用此程序将外部流量通过多个节点代理至内网，并实现管理功能

PS:谢谢大家的star，这个程序还不成熟规范，写来仅为学习，我也还是个菜鸟，后续会继续优化，同时欢迎大家使用后提出问题 :kissing_heart:。

> 此工具仅限于安全研究和教学，用户承担因使用此工具而导致的所有法律和相关责任！ 作者不承担任何法律和相关责任！

## 特性

- 一目了然的节点管理
- 正向/反向连接
- 多平台适配
- 多级socks5流量代理转发
- ssh代理连接
- 远程交互式shell
- 上传及下载文件
- 端口本地/远程映射
- 节点间流量以AES-256(CBC模式)进行加密

## Usage

Stowaway分为admin端和agent端两种形式


不想编译的盆油可以直接用[release](https://github.com/ph4ntonn/Stowaway/releases)下编译完成的程序(同时提供未经压缩版及upx压缩版，可各取所需)

第一种情况： Admin端监听，等待startnode连接

```
Admin端：./stowaway admin -l 9999 -s 123
  
  命令解析：
  
  admin代表以admin模式启动
  
  -l 参数代表监听端口

  -s 参数代表节点通信加密密钥(admin端与agent端必须一致!)
 
startnode端： ./stowaway agent -m 127.0.0.1:9999 -l 10000 --startnode -s 123 --reconnect 5
  
  命令解析：
  
  agent代表以agent端模式启动
  
  -m 代表上一级节点的地址
  
  -l 代表监听端口

  -s 参数代表节点通信加密密钥(admin端与agent端必须一致!)

  --startnode 代表此节点是agent端的第一个节点（第一个节点必须加上--startnode选项！若无--startnode表示为普通节点，命令与startnode一致）

  --reconnect 代表startnode将在admin下线后主动尝试不断重连（此例子中为每5秒重连一次）注意：若需要重连功能，只需要在startnode使用此参数即可，其后节点无需此参数，正常启动即可



此时若后续的节点希望以passive模式启动（即本节点等待上一级节点的主动连接，而不是主动连接上一节点）

那么，上述startnode命令可改为 ./stowaway agent -m 127.0.0.1:9999 --startnode -s 123 --reconnect 5

后续节点启动命令为：./stowaway agent -l 10001 -s 123 -r

  -r 代表以passive模式启动（即本节点等待上一级节点的主动连接，而不是主动连接上一节点，若正向连接可以去除此选项）

此时在admin端进入startnode(use 1)，使用connect命令（connect 127.0.0.1:10001)即可将后续节点加入网络

若后续节点希望以active模式启动（即本节点主动连接上一级节点）

那么，startnode启动命令仍为：./stowaway agent -m 127.0.0.1:9999 -l 10000 --startnode -s 123 --reconnect 5 

后续节点启动命令为：./stowaway agent -m 127.0.0.1:10000 -l 10001 -s 123

此时即可将后续节点加入网络
 
```

第二种情况：Admin端主动连接startnode端

```
Admin端: ./stowaway admin -s 123 -c 127.0.0.1:9999
  
  命令解析：
  
  admin代表以admin模式启动
  
  -s 同上

  -c 代表startnode所在的地址

startnode端: ./stowaway agent -l 9999 -s 123 --startnode -r

  命令解析：

  agent代表以agent模式启动

  -l，-s ，--startnode同上

  -r/--reverse：代表此节点以passive模式启动

后续普通节点同第一种情况中的普通节点启动方法一致
```

**几个注意点：**

**1.除了admin节点以外，普通的agent以及startnode节点可以被多个agent端连接，以组成树状网络**

**2.当有节点掉线时（假设a节点后节点b掉线了），那么此时所有的socks，reflect，forward服务都会被强制停止（不管这个服务是否属于b节点）,需要重新手动启用**

## Example

一个简单的例子(以一个admin端三个agent端为例）：

Admin端：

![admin](https://github.com/ph4ntonn/Stowaway/blob/master/img/admin.png)

Startnode端：

![startnode](https://github.com/ph4ntonn/Stowaway/blob/master/img/startnode.png)

第一个普通Node(设置为反向连接模式)：

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/node1.png)

此时，进入(use命令，本例中为use 1）此节点的上级节点（即startnode），利用命令connect 127.0.0.1:10001 即可将此反向模式节点加入网络

第二个普通Node：

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/node2.png)

连入完成后，admin端查看节点拓扑：

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/tree.png)

连入完成后，admin端查看节点详细信息：

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/detail.png)

为此节点设置备忘：

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/addnote.png)

为此节点删除备忘：

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/delnote.png)

此时在admin端操控第二个普通node节点：

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/manipulate.png)

打开远程shell：

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/shell.png)

此时就可以在admin端操纵第二个普通节点的shell

打开socks5代理：

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/socks5.png)

此时用户即可以将admin端的7777端口作为socks5代理端口，将流量代理至第二个普通node节点(当想关闭socks服务时，在节点模式下输入stopsocks即可关闭与此节点相关的socks代理服务).
如果需要设置socks5用户名密码（Firefox支持，Chrome不支持），例如需要设置用户名为ph4ntom，密码为11235，则可将输入命令改为:socks 7777 ph4ntom 11235 | 注意：切勿在用户名以及密码中使用冒号（:)

关闭socks5代理：

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/shutdownsocks5.png)

打开ssh：

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/ssh.png)

此时第二个普通节点会作为ssh客户端,(此节点)会发起ssh连接来访问指定的ssh服务，并将ssh数据回传至admin端

PS: 在ssh模式下，你可以用pwd来判断自己所处的文件夹（好吧，其实就是没法把banner传回来。。）

上传/下载文件：

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/upload.png)

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/download.png)

上传下载功能命令为 upload/download 后加文件路径（注意要在节点模式下使用）,此时就可以上传文件至指定节点/下载指定节点的文件(注意，同时只能传输一个文件，请务必等之前一个传输完成再进行下一步操作)

端口本地/远程映射：

本地端口映射至远程端口

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/portforward.png)
![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/connectforward.png)

此时连接127.0.0.1的8888端口，就相当于连接至127.0.0.1的22端口（本地映射至远程）

如果不想forward了，可以关闭

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/shutdownforward.png)

远程端口映射至本地

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/portreflect.png)
![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/connectreflect.png)

此时外部访问127.0.0.1的80端口，就相当于访问127.0.0.1的22端口（远程映射至本地）

如果不想reflect了，可以关闭

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/shutdownreflect.png)

```
具体命令使用可输入help查询
```
## TODO

- [x] 节点间通信流量加密
- [x] 关闭代理与端口转发
- [x] 重连功能
- [ ] 清理代码，优化逻辑
- [x] 节点反向连接
- [x] 端口映射
- [ ] 支持端口复用(试验后觉得比较鸡肋，暂时先不加进去)

### 注意事项

- 此程序仅是闲暇时开发学习，结构及代码结构不够严谨，功能可能存在bug，请多多谅解
- 当admin端掉线，所有后续连接的agent端都会退出(当startnode未开启重连模式时)
- 当多个agent端中有一个掉线，后续的agent端都会掉线
- 在admin启动后，必须有节点连入才可操作
- 如需从源代码编译本项目，请运行build_admin.sh/build_agent.sh文件来编译对应类型的Stowaway(注意！！！！！！默认编译的是agent模式，此时请运行build_agent.sh,如需编译admin，请查看main.go文件中的提示，按照提示进行操作后，运行build_admin.sh文件)

### 致谢

- [rootkiter#Termite](https://github.com/rootkiter/Termite)
- [Venom](https://github.com/Dliv3/Venom)