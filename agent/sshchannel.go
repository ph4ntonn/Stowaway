package agent

import (
	"Stowaway/common"
	"Stowaway/node"
	"fmt"
	"strings"

	"golang.org/x/crypto/ssh"
)

//利用sshtunnel来连接下一个节点，以此在防火墙限制流量时仍然可以进行穿透
func SshTunnelNextNode(info string, nodeid uint32) error {
	var authpayload ssh.AuthMethod
	spiltedinfo := strings.Split(info, ":::")
	host := spiltedinfo[0]
	username := spiltedinfo[1]
	authway := spiltedinfo[2]
	lport := spiltedinfo[3]
	method := spiltedinfo[4]

	if method == "1" {
		authpayload = ssh.Password(authway)
	} else if method == "2" {
		key, err := ssh.ParsePrivateKey([]byte(authway))
		if err != nil {
			sshMess, _ := common.ConstructPayload(0, "", "COMMAND", "SSHCERTERROR", " ", " ", 0, nodeid, AgentStatus.AESKey, false)
			ProxyChan.ProxyChanToUpperNode <- sshMess
			return err
		}
		authpayload = ssh.PublicKeys(key)
	}

	SshClient, err := ssh.Dial("tcp", host, &ssh.ClientConfig{
		User:            username,
		Auth:            []ssh.AuthMethod{authpayload},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	})
	if err != nil {
		sshMess, _ := common.ConstructPayload(0, "", "COMMAND", "SSHTUNNELRESP", " ", "FAILED", 0, nodeid, AgentStatus.AESKey, false)
		ProxyChan.ProxyChanToUpperNode <- sshMess
		return err
	}

	nodeConn, err := SshClient.Dial("tcp", fmt.Sprintf("127.0.0.1:%s", lport))

	if err != nil {
		sshMess, _ := common.ConstructPayload(0, "", "COMMAND", "SSHTUNNELRESP", " ", "FAILED", 0, nodeid, AgentStatus.AESKey, false)
		ProxyChan.ProxyChanToUpperNode <- sshMess
		return err
	}

	helloMess, _ := common.ConstructPayload(nodeid, "", "COMMAND", "STOWAWAYAGENT", " ", " ", 0, 0, AgentStatus.AESKey, false)
	nodeConn.Write(helloMess)
	for {
		command, err := common.ExtractPayload(nodeConn, AgentStatus.AESKey, 0, true)
		if err != nil {
			sshMess, _ := common.ConstructPayload(0, "", "COMMAND", "SSHTUNNELRESP", " ", "FAILED", 0, nodeid, AgentStatus.AESKey, false)
			ProxyChan.ProxyChanToUpperNode <- sshMess
			return err
		}
		switch command.Command {
		case "INIT":
			NewNodeMessage, _ := common.ConstructPayload(0, "", "COMMAND", "NEW", " ", nodeConn.RemoteAddr().String(), 0, nodeid, AgentStatus.AESKey, false)
			node.NodeInfo.LowerNode.Payload[0] = nodeConn
			node.ControlConnForLowerNodeChan <- nodeConn
			node.NewNodeMessageChan <- NewNodeMessage
			node.IsAdmin <- false
			sshMess, _ := common.ConstructPayload(0, "", "COMMAND", "SSHTUNNELRESP", " ", "SUCCESS", 0, nodeid, AgentStatus.AESKey, false)
			ProxyChan.ProxyChanToUpperNode <- sshMess
			return nil
		}
	}
}
