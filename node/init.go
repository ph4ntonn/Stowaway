package node

import (
	"Stowaway/common"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

var NewNodeMessage []byte
var PeerNode string
var ControlConnForLowerNodeChan = make(chan net.Conn, 1)
var DataConnForLowerNodeChan = make(chan net.Conn, 1)
var NewNodeMessageChan = make(chan []byte, 1)

func StartNodeConn(monitor string, listenPort string, nodeID uint32, key []byte) (net.Conn, net.Conn, uint32, error) {
	controlConnToUpperNode, err := net.Dial("tcp", monitor)
	if err != nil {
		logrus.Error("Connection refused!")
		return controlConnToUpperNode, controlConnToUpperNode, 11235, err
	}
	respcommand, err := common.ConstructCommand("INIT", "FIRSTCONNECT", nodeID, key)
	if err != nil {
		logrus.Errorf("Error occured: %s", err)
	}
	_, err = controlConnToUpperNode.Write(respcommand)
	if err != nil {
		logrus.Errorf("Error occured: %s", err)
	}
	respcommand, err = common.ConstructCommand("LISTENPORT", listenPort, nodeID, key)
	if err != nil {
		logrus.Errorf("Error occured: %s", err)
	}
	_, err = controlConnToUpperNode.Write(respcommand)
	if err != nil {
		logrus.Errorf("Error occured: %s", err)
	}
	for {
		command, _ := common.ExtractCommand(controlConnToUpperNode, key)
		switch command.Command {
		case "ID":
			nodeID = command.NodeId
		case "ACCEPT":
			switch command.Info {
			case "DATA":
				dataConnToUpperNode, err := net.Dial("tcp", monitor)
				if err != nil {
					logrus.Errorf("ERROR OCCURED!: %s", err)
				}
				return controlConnToUpperNode, dataConnToUpperNode, nodeID, nil
			}
		}
	}
}

func StartNodeListen(listenPort string, NodeId uint32, nodeconnected string, key []byte) (net.Conn, net.Conn, []byte, error) {
	listenAddr := fmt.Sprintf("0.0.0.0:%s", listenPort)
	WaitingForLowerNode, err := net.Listen("tcp", listenAddr)

	var result [1]net.Conn

	if err != nil {
		logrus.Error("Cannot listen on port %s", listenPort)
		os.Exit(1)
	}
	for {
		ConnToLowerNode, err := WaitingForLowerNode.Accept() //判断一下是否是合法连接
		if err != nil {
			logrus.Error(err)
			continue
		}
		if nodeconnected == "0.0.0.0" {
			command, err := common.ExtractCommand(ConnToLowerNode, key)
			if err != nil {
				logrus.Error(err)
				continue
			}
			if command.Command == "INIT" {
				if command.NodeId == 0 {
					respNodeID := NodeId + 1
					respCommand, _ := common.ConstructCommand("ID", "", respNodeID, key)
					_, err := ConnToLowerNode.Write(respCommand)
					NewNodeMessage, _ = common.ConstructCommand("NEW", ConnToLowerNode.RemoteAddr().String(), respNodeID, key)
					if err != nil {
						logrus.Error(err)
						continue
					}
					controlConnToLowerNode := ConnToLowerNode
					result[0] = controlConnToLowerNode
					nodeconnected = strings.Split(ConnToLowerNode.RemoteAddr().String(), ":")[0]
					respCommand, _ = common.ConstructCommand("ACCEPT", "DATA", respNodeID, key)
					_, err = ConnToLowerNode.Write(respCommand)
					if err != nil {
						logrus.Error(err)
						continue
					}
				} else {
					respCommand, _ := common.ConstructCommand("ID", "", command.NodeId, key)
					_, err := ConnToLowerNode.Write(respCommand)
					if err != nil {
						logrus.Error(err)
						continue
					}
					respCommand, _ = common.ConstructCommand("ACCEPT", "DATA", command.NodeId, key)
					_, err = ConnToLowerNode.Write(respCommand)
					if err != nil {
						logrus.Error(err)
						continue
					}
					controlConnToLowerNode := ConnToLowerNode
					result[0] = controlConnToLowerNode
					nodeconnected = strings.Split(ConnToLowerNode.RemoteAddr().String(), ":")[0]
				}
			} else {
				logrus.Error("Illegal connection!")
			}
		} else if nodeconnected == strings.Split(ConnToLowerNode.RemoteAddr().String(), ":")[0] {
			dataConToLowerNode := ConnToLowerNode
			ControlConnForLowerNodeChan <- result[0]
			DataConnForLowerNodeChan <- dataConToLowerNode
			NewNodeMessageChan <- NewNodeMessage
			PeerNode = strings.Split(dataConToLowerNode.RemoteAddr().String(), ":")[0]
			nodeconnected = "0.0.0.0" //继续接受连接？
		} else {
			continue
		}
	}
}
