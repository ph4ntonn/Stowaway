//go:build !windows

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

	"github.com/nsf/termbox-go"
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func main() {
	printer.InitPrinter()

	termbox.Init()
	termbox.SetCursor(0, 0)
	termbox.Flush()

	go listenCtrlC()

	options := initial.ParseOptions()

	cli.Banner()

	share.GeneratePreAuthToken(options.Secret)

	protocol.SetUpDownStream("raw", options.Downstream)

	topo := topology.NewTopology()
	go topo.Run()

	printer.Warning("[*] Waiting for new connection...\r\n")
	var conn net.Conn
	switch options.Mode {
	case initial.NORMAL_ACTIVE:
		conn = initial.NormalActive(options, topo, nil)
	case initial.NORMAL_PASSIVE:
		conn = initial.NormalPassive(options, topo)
	case initial.SOCKS5_PROXY_ACTIVE:
		proxy := share.NewSocks5Proxy(options.Connect, options.Socks5Proxy, options.Socks5ProxyU, options.Socks5ProxyP)
		conn = initial.NormalActive(options, topo, proxy)
	case initial.HTTP_PROXY_ACTIVE:
		proxy := share.NewHTTPProxy(options.Connect, options.HttpProxy)
		conn = initial.NormalActive(options, topo, proxy)
	default:
		printer.Fail("[*] Unknown Mode")
		os.Exit(0)
	}

	// kill listenCtrlC
	termbox.Interrupt()

	admin := process.NewAdmin(options, topo)

	topoTask := &topology.TopoTask{
		Mode: topology.CALCULATE,
	}
	topo.TaskChan <- topoTask
	<-topo.ResultChan

	global.InitialGComponent(conn, options.Secret, protocol.ADMIN_UUID)

	admin.Run()
}

// let process exit if nothing connected
func listenCtrlC() {
	for {
		event := termbox.PollEvent()
		if event.Type == termbox.EventInterrupt {
			break
		}

		if event.Key == termbox.KeyCtrlC {
			termbox.Close()
			os.Exit(0)
		}
	}
}
