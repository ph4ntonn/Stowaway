/*
 * @Author: ph4ntom
 * @Date: 2021-03-08 14:35:15
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-03-20 13:42:23
 */
package main

import (
	"log"
	"runtime"

	"Stowaway/agent/initial"
	"Stowaway/agent/process"
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func main() {
	options := initial.ParseOptions()

	agent := process.NewAgent(options)

	switch options.Mode {
	case initial.NORMAL_PASSIVE:
		agent.Conn, agent.ID = initial.NormalPassive(options)
	case initial.NORMAL_RECONNECT_ACTIVE:
		fallthrough
	case initial.NORMAL_ACTIVE:
		agent.Conn, agent.ID = initial.NormalActive(options)
	default:
		log.Fatal("[*]Unknown Mode")
	}

	agent.Run()
}
