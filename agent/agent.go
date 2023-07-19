package main

import (
	"log"
	"net"
	"runtime"

	"Stowaway/agent/initial"
	"Stowaway/agent/process"
	"Stowaway/global"
	"Stowaway/protocol"
	"Stowaway/share"
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func main() {
	options := initial.ParseOptions()

	agent := process.NewAgent(options)

	protocol.DecideType(options.Upstream, options.Downstream)

	var conn net.Conn
	switch options.Mode {
	case initial.NORMAL_PASSIVE:
		conn, agent.UUID = initial.NormalPassive(options)
	case initial.NORMAL_RECONNECT_ACTIVE:
		fallthrough
	case initial.NORMAL_ACTIVE:
		conn, agent.UUID = initial.NormalActive(options, nil)
	case initial.SOCKS5_PROXY_RECONNECT_ACTIVE:
		fallthrough
	case initial.SOCKS5_PROXY_ACTIVE:
		proxy := share.NewSocks5Proxy(options.Connect, options.Socks5Proxy, options.Socks5ProxyU, options.Socks5ProxyP)
		conn, agent.UUID = initial.NormalActive(options, proxy)
	case initial.HTTP_PROXY_RECONNECT_ACTIVE:
		fallthrough
	case initial.HTTP_PROXY_ACTIVE:
		proxy := share.NewHTTPProxy(options.Connect, options.HttpProxy)
		conn, agent.UUID = initial.NormalActive(options, proxy)
	case initial.IPTABLES_REUSE_PASSIVE:
		defer initial.DeletePortReuseRules(options.Listen, options.ReusePort)
		conn, agent.UUID = initial.IPTableReusePassive(options)
	case initial.SO_REUSE_PASSIVE:
		conn, agent.UUID = initial.SoReusePassive(options)
	default:
		log.Fatal("[*] Unknown Mode")
	}

	global.InitialGComponent(conn, options.Secret, agent.UUID)

	agent.Run()
}
