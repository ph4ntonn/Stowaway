package main

import (
	"net"
	"os"
	"runtime"

	"Stowaway/admin/cli"
	"Stowaway/admin/initial"
	"Stowaway/admin/printer"
	"Stowaway/admin/process"
	"Stowaway/admin/topology"
	"Stowaway/global"
	"Stowaway/protocol"
	"Stowaway/share"
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func main() {
	printer.InitPrinter()

	options := initial.ParseOptions()

	protocol.DecideType(options.Upstream, options.Downstream)

	cli.Banner()

	topo := topology.NewTopology()
	go topo.Run()

	printer.Warning("[*] Waiting for new connection...\n")
	var conn net.Conn
	switch options.Mode {
	case initial.NORMAL_ACTIVE:
		conn = initial.NormalActive(options, topo, nil)
	case initial.NORMAL_PASSIVE:
		conn = initial.NormalPassive(options, topo)
	case initial.PROXY_ACTIVE:
		proxy := share.NewProxy(options.Connect, options.Proxy, options.ProxyU, options.ProxyP)
		conn = initial.NormalActive(options, topo, proxy)
	default:
		printer.Fail("[*] Unknown Mode")
		os.Exit(0)
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
