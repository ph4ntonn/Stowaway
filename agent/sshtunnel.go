package agent

import (
	"Stowaway/common"
	"Stowaway/node"
	"fmt"
	"strings"

	"golang.org/x/crypto/ssh"
)

//利用sshtunnel来连接下一个节点，以此在防火墙限制流量时仍然可以进行穿透
func SshTunnelNextNode(info string, nodeid string) error {
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
			sshMess, _ := common.ConstructPayload(common.AdminId, "", "COMMAND", "SSHCERTERROR", " ", " ", 0, nodeid, AgentStatus.AESKey, false)
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
		sshMess, _ := common.ConstructPayload(common.AdminId, "", "COMMAND", "SSHTUNNELRESP", " ", "FAILED", 0, nodeid, AgentStatus.AESKey, false)
		ProxyChan.ProxyChanToUpperNode <- sshMess
		return err
	}

	nodeConn, err := SshClient.Dial("tcp", fmt.Sprintf("127.0.0.1:%s", lport))

	if err != nil {
		sshMess, _ := common.ConstructPayload(common.AdminId, "", "COMMAND", "SSHTUNNELRESP", " ", "FAILED", 0, nodeid, AgentStatus.AESKey, false)
		ProxyChan.ProxyChanToUpperNode <- sshMess
		return err
	}

	helloMess, _ := common.ConstructPayload(nodeid, "", "COMMAND", "STOWAWAYAGENT", " ", " ", 0, common.AdminId, AgentStatus.AESKey, false)
	nodeConn.Write(helloMess)
	for {
		command, err := common.ExtractPayload(nodeConn, AgentStatus.AESKey, common.AdminId, true)
		if err != nil {
			sshMess, _ := common.ConstructPayload(common.AdminId, "", "COMMAND", "SSHTUNNELRESP", " ", "FAILED", 0, nodeid, AgentStatus.AESKey, false)
			ProxyChan.ProxyChanToUpperNode <- sshMess
			return err
		}
		switch command.Command {
		case "INIT":
			NewNodeMessage, _ := common.ConstructPayload(common.AdminId, "", "COMMAND", "NEW", " ", nodeConn.RemoteAddr().String(), 0, nodeid, AgentStatus.AESKey, false)
			node.NodeInfo.LowerNode.Payload[common.AdminId] = nodeConn
			node.ControlConnForLowerNodeChan <- nodeConn
			node.NewNodeMessageChan <- NewNodeMessage
			node.IsAdmin <- false
			sshMess, _ := common.ConstructPayload(common.AdminId, "", "COMMAND", "SSHTUNNELRESP", " ", "SUCCESS", 0, nodeid, AgentStatus.AESKey, false)
			ProxyChan.ProxyChanToUpperNode <- sshMess
			return nil
		case "REONLINE":
			//普通节点重连
			node.ReOnlineId <- command.CurrentId
			node.ReOnlineConn <- nodeConn
			<-node.PrepareForReOnlineNodeReady
			NewNodeMessage, _ := common.ConstructPayload(nodeid, "", "COMMAND", "REONLINESUC", " ", " ", 0, nodeid, AgentStatus.AESKey, false)
			nodeConn.Write(NewNodeMessage)
			sshMess, _ := common.ConstructPayload(common.AdminId, "", "COMMAND", "SSHTUNNELRESP", " ", "SUCCESS", 0, nodeid, AgentStatus.AESKey, false)
			ProxyChan.ProxyChanToUpperNode <- sshMess
			return nil
		}
	}
}
