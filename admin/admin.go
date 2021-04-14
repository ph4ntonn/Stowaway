/*
 * @Author: ph4ntom
 * @Date: 2021-03-08 14:35:02
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-03-30 16:16:44
 */
package main

import (
	"Stowaway/admin/process"
	"Stowaway/admin/topology"
	"Stowaway/global"
	"Stowaway/protocol"
	"log"
	"net"
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

	topo := topology.NewTopology()
	go topo.Run()

	log.Println("[*]Waiting for new connection...")
	var conn net.Conn
	switch options.Mode {
	case initial.NORMAL_ACTIVE:
		conn = initial.NormalActive(options, topo)
	case initial.NORMAL_PASSIVE:
		conn = initial.NormalPassive(options, topo)
	default:
		log.Fatal("[*]Unknown Mode")
	}

	admin := process.NewAdmin()

	admin.Topology = topo

	topoTask := &topology.TopoTask{
		Mode: topology.CALCULATE,
	}
	topo.TaskChan <- topoTask
	<-topo.ResultChan

	global.InitialGComponent(conn, options.Secret, protocol.ADMIN_UUID)

	admin.Run()
}
