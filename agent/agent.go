/*
 * @Author: ph4ntom
 * @Date: 2021-03-08 14:35:15
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-03-20 16:34:07
 */
package main

import (
	"log"
	"net"
	"runtime"

	"Stowaway/agent/initial"
	"Stowaway/agent/process"
	"Stowaway/global"
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func main() {
	options := initial.ParseOptions()

	agent := process.NewAgent()

	var conn net.Conn
	switch options.Mode {
	case initial.NORMAL_PASSIVE:
		conn, agent.UUID = initial.NormalPassive(options)
	case initial.NORMAL_RECONNECT_ACTIVE:
		fallthrough
	case initial.NORMAL_ACTIVE:
		conn, agent.UUID = initial.NormalActive(options, nil)
	case initial.PROXY_ACTIVE:
		proxy := initial.NewProxy(options.Connect, options.Proxy, options.ProxyU, options.ProxyP)
		conn, agent.UUID = initial.NormalActive(options, proxy)
	default:
		log.Fatal("[*]Unknown Mode")
	}

	global.InitialGComponent(conn, options.Secret, agent.UUID)

	agent.Run()
}
