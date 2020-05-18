package agent

import (
	"fmt"
	"net"

	"Stowaway/node"
	"Stowaway/utils"
)

//管理下行节点代码

// HandleConnToLowerNode 管理发往下级节点的信道
func HandleConnToLowerNode() {
	for {
		proxyData := <-AgentStuff.ProxyChan.ProxyChanToLowerNode

		node.NodeInfo.LowerNode.Lock()
		if _, ok := node.NodeInfo.LowerNode.Payload[proxyData.Route]; ok { //检查此节点是否存活，防止admin误操作在已掉线的节点输入命令导致节点panic
			node.NodeInfo.LowerNode.Payload[proxyData.Route].Write(proxyData.Data)
		}
		node.NodeInfo.LowerNode.Unlock()
	}

}

// HandleConnFromLowerNode 处理下级节点发往上级节点的信息
func HandleConnFromLowerNode(connForLowerNode net.Conn, currentid, lowerid string) {
	for {
		command, err := utils.ExtractPayload(connForLowerNode, AgentStatus.AESKey, currentid, false)
		if err != nil {
			connForLowerNode.Close()
			node.NodeInfo.LowerNode.Lock()
			delete(node.NodeInfo.LowerNode.Payload, lowerid) //下级节点掉线，立即将此节点从自己的子节点列表删除
			node.NodeInfo.LowerNode.Unlock()
			offlineMess, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", "AGENTOFFLINE", " ", lowerid, 0, currentid, AgentStatus.AESKey, false) //通知admin下级节点已经下线
			AgentStuff.ProxyChan.ProxyChanToUpperNode <- offlineMess
			return
		}
		switch command.Type {
		case "COMMAND":
			switch command.Command {
			case "RECONNID":
				if _, ok := node.NodeInfo.LowerNode.Payload[command.CurrentId]; ok {
					info := fmt.Sprintf("%s:::%s", currentid, connForLowerNode.RemoteAddr().String())
					proxyCommand, _ := utils.ConstructPayload(utils.AdminId, "", "COMMAND", command.Command, " ", info, 0, command.CurrentId, AgentStatus.AESKey, false)
					AgentStuff.ProxyChan.ProxyChanToUpperNode <- proxyCommand
					continue
				} else {
					proxyCommand, _ := utils.ConstructPayload(
						command.NodeId,
						command.Route,
						command.Type,
						command.Command,
						command.FileSliceNum,
						command.Info,
						command.Clientid,
						command.CurrentId,
						AgentStatus.AESKey,
						true,
					)
					AgentStuff.ProxyChan.ProxyChanToUpperNode <- proxyCommand
					continue
				}
			case "HEARTBEAT":
				hbcommpack, _ := utils.ConstructPayload(command.CurrentId, "", "COMMAND", "KEEPALIVE", " ", " ", 0, currentid, AgentStatus.AESKey, false)
				passToLowerData := utils.NewPassToLowerNodeData()
				passToLowerData.Data = hbcommpack
				passToLowerData.Route = command.CurrentId
				AgentStuff.ProxyChan.ProxyChanToLowerNode <- passToLowerData
				continue
			default:
				proxyData, _ := utils.ConstructPayload(
					command.NodeId,
					command.Route,
					command.Type,
					command.Command,
					command.FileSliceNum,
					command.Info,
					command.Clientid,
					command.CurrentId,
					AgentStatus.AESKey,
					true,
				)
				AgentStuff.ProxyChan.ProxyChanToUpperNode <- proxyData
			}
		case "DATA":
			proxyData, _ := utils.ConstructPayload(
				command.NodeId,
				command.Route,
				command.Type,
				command.Command,
				command.FileSliceNum,
				command.Info,
				command.Clientid,
				command.CurrentId,
				AgentStatus.AESKey,
				true,
			)
			AgentStuff.ProxyChan.ProxyChanToUpperNode <- proxyData
		}
	}
}

//管理下行节点代码结束
