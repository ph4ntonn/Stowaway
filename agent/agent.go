/*
 * @Author: ph4ntom
 * @Date: 2021-03-08 14:35:15
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-03-16 19:21:38
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

	agent := new(process.Agent)
	agent.Prepare(options)

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
