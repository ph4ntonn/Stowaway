package node

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"Stowaway/utils"
)

const CHAIN_NAME = "STOWAWAY"

var START_FORWARDING string
var STOP_FORWARDING string

//以下大致与SO_REUSEPORT,SO_REUSEADDR模式下相同

/*-------------------------Iptable复用模式功能代码--------------------------*/

// SetForwardMessage 设置启动转发密钥
func SetForwardMessage(key []byte) {
	secret := utils.GetStringMd5(string(key))
	prefix := secret[8:16]
	start_suffix := secret[16:24]
	stop_suffix := utils.StringReverse(secret[16:24])
	START_FORWARDING = prefix + start_suffix
	STOP_FORWARDING = prefix + stop_suffix
}

// AcceptConnFromUpperNodeIPTableReuse 在iptable reuse状态下接收上一级节点的连接
func AcceptConnFromUpperNodeIPTableReuse(report, localPort string, nodeid string, key []byte) (net.Conn, string) {
	listenAddr := fmt.Sprintf("0.0.0.0:%s", localPort)
	waitingForConn, err := net.Listen("tcp", listenAddr)

	if err != nil {
		log.Fatalf("[*]Cannot reuse port %s", localPort)
	}
	for {
		comingConn, err := waitingForConn.Accept()
		if err != nil {
			log.Println("[*]", err)
			continue
		}

		err = CheckValid(comingConn, true, report)
		if err != nil {
			continue
		}

		utils.ExtractPayload(comingConn, key, utils.AdminId, true)

		utils.ConstructPayloadAndSend(comingConn, nodeid, "", "COMMAND", "INIT", " ", report, 0, utils.AdminId, key, false)

		command, _ := utils.ExtractPayload(comingConn, key, utils.AdminId, true) //等待分配id
		if command.Command == "ID" {
			nodeid = command.NodeId
			waitingForConn.Close()
			return comingConn, nodeid
		}

	}
}

// StartNodeListenIPTableReuse 初始化节点监听操作
func StartNodeListenIPTableReuse(report, localPort string, nodeid string, key []byte) {
	var newNodeMessage []byte

	if localPort == "" { //如果没有port，直接退出
		return
	}

	listenAddr := fmt.Sprintf("0.0.0.0:%s", localPort)
	waitingForLowerNode, err := net.Listen("tcp", listenAddr)

	if err != nil {
		log.Fatalf("[*]Cannot listen on port %s", localPort)
	}

	for {
		connToLowerNode, err := waitingForLowerNode.Accept()
		if err != nil {
			log.Println("[*]", err)
			return
		}

		err = CheckValid(connToLowerNode, true, report)
		if err != nil {
			continue
		}

		for i := 0; i < 2; i++ {
			command, _ := utils.ExtractPayload(connToLowerNode, key, utils.AdminId, true)
			switch command.Command {
			case "STOWAWAYADMIN":
				utils.ConstructPayloadAndSend(connToLowerNode, nodeid, "", "COMMAND", "INIT", " ", report, 0, utils.AdminId, key, false)
			case "ID":
				NodeStuff.ControlConnForLowerNodeChan <- connToLowerNode
				NodeStuff.NewNodeMessageChan <- newNodeMessage
				NodeStuff.IsAdmin <- true
			case "REONLINESUC":
				NodeStuff.Adminconn <- connToLowerNode
			case "STOWAWAYAGENT":
				if !NodeStuff.Offline {
					utils.ConstructPayloadAndSend(connToLowerNode, nodeid, "", "COMMAND", "CONFIRM", " ", " ", 0, nodeid, key, false)
				} else {
					utils.ConstructPayloadAndSend(connToLowerNode, nodeid, "", "COMMAND", "REONLINE", " ", report, 0, nodeid, key, false)
				}
			case "INIT":
				newNodeMessage, _ = utils.ConstructPayload(utils.AdminId, "", "COMMAND", "NEW", " ", connToLowerNode.RemoteAddr().String(), 0, nodeid, key, false)
				NodeInfo.LowerNode.Payload[utils.AdminId] = connToLowerNode
				NodeStuff.ControlConnForLowerNodeChan <- connToLowerNode
				NodeStuff.NewNodeMessageChan <- newNodeMessage
				NodeStuff.IsAdmin <- false
			}
		}
	}
}

/*-------------------------Iptable复用模式主要功能代码--------------------------*/

// DeletePortReuseRules 删除iptable规则
func DeletePortReuseRules(localPort string, reusedPort string) error {
	var cmds []string

	cmds = append(cmds, fmt.Sprintf("iptables -t nat -D PREROUTING -p tcp --dport %s --syn -m recent --rcheck --seconds 3600 --name %s --rsource -j %s", reusedPort, strings.ToLower(CHAIN_NAME), CHAIN_NAME))
	cmds = append(cmds, fmt.Sprintf("iptables -D INPUT -p tcp -m string --string %s --algo bm -m recent --name %s --remove -j ACCEPT", STOP_FORWARDING, strings.ToLower(CHAIN_NAME)))
	cmds = append(cmds, fmt.Sprintf("iptables -D INPUT -p tcp -m string --string %s --algo bm -m recent --set --name %s --rsource -j ACCEPT", START_FORWARDING, strings.ToLower(CHAIN_NAME)))
	cmds = append(cmds, fmt.Sprintf("iptables -t nat -F %s", CHAIN_NAME))
	cmds = append(cmds, fmt.Sprintf("iptables -t nat -X %s", CHAIN_NAME))

	for _, each := range cmds {
		cmd := strings.Split(each, " ")
		err := exec.Command(cmd[0], cmd[1:]...).Run() //删除规则
		if err != nil {
			log.Println("[*]Error!Use the '" + each + "' to delete rules.")
		}
	}

	fmt.Println("[*]All rules have been cleared successfully!")

	return nil
}

// SetPortReuseRules 添加iptable规则
func SetPortReuseRules(localPort string, reusedPort string) error {
	sigs := make(chan os.Signal, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM) //监听ctrl+c、kill命令
	go func() {
		for {
			<-sigs
			DeletePortReuseRules(localPort, reusedPort)
			os.Exit(0)
		}
	}()

	var cmds []string
	cmds = append(cmds, fmt.Sprintf("iptables -t nat -N %s", CHAIN_NAME))                                                                                                                                      //新建自定义链
	cmds = append(cmds, fmt.Sprintf("iptables -t nat -A %s -p tcp -j REDIRECT --to-port %s", CHAIN_NAME, localPort))                                                                                           //将自定义链定义为转发流量至自定义监听端口
	cmds = append(cmds, fmt.Sprintf("iptables -A INPUT -p tcp -m string --string %s --algo bm -m recent --set --name %s --rsource -j ACCEPT", START_FORWARDING, strings.ToLower(CHAIN_NAME)))                  //设置当有一个报文带着特定字符串经过INPUT链时，将此报文的源地址加入一个特定列表中
	cmds = append(cmds, fmt.Sprintf("iptables -A INPUT -p tcp -m string --string %s --algo bm -m recent --name %s --remove -j ACCEPT", STOP_FORWARDING, strings.ToLower(CHAIN_NAME)))                          //设置当有一个报文带着特定字符串经过INPUT链时，将此报文的源地址从一个特定列表中移除
	cmds = append(cmds, fmt.Sprintf("iptables -t nat -A PREROUTING -p tcp --dport %s --syn -m recent --rcheck --seconds 3600 --name %s --rsource -j %s", reusedPort, strings.ToLower(CHAIN_NAME), CHAIN_NAME)) // 设置当有任意报文访问指定的复用端口时，检查特定列表，如果此报文的源地址在特定列表中且不超过3600秒，则执行自定义链

	for _, each := range cmds {
		cmd := strings.Split(each, " ")
		err := exec.Command(cmd[0], cmd[1:]...).Run() //添加规则
		if err != nil {
			fmt.Println(each)
			return err
		}
	}

	return nil
}
