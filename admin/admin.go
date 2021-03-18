/*
 * @Author: ph4ntom
 * @Date: 2021-03-08 14:35:02
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-03-18 16:41:10
 */
package main

import (
	"Stowaway/admin/process"
	"Stowaway/admin/topology"
	"log"
	"runtime"

	"Stowaway/admin/cli"
	"Stowaway/admin/initial"
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func main() {
	options := initial.ParseOptions()

	cli.Banner()

	admin := process.NewAdmin(options)

	topo := topology.NewTopology()
	go topo.Run()

	route := topology.NewRoute()
	go route.Run(topo)

	log.Println("[*]Waiting for new connection...")
	switch options.Mode {
	case initial.NORMAL_ACTIVE:
		admin.Conn = initial.NormalActive(options, topo)
	case initial.NORMAL_PASSIVE:
		admin.Conn = initial.NormalPassive(options, topo)
	default:
		log.Fatal("[*]Unknown Mode")
	}

	admin.Topology = topo
	admin.Route = route

	admin.Run()
}
