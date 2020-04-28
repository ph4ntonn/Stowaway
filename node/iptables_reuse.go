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
const START_FORWARDING = "stowawaycoming"
const STOP_FORWARDING = "stowawayleaving"

/*-------------------------Iptable复用模式功能代码--------------------------*/
//以下大致与SO_REUSEPORT,SO_REUSEADDR模式下相同
func AcceptConnFromUpperNodeIPTableReuse(report, localport string, nodeid string, key []byte) (net.Conn, string) {
	listenAddr := fmt.Sprintf("0.0.0.0:%s", localport)
	WaitingForConn, err := net.Listen("tcp", listenAddr)

	if err != nil {
		log.Printf("[*]Cannot reuse port %s", localport)
		os.Exit(0)
	}
	for {
		Comingconn, err := WaitingForConn.Accept()
		if err != nil {
			log.Println("[*]", err)
			continue
		}

		err = CheckValid(Comingconn, true, report)
		if err != nil {
			continue
		}

		utils.ExtractPayload(Comingconn, key, utils.AdminId, true)

		respcommand, _ := utils.ConstructPayload(nodeid, "", "COMMAND", "INIT", " ", report, 0, utils.AdminId, key, false)
		Comingconn.Write(respcommand)

		command, _ := utils.ExtractPayload(Comingconn, key, utils.AdminId, true) //等待分配id
		if command.Command == "ID" {
			nodeid = command.NodeId
			WaitingForConn.Close()
			return Comingconn, nodeid
		}

	}
}

//初始化节点监听操作
func StartNodeListenIPTableReuse(report, localport string, NodeId string, key []byte) {
	var NewNodeMessage []byte

	if localport == "" { //如果没有port，直接退出
		return
	}

	listenAddr := fmt.Sprintf("0.0.0.0:%s", localport)
	WaitingForLowerNode, err := net.Listen("tcp", listenAddr)

	if err != nil {
		log.Printf("[*]Cannot listen on port %s", localport)
		os.Exit(0)
	}

	for {
		ConnToLowerNode, err := WaitingForLowerNode.Accept()
		if err != nil {
			log.Println("[*]", err)
			return
		}

		err = CheckValid(ConnToLowerNode, true, report)
		if err != nil {
			continue
		}

		for i := 0; i < 2; i++ {
			command, _ := utils.ExtractPayload(ConnToLowerNode, key, utils.AdminId, true)
			switch command.Command {
			case "STOWAWAYADMIN":
				respcommand, _ := utils.ConstructPayload(NodeId, "", "COMMAND", "INIT", " ", report, 0, utils.AdminId, key, false)
				ConnToLowerNode.Write(respcommand)
			case "ID":
				NodeStuff.ControlConnForLowerNodeChan <- ConnToLowerNode
				NodeStuff.NewNodeMessageChan <- NewNodeMessage
				NodeStuff.IsAdmin <- true
			case "REONLINESUC":
				NodeStuff.Adminconn <- ConnToLowerNode
			case "STOWAWAYAGENT":
				if !NodeStuff.Offline {
					NewNodeMessage, _ = utils.ConstructPayload(NodeId, "", "COMMAND", "CONFIRM", " ", " ", 0, NodeId, key, false)
					ConnToLowerNode.Write(NewNodeMessage)
				} else {
					respcommand, _ := utils.ConstructPayload(NodeId, "", "COMMAND", "REONLINE", " ", report, 0, NodeId, key, false)
					ConnToLowerNode.Write(respcommand)
				}
			case "INIT":
				NewNodeMessage, _ = utils.ConstructPayload(utils.AdminId, "", "COMMAND", "NEW", " ", ConnToLowerNode.RemoteAddr().String(), 0, NodeId, key, false)

				NodeInfo.LowerNode.Payload[utils.AdminId] = ConnToLowerNode
				NodeStuff.ControlConnForLowerNodeChan <- ConnToLowerNode
				NodeStuff.NewNodeMessageChan <- NewNodeMessage
				NodeStuff.IsAdmin <- false
			}
		}
	}
}

/*-------------------------Iptable复用模式主要功能代码--------------------------*/
//删除iptable规则
func DeletePortReuseRules(localPort string, reusedPort string) error {
	var cmds []string

	cmds = append(cmds, fmt.Sprintf("iptables -t nat -D PREROUTING -p tcp --dport %s --syn -m recent --rcheck --seconds 3600 --name %s --rsource -j %s", reusedPort, strings.ToLower(CHAIN_NAME), CHAIN_NAME))
	cmds = append(cmds, fmt.Sprintf("iptables -D INPUT -p tcp -m string --string %s --algo bm -m recent --name %s --remove -j ACCEPT", STOP_FORWARDING, strings.ToLower(CHAIN_NAME)))
	cmds = append(cmds, fmt.Sprintf("iptables -D INPUT -p tcp -m string --string %s --algo bm -m recent --set --name %s --rsource -j ACCEPT", START_FORWARDING, strings.ToLower(CHAIN_NAME)))
	cmds = append(cmds, fmt.Sprintf("iptables -t nat -F %s", CHAIN_NAME))
	cmds = append(cmds, fmt.Sprintf("iptables -t nat -X %s", CHAIN_NAME))

	for _, each := range cmds {
		cmd := strings.Split(each, " ")
		err := exec.Command(cmd[0], cmd[1:]...).Run() //添加规则
		if err != nil {
			log.Println("[*]Error!Use the '" + each + "' to delete rules.")
		}
	}

	fmt.Println("[*]All rules have been cleared successfully!")

	return nil
}

//添加iptable规则
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
		err := exec.Command(cmd[0], cmd[1:]...).Run() //删除规则
		if err != nil {
			fmt.Println(each)
			return err
		}
	}

	return nil
}
