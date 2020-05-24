package agent

import (
	"fmt"
	"strings"

	"Stowaway/node"
	"Stowaway/utils"

	"golang.org/x/crypto/ssh"
)

// SSHTunnelNextNode 利用sshtunnel来连接下一个节点，以此在防火墙限制流量时仍然可以进行穿透
func SSHTunnelNextNode(info string, nodeid string) error {
	var authPayload ssh.AuthMethod
	spiltedInfo := strings.Split(info, ":::")

	host := spiltedInfo[0]
	username := spiltedInfo[1]
	authWay := spiltedInfo[2]
	lport := spiltedInfo[3]
	method := spiltedInfo[4]

	if method == "1" {
		authPayload = ssh.Password(authWay)
	} else if method == "2" {
		key, err := ssh.ParsePrivateKey([]byte(authWay))
		if err != nil {
			sshMess, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "SSHCERTERROR", " ", " ", 0, nodeid, AgentStatus.AESKey, false)
			AgentStuff.ProxyChan.ProxyChanToUpperNode <- sshMess
			return err
		}
		authPayload = ssh.PublicKeys(key)
	}

	sshClient, err := ssh.Dial("tcp", host, &ssh.ClientConfig{
		User:            username,
		Auth:            []ssh.AuthMethod{authPayload},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	})
	if err != nil {
		sshMess, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "SSHTUNNELRESP", " ", "FAILED", 0, nodeid, AgentStatus.AESKey, false)
		AgentStuff.ProxyChan.ProxyChanToUpperNode <- sshMess
		return err
	}

	nodeConn, err := sshClient.Dial("tcp", fmt.Sprintf("127.0.0.1:%s", lport))
	if err != nil {
		sshMess, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "SSHTUNNELRESP", " ", "FAILED", 0, nodeid, AgentStatus.AESKey, false)
		AgentStuff.ProxyChan.ProxyChanToUpperNode <- sshMess
		nodeConn.Close()
		return err
	}

	err = node.SendSecret(nodeConn, AgentStatus.AESKey)
	if err != nil {
		sshMess, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "SSHTUNNELRESP", " ", "FAILED", 0, nodeid, AgentStatus.AESKey, false)
		AgentStuff.ProxyChan.ProxyChanToUpperNode <- sshMess
		nodeConn.Close()
		return err
	}

	utils.ConstructPayloadAndSend(nodeConn, nodeid, "", "COMMAND", "STOWAWAYAGENT", " ", " ", 0, utils.AdminId, AgentStatus.AESKey, false)

	for {
		command, err := utils.ExtractPayload(nodeConn, AgentStatus.AESKey, utils.AdminId, true)
		if err != nil {
			sshMess, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "SSHTUNNELRESP", " ", "FAILED", 0, nodeid, AgentStatus.AESKey, false)
			AgentStuff.ProxyChan.ProxyChanToUpperNode <- sshMess
			nodeConn.Close()
			return err
		}
		switch command.Command {
		case "INIT":
			addr := strings.Split(sshClient.RemoteAddr().String(), ":")[0] + ":" + lport
			newNodeMessage, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "NEW", " ", addr, 0, nodeid, AgentStatus.AESKey, false)
			node.NodeInfo.LowerNode.Payload[utils.AdminId] = nodeConn
			node.NodeStuff.ControlConnForLowerNodeChan <- nodeConn
			node.NodeStuff.NewNodeMessageChan <- newNodeMessage
			node.NodeStuff.IsAdmin <- false

			sshMess, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "SSHTUNNELRESP", " ", "SUCCESS", 0, nodeid, AgentStatus.AESKey, false)
			AgentStuff.ProxyChan.ProxyChanToUpperNode <- sshMess

			return nil
		case "REONLINE":
			//普通节点重连
			node.NodeStuff.ReOnlineID <- command.CurrentId
			node.NodeStuff.ReOnlineConn <- nodeConn

			<-node.NodeStuff.PrepareForReOnlineNodeReady

			utils.ConstructPayloadAndSend(nodeConn, nodeid, "", "COMMAND", "REONLINESUC", " ", " ", 0, nodeid, AgentStatus.AESKey, false)

			sshMess, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "SSHTUNNELRESP", " ", "SUCCESS", 0, nodeid, AgentStatus.AESKey, false)
			AgentStuff.ProxyChan.ProxyChanToUpperNode <- sshMess

			return nil
		}
	}
}
