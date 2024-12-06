package cli

import (
	"sort"
)

type Helper struct {
	adminList []string
	nodeList  []string

	adminTree *tireTree
	nodeTree  *tireTree

	min int
	max int

	TaskChan   chan *HelperTask
	ResultChan chan []string
}

type HelperTask struct {
	IsNodeMode bool
	Uncomplete string
}

type tireNode struct {
	isEnd    bool
	children map[int]*tireNode
}

type tireTree struct {
	root *tireNode
}

func NewHelper() *Helper {
	helper := new(Helper)
	helper.adminList = []string{
		"use",
		"detail",
		"topo",
		"help",
		"exit",
	}

	helper.nodeList = []string{
		"help",
		"status",
		"listen",
		"addmemo",
		"delmemo",
		"ssh",
		"shell",
		"socks",
		"sshtunnel",
		"connect",
		"stopsocks",
		"upload",
		"download",
		"forward",
		"stopforward",
		"backward",
		"stopbackward",
		"shutdown",
		"back",
		"exit",
	}

	helper.min = 0
	helper.max = 11

	helper.adminTree = new(tireTree)
	helper.adminTree.root = new(tireNode)
	helper.adminTree.root.children = make(map[int]*tireNode)

	helper.nodeTree = new(tireTree)
	helper.nodeTree.root = new(tireNode)
	helper.nodeTree.root.children = make(map[int]*tireNode)

	helper.TaskChan = make(chan *HelperTask)
	helper.ResultChan = make(chan []string)

	return helper
}

func (helper *Helper) Run() {
	helper.insertAdmin()
	helper.insertNode()

	for {
		task := <-helper.TaskChan
		helper.ResultChan <- helper.search(task)
	}
}

func (helper *Helper) insertAdmin() {
	node := helper.adminTree.root
	for _, command := range helper.adminList {
		for i := 0; i < len(command); i++ {
			currentChar := int(command[i])
			if _, ok := node.children[currentChar]; !ok {
				node.children[currentChar] = new(tireNode)
				node.children[currentChar].children = make(map[int]*tireNode)
				node = node.children[currentChar]
				continue
			} else {
				node = node.children[currentChar]
				continue
			}
		}
		node.isEnd = true
		node = helper.adminTree.root
	}
}

func (helper *Helper) insertNode() {
	node := helper.nodeTree.root
	for _, command := range helper.nodeList {
		for i := 0; i < len(command); i++ {
			currentChar := int(command[i])
			if _, ok := node.children[currentChar]; !ok {
				node.children[currentChar] = new(tireNode)
				node.children[currentChar].children = make(map[int]*tireNode)
				node = node.children[currentChar]
				continue
			} else {
				node = node.children[currentChar]
				continue
			}
		}
		node.isEnd = true
		node = helper.nodeTree.root
	}
}

func (helper *Helper) search(task *HelperTask) []string {
	var complete []string
	var samePrefix string
	var node *tireNode

	if !task.IsNodeMode {
		node = helper.adminTree.root
	} else {
		node = helper.nodeTree.root
	}

	unComplete := task.Uncomplete

	if len(unComplete) < helper.min || len(unComplete) > helper.max {
		return complete
	}

	for i := 0; i < len(unComplete); i++ {
		currentChar := int(unComplete[i])
		if _, ok := node.children[currentChar]; ok {
			samePrefix = samePrefix + string(unComplete[i])
			node = node.children[currentChar]
			continue
		} else {
			return complete
		}
	}

	if samePrefix == "" {
		return complete
	}

	var diffSuffix []string
	var tSuffix string

	helper.getSuffix(node, &diffSuffix, tSuffix)

	for _, suffix := range diffSuffix {
		complete = append(complete, samePrefix+suffix)
	}

	sort.Strings(complete)

	return complete
}

func (helper *Helper) getSuffix(node *tireNode, suffix *[]string, tSuffix string) {
	if node.isEnd && len(node.children) != 0 {
		*suffix = append(*suffix, tSuffix)
	}

	if len(node.children) != 0 {
		for char := range node.children {
			ttSuffix := tSuffix + string(char)
			tNode := node.children[char]
			helper.getSuffix(tNode, suffix, ttSuffix)
		}
	} else {
		*suffix = append(*suffix, tSuffix)
	}
}
