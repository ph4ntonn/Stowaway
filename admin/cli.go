package admin

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

var (
	AdminCommandChan = make(chan []string)
	Nodes            = make(map[uint32]string)
)

// 启动控制台
func Controlpanel() {
	inputReader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("(%s) >> ", *CliStatus)
		input, err := inputReader.ReadString('\n')
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		command := strings.Replace(input, "\n", "", -1)
		execCommand := strings.Split(command, " ")
		AdminCommandChan <- execCommand

		<-ReadyChange
		<-IsShellMode
	}
}

// 显示节点拓扑信息
func ShowChain() {
	if StartNode != "0.0.0.0" {
		fmt.Printf("StartNode:[1] %s\n", StartNode)
		for Nodeid, Nodeaddress := range Nodes {
			id := fmt.Sprint(Nodeid)
			fmt.Printf("Nodes [%s]: %s\n", id, Nodeaddress)
		}
	} else {
		fmt.Println("There is no agent connected!")
	}

}

// 将节点加入拓扑
func AddToChain() {
	for {
		newNode := <-NodesReadyToadd
		for key, value := range newNode {
			Nodes[key] = value
		}
	}
}
