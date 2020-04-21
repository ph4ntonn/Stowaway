# Stowaway

[English](README_EN.md)

Stowaway是一个利用go语言编写、专为渗透测试工作者制作的多级代理工具

用户可使用此程序将外部流量通过多个节点代理至内网，突破内网访问限制，并轻松实现管理功能

PS:谢谢大家的star，同时欢迎大家使用后提出问题 :kissing_heart:。

> 此工具仅限于安全研究和教学，用户承担因使用此工具而导致的所有法律和相关责任！ 作者不承担任何法律和相关责任！

## 特性

- 一目了然的节点树管理
- 节点信息展示
- 节点间正向/反向连接
- ssh隧道连接
- 多平台适配
- 多级socks5流量代理转发
- ssh代理连接
- 远程交互式shell
- 上传及下载文件
- 端口本地/远程映射
- 端口复用
- 节点间相互认证
- 节点间流量以AES-256(CBC模式)进行加密

## Usage

Stowaway一共分为三种角色，admin，startnode和普通node，其中startnode和普通node的区别在于startnode是第一个节点（即所有普通节点里的第一个节点，相当于“入口”节点）


不想编译的盆油可以直接用[release](https://github.com/ph4ntonn/Stowaway/releases)下编译完成的程序(同时提供未经压缩版及upx压缩版，可各取所需)

第一种情况： Admin端监听，等待startnode连接

```
Admin端：./stowaway_admin -l 9999 -s 123
  
  命令解析：
  
  -l 参数代表监听端口

  -s 参数代表节点通信加密密钥(admin端与agent端必须一致!)
 
startnode端： ./stowaway_agent -m 127.0.0.1:9999 -l 10000 --startnode -s 123 --reconnect 5
  
  命令解析：
  
  -m 代表上一级节点的地址
  
  -l 代表监听端口(如果暂时不需要监听，也可直接省略,后续如果需要开启，可参见listen命令的用法)

  -s 参数代表节点通信加密密钥(admin端与agent端必须一致!)

  --startnode 代表此节点是agent端的第一个节点（第一个节点必须加上--startnode选项！若无--startnode表示为普通节点，命令与startnode一致）

  --reconnect 代表startnode将在admin下线后主动尝试不断重连（此例子中为每5秒重连一次）注意：若需要重连功能，只需要在startnode使用此参数即可，其后节点无需此参数，正常启动即可,若不需要重连，省略此选项即可



此时若后续的节点希望以passive模式启动（即本节点等待上一级节点的主动连接，而不是主动连接上一节点）

那么，上述startnode命令可改为 ./stowaway_agent -m 127.0.0.1:9999 --startnode -s 123 --reconnect 5

后续节点启动命令为：./stowaway_agent -l 10001 -s 123 -r

  -r 代表以passive模式启动（即本节点等待上一级节点的主动连接，而不是主动连接上一节点，若正向连接可以去除此选项）

  -l 代表监听端口

此时在admin端进入startnode(use 1)，使用connect命令（connect 127.0.0.1:10001)即可将后续节点加入网络



若后续节点希望以active模式启动（即本节点主动连接上一级节点）

那么，startnode启动命令仍为：./stowaway_agent -m 127.0.0.1:9999 -l 10000 --startnode -s 123 --reconnect 5 

后续节点启动命令为：./stowaway_agent -m 127.0.0.1:10000 -s 123

此时即可将后续节点加入网络
 
```

第二种情况：Admin端主动连接startnode端

```
Admin端: ./stowaway_admin -s 123 -c 127.0.0.1:9999
  
  命令解析：
  
  -s 同上

  -c 代表startnode所在的地址

startnode端: ./stowaway_agent -l 9999 -s 123 --startnode -r

  命令解析：

  -l，-s ，--startnode同上

  -r/--reverse：代表此节点以passive模式启动

后续普通节点同第一种情况中的普通节点启动方法一致

下一次想要重连时，再次执行./stowaway_admin -s 123 -c 127.0.0.1:9999，即可重建网络

```

```
端口复用机制：

  当前Stowaway提供基于SO_REUSEPORT和SO_REUSEADDR特性的端口复用功能及基于iptables的端口复用功能

  在linux下可以大部分的功能端口

  在windows下不可复用iis，rdp端口，可以复用mysql，apache服务的端口

  nginx在默认状态下不可复用

SO_REUSEPORT和SO_REUSEADDR模式下示例：(若startnode端采用端口复用机制复用80端口)

  主要支持windows环境下的复用  

  Admin端：./stowaway_admin -c 192.168.0.105:80 -s 123 --rhostreuse

    命令解析：

    -c/-s 同上，不再赘述

    --rhostreuse 此选项被设置时，代表需要连接的节点正在端口复用的模式下运行(如果被连接的节点处于端口复用模式，必须设置此选项)

  此时startnode端： ./stowaway_agent -s 123 --startnode --report 80 --rehost 192.168.0.105

    命令解析：

    -s/--startnode同上

    --report 代表需要被复用的端口

    --rehost 代表复用端口时需要监听的本机ip（不可用0.0.0.0）

  此时如果后续有节点想要连接startnode: ./stowaway_agent -s 123 -m 192.168.0.105:80 --rhostreuse

  命令解析如admin，不再赘述

iptables模式下示例：(若startnode端采用端口复用机制复用22端口)

  主要支持linux环境下的复用，需要root权限

  此复用不是纯粹的“复用”，主要是靠设置iptables规则，使得流量在路由决策之前被处理，从而将流量从目标“report”导向“listenport(-l)”，递交给agent端进行处理分发，实现某种意义上的端口“复用”

startnode端： ./stowaway_agent -s 123 --startnode --report 22 -l 10000

    命令解析：

    -s/--startnode同上

    --report 代表需要被复用的端口

    -l 代表复用端口时需要监听的端口（渗透测试者所有访问report端口的流量将会导向这个端口）

在startnode启动后，使用script目录下的reuse.py

打开复用：python reuse.py --start --rhost 192.168.0.105 --rport 22

此时Admin端就可以连接：./stowaway_admin -c 192.168.0.105:22 -s 123 --rhostreuse

  命令解析：

    -c/-s 同上，不再赘述

    --rhostreuse 此选项被设置时，代表需要连接的节点正在端口复用的模式下运行(如果被连接的节点处于端口复用模式，必须设置此选项)

此时如果后续有节点想要连接startnode: ./stowaway_agent -s 123 -m 192.168.0.105:22 --rhostreuse

    命令解析如admin，不再赘述 

注意：如果startnode被ctrl-c或者kill命令杀死，程序将会自动清理iptables规则，但如果被kill -9 杀死，则无法自动清除

故而为了防止startnode异常退出后，iptables规则没有被清理导致被复用的服务无法访问，script目录下的reuse.py提供了关闭iptables规则的功能

当需要关闭时，运行：python reuse.py --stop --rhost 192.168.0.105 --rport 22

即可关闭转发规则，使得原服务能够被正常访问

```

**几个注意点：**

**1.除了admin节点以外，普通的agent以及startnode节点可以被多个agent端连接，以组成树状网络**

**2.当有节点掉线时，那么此时与此节点有关的socks，reflect，forward服务都会被强制停止**

**3.如因网络波动或中间节点掉线，导致某一个分支断开（举个例子，admin后接着startnode，startnode后分为两支，一支是startnode->node 2 -> node 3 -> node 4, 一支是startnode->node 5 ->node 6），那么如果node2掉线，node3及node4将不会掉线，而是继续保持存活。此时用户若想将node3及node4重新加入网络，那么用户有两种选择，一种是假如startnode可以直接访问node3，那么用户可随时在startnode将node3用connect或者sshtunnel命令重新加入网络（切记，就算startnode同时也可以访问node4，也请不要直接连接node4，请连接整个缺失链(node3->node4)的头节点node3），这样就可以将node3及node4重新加入网络；另一种选择是当startnode无法直接访问node3时（即必须经过node2），那么请先将node2重启并加入网络，之后再在node2上使用connect或者sshtunnel命令连接node3，从而将node3及node4加入网络（同样的，就算node2能直接访问node4，也请不要连接node4，连接node3即可）。**

## Example

一个简单的例子：

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
如果需要设置socks5用户名密码（Firefox支持，Chrome不支持），例如需要设置用户名为ph4ntom，密码为11235，则可将输入命令改为:socks 7777 ph4ntom 11235 (PS：切勿在用户名以及密码中使用冒号(:))

关闭socks5代理：

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/shutdownsocks5.png)

打开ssh：

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/ssh.png)

此时第二个普通节点会作为ssh客户端,(此节点)会发起ssh连接来访问指定的ssh服务，并将ssh数据回传至admin端

PS: 在ssh模式下，你可以用pwd来判断自己所处的文件夹（好吧，其实就是没法把banner传回来。。）

此时若还有节点需要加入网络，可使用sshtunnel命令：

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/waiting.png)
![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/sshtunnel.png)

此时将可以利用sshtunnel将节点加入网络，这一方法适用于当防火墙做了流量限制，只有ssh流量能够通过的情况（一般情况下推荐使用connect命令即可，不需要使用sshtunnel）。

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

如果想在node3上新开一个监听端口来让后续节点连接（原来如果启动时设置过-l选项，则之前的监听不会因此停止）

![node](https://github.com/ph4ntonn/Stowaway/blob/master/img/listen.png)
```
具体命令使用可输入help查询
```
## TODO

- [x] 节点间通信流量加密
- [x] 加强功能健壮性，解决了因为节点掉线、误操作带来的程序及网络崩溃
- [x] 对可使用的各项功能同时提供开启/关闭操作
- [x] 自动重连功能
- [ ] 清理代码，优化逻辑
- [x] 节点反向连接
- [x] 端口映射
- [x] 支持端口复用(感觉虽然效果不是特别好，但聊胜于无)

### 注意事项

- 此程序仅是闲暇时开发学习，结构及代码结构不够严谨，功能可能存在bug，请多多谅解
- 本程序编译出来稍微有一些大，但是实测实际占有内存空间并不会很大，大概会比尽量压缩文件大小的情况下多出0.5-1m左右（IOT平台应该也不差这0.5-1m。。，当然有时间我就把程序瘦瘦身 XD）
- admin不在线时，新节点将不允许加入
- 如需从源代码编译本项目，请运行build_admin.sh/build_agent.sh文件来编译对应类型的Stowaway(注意！！！！！！默认编译的是agent模式，此时请运行build_agent.sh,如需编译admin，请查看main.go文件中的提示，按照提示进行操作后，运行build_admin.sh文件)

### 致谢

- [rootkiter#Termite](https://github.com/rootkiter/Termite)
- [Venom](https://github.com/Dliv3/Venom)