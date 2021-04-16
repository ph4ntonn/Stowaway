package initial

import (
	"errors"
	"flag"
	"log"
	"net"
	"os"
)

const (
	NORMAL_ACTIVE = iota
	PROXY_ACTIVE
	REUSE_ACTIVE
	NORMAL_RECONNECT_ACTIVE
	PROXY_RECONNECT_ACTIVE
	PROXY_REUSE_ACTIVE
	NORMAL_PASSIVE
	SO_REUSE_PASSIVE
	IPTABLES_REUSE_PASSIVE
)

type Options struct {
	Mode       int
	Secret     string
	Listen     string
	Reconnect  uint64
	Reverse    bool
	Connect    string
	ReuseHost  string
	ReusePort  string
	RhostReuse bool
	Proxy      string
	ProxyU     string
	ProxyP     string
}

var Args *Options

func init() {
	Args = new(Options)
	flag.StringVar(&Args.Secret, "s", "", "")
	flag.StringVar(&Args.Listen, "l", "", "")
	flag.Uint64Var(&Args.Reconnect, "reconnect", 0, "")
	flag.BoolVar(&Args.Reverse, "r", false, "")
	flag.StringVar(&Args.Connect, "c", "", "")
	flag.StringVar(&Args.ReuseHost, "rehost", "", "")
	flag.StringVar(&Args.ReusePort, "report", "", "")
	flag.BoolVar(&Args.RhostReuse, "rhostreuse", false, "")
	flag.StringVar(&Args.Proxy, "proxy", "", "")
	flag.StringVar(&Args.ProxyU, "proxyu", "", "")
	flag.StringVar(&Args.ProxyP, "proxyp", "", "")

	flag.Usage = func() {}
}

// ParseOptions Parsing user's options
func ParseOptions() *Options {

	flag.Parse()

	if Args.Listen != "" && Args.Reverse && Args.Connect == "" && Args.Reconnect == 0 && Args.ReuseHost == "" && Args.ReusePort == "" && !Args.RhostReuse && Args.Proxy == "" && Args.ProxyU == "" && Args.ProxyP == "" { // ./stowaway_agent -l <port> -r -s [secret]
		Args.Mode = NORMAL_PASSIVE
		log.Printf("[*]Starting agent node passively.Now listening on port %s\n", Args.Listen)
	} else if Args.Listen == "" && !Args.Reverse && Args.Connect != "" && Args.Reconnect == 0 && Args.ReuseHost == "" && Args.ReusePort == "" && !Args.RhostReuse && Args.Proxy == "" && Args.ProxyU == "" && Args.ProxyP == "" { // ./stowaway_agent -c <ip:port> -s [secret]
		Args.Mode = NORMAL_ACTIVE
		log.Printf("[*]Starting agent node actively.Connecting to %s\n", Args.Connect)
	} else if Args.Listen == "" && !Args.Reverse && Args.Connect != "" && Args.Reconnect != 0 && Args.ReuseHost == "" && Args.ReusePort == "" && !Args.RhostReuse && Args.Proxy == "" && Args.ProxyU == "" && Args.ProxyP == "" { // ./stowaway_agent -c <ip:port> -s [secret] --reconnect <seconds>
		Args.Mode = NORMAL_RECONNECT_ACTIVE
		log.Printf("[*]Starting agent node actively.Connecting to %s.Reconnecting every %d seconds\n", Args.Connect, Args.Reconnect)
	} else if Args.Listen == "" && !Args.Reverse && Args.Connect == "" && Args.Reconnect == 0 && Args.ReuseHost != "" && Args.ReusePort != "" && !Args.RhostReuse && Args.Proxy == "" && Args.ProxyU == "" && Args.ProxyP == "" { // ./stowaway_agent --rehost <ip> --report <port> -s [secret]
		Args.Mode = SO_REUSE_PASSIVE
		log.Printf("[*]Starting agent node passively.Now reusing host %s, port %s(SO_REUSEPORT,SO_REUSEADDR)\n", Args.ReuseHost, Args.ReusePort)
	} else if Args.Listen != "" && !Args.Reverse && Args.Connect == "" && Args.Reconnect == 0 && Args.ReuseHost == "" && Args.ReusePort != "" && !Args.RhostReuse && Args.Proxy == "" && Args.ProxyU == "" && Args.ProxyP == "" { // ./stowaway_agent -l <port> --report <port> -s [secret]
		Args.Mode = IPTABLES_REUSE_PASSIVE
		log.Printf("[*]Starting agent node passively.Now reusing port %s(IPTABLES)\n", Args.ReusePort)
	} else if Args.Listen == "" && !Args.Reverse && Args.Connect != "" && Args.Reconnect == 0 && Args.ReuseHost == "" && Args.ReusePort == "" && Args.RhostReuse && Args.Proxy == "" && Args.ProxyU == "" && Args.ProxyP == "" { // ./stowaway_agent -c <ip:port> -s [secret] --rhostreuse
		Args.Mode = REUSE_ACTIVE
		log.Printf("[*]Starting agent node actively.Connecting to %s.Declaring the remote node is port-reusing\n", Args.Connect)
	} else if Args.Listen == "" && !Args.Reverse && Args.Connect != "" && Args.Reconnect == 0 && Args.ReuseHost == "" && Args.ReusePort == "" && !Args.RhostReuse && Args.Proxy != "" { // ./stowaway_agent -c <ip:port> -s [secret] --proxy <ip:port> --proxyu [username] --proxyp [password]
		Args.Mode = PROXY_ACTIVE
		log.Printf("[*]Starting agent node actively.Connecting to %s via proxy %s\n", Args.Connect, Args.Proxy)
	} else if Args.Listen == "" && !Args.Reverse && Args.Connect != "" && Args.Reconnect == 0 && Args.ReuseHost == "" && Args.ReusePort == "" && Args.RhostReuse && Args.Proxy != "" { // ./stowaway_agent -c <ip:port> -s [secret] --proxy <ip:port> --proxyu [username] --proxyp [password] --rhostreuse
		Args.Mode = PROXY_REUSE_ACTIVE
		log.Printf("[*]Starting agent node actively.Connecting to %s via proxy %s.Declaring the remote node is port-reusing\n", Args.Connect, Args.Proxy)
	} else if Args.Listen == "" && !Args.Reverse && Args.Connect != "" && Args.Reconnect != 0 && Args.ReuseHost == "" && Args.ReusePort == "" && !Args.RhostReuse && Args.Proxy != "" { // ./stowaway_agent -c <ip:port> -s [secret] --proxy <ip:port> --proxyu [username] --proxyp [password] --reconnect <seconds>
		Args.Mode = PROXY_RECONNECT_ACTIVE
		log.Printf("[*]Starting agent node actively.Connecting to %s via proxy %s.Reconnecting every %d seconds\n", Args.Connect, Args.Proxy, Args.Reconnect)
	} else {
		log.Println("[*]Bad options!")
		os.Exit(1)
	}

	if err := checkOptions(Args); err != nil {
		log.Fatalf("[*]Options err: %s\n", err.Error())
	}

	return Args
}

func checkOptions(option *Options) error {
	var err error

	if Args.Connect != "" {
		_, err = net.ResolveTCPAddr("", option.Connect)
	}

	if Args.Proxy != "" {
		_, err = net.ResolveTCPAddr("", option.Proxy)
	}

	if Args.ReuseHost != "" {
		if addr := net.ParseIP(Args.ReuseHost); addr == nil {
			err = errors.New("ReuseHost is not a valid IP addr")
		}
	}

	return err
}
