package config

const VERSION = "1.6" //版本号

const VALIDMESSAGE = "STOWAWAY"      //reuse模式下发送的特征数据
const READYMESSAGE = "STOWAWAYREADY" //同上

const INFO_LEN = 64         //数据包实际承载信息长度
const TYPE_LEN = 4          //数据包类型长度
const NODE_LEN = 10         //数据包节点标号实际长度
const CLIENT_LEN = 16       //数据包clientid长度
const FILESLICENUM_LEN = 64 //数据包文件分片标号长度
const COMMAND_LEN = 5       //数据包命令类型长度
const ROUTE_LEN = 16        //数据包路由表长度
