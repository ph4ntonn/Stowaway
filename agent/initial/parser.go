package initial

import (
	"errors"
	"flag"
	"log"
	"net"
	"os"
	"strings"
)

const (
	NORMAL_ACTIVE = iota
	NORMAL_RECONNECT_ACTIVE
	NORMAL_PASSIVE
	SOCKS5_PROXY_ACTIVE
	HTTP_PROXY_ACTIVE
	SOCKS5_PROXY_RECONNECT_ACTIVE
	HTTP_PROXY_RECONNECT_ACTIVE
	SO_REUSE_PASSIVE
	IPTABLES_REUSE_PASSIVE
)

type Options struct {
	Mode         int
	Secret       string
	Listen       string
	Reconnect    uint64
	Connect      string
	ReuseHost    string
	ReusePort    string
	Socks5Proxy  string
	Socks5ProxyU string
	Socks5ProxyP string
	HttpProxy    string
	Upstream     string
	Downstream   string
	Charset      string
	Domain       string
	TlsEnable    bool
}

var args *Options

func init() {
	args = new(Options)

	flag.StringVar(&args.Secret, "s", "", "")
	flag.StringVar(&args.Listen, "l", "", "")
	flag.Uint64Var(&args.Reconnect, "reconnect", 0, "")
	flag.StringVar(&args.Connect, "c", "", "")
	flag.StringVar(&args.ReuseHost, "rehost", "", "")
	flag.StringVar(&args.ReusePort, "report", "", "")
	flag.StringVar(&args.Socks5Proxy, "socks5-proxy", "", "")
	flag.StringVar(&args.Socks5ProxyU, "socks5-proxyu", "", "")
	flag.StringVar(&args.Socks5ProxyP, "socks5-proxyp", "", "")
	flag.StringVar(&args.HttpProxy, "http-proxy", "", "")
	flag.StringVar(&args.Upstream, "up", "raw", "")
	flag.StringVar(&args.Downstream, "down", "raw", "")
	flag.StringVar(&args.Charset, "cs", "utf-8", "")
	flag.StringVar(&args.Domain, "domain", "", "")
	flag.BoolVar(&args.TlsEnable, "tls-enable", false, "")

	flag.Usage = func() {}
}

// ParseOptions Parsing user's options
func ParseOptions() *Options {

	flag.Parse()

	if args.Listen != "" && args.Connect == "" && args.Reconnect == 0 && args.ReuseHost == "" && args.ReusePort == "" && args.Socks5Proxy == "" && args.Socks5ProxyU == "" && args.Socks5ProxyP == "" && args.HttpProxy == "" { // ./stowaway_agent -l <port> -s [secret]
		args.Mode = NORMAL_PASSIVE
		log.Printf("[*] Starting agent node passively.Now listening on port %s\n", args.Listen)
	} else if args.Listen == "" && args.Connect != "" && args.Reconnect == 0 && args.ReuseHost == "" && args.ReusePort == "" && args.Socks5Proxy == "" && args.Socks5ProxyU == "" && args.Socks5ProxyP == "" && args.HttpProxy == "" { // ./stowaway_agent -c <ip:port> -s [secret]
		args.Mode = NORMAL_ACTIVE
		log.Printf("[*] Starting agent node actively.Connecting to %s\n", args.Connect)
	} else if args.Listen == "" && args.Connect != "" && args.Reconnect != 0 && args.ReuseHost == "" && args.ReusePort == "" && args.Socks5Proxy == "" && args.Socks5ProxyU == "" && args.Socks5ProxyP == "" && args.HttpProxy == "" { // ./stowaway_agent -c <ip:port> -s [secret] --reconnect <seconds>
		args.Mode = NORMAL_RECONNECT_ACTIVE
		log.Printf("[*] Starting agent node actively.Connecting to %s.Reconnecting every %d seconds\n", args.Connect, args.Reconnect)
	} else if args.Listen == "" && args.Connect == "" && args.Reconnect == 0 && args.ReuseHost != "" && args.ReusePort != "" && args.Socks5Proxy == "" && args.Socks5ProxyU == "" && args.Socks5ProxyP == "" && args.HttpProxy == "" { // ./stowaway_agent --rehost <ip> --report <port> -s [secret]
		args.Mode = SO_REUSE_PASSIVE
		log.Printf("[*] Starting agent node passively.Now reusing host %s, port %s(SO_REUSEPORT,SO_REUSEADDR)\n", args.ReuseHost, args.ReusePort)
	} else if args.Listen != "" && args.Connect == "" && args.Reconnect == 0 && args.ReuseHost == "" && args.ReusePort != "" && args.Socks5Proxy == "" && args.Socks5ProxyU == "" && args.Socks5ProxyP == "" && args.HttpProxy == "" { // ./stowaway_agent -l <port> --report <port> -s [secret]
		args.Mode = IPTABLES_REUSE_PASSIVE
		log.Printf("[*] Starting agent node passively.Now reusing port %s(IPTABLES)\n", args.ReusePort)
	} else if args.Listen == "" && args.Connect != "" && args.Reconnect == 0 && args.ReuseHost == "" && args.ReusePort == "" && args.Socks5Proxy != "" && args.HttpProxy == "" { // ./stowaway_agent -c <ip:port> -s [secret] --proxy <ip:port> --proxyu [username] --proxyp [password]
		args.Mode = SOCKS5_PROXY_ACTIVE
		log.Printf("[*] Starting agent node actively.Connecting to %s via socks5 proxy %s\n", args.Connect, args.Socks5Proxy)
	} else if args.Listen == "" && args.Connect != "" && args.Reconnect != 0 && args.ReuseHost == "" && args.ReusePort == "" && args.Socks5Proxy != "" && args.HttpProxy == "" { // ./stowaway_agent -c <ip:port> -s [secret] --proxy <ip:port> --proxyu [username] --proxyp [password] --reconnect <seconds>
		args.Mode = SOCKS5_PROXY_RECONNECT_ACTIVE
		log.Printf("[*] Starting agent node actively.Connecting to %s via socks5 proxy %s.Reconnecting every %d seconds\n", args.Connect, args.Socks5Proxy, args.Reconnect)
	} else if args.Listen == "" && args.Connect != "" && args.Reconnect == 0 && args.ReuseHost == "" && args.ReusePort == "" && args.Socks5Proxy == "" && args.HttpProxy != "" {
		args.Mode = HTTP_PROXY_ACTIVE
		log.Printf("[*] Starting agent node actively.Connecting to %s via http proxy %s\n", args.Connect, args.HttpProxy)
	} else if args.Listen == "" && args.Connect != "" && args.Reconnect != 0 && args.ReuseHost == "" && args.ReusePort == "" && args.Socks5Proxy == "" && args.HttpProxy != "" {
		args.Mode = HTTP_PROXY_RECONNECT_ACTIVE
		log.Printf("[*] Starting agent node actively.Connecting to %s via http proxy %s.Reconnecting every %d seconds\n", args.Connect, args.HttpProxy, args.Reconnect)
	} else {
		os.Exit(1)
	}

	if args.Charset != "utf-8" && args.Charset != "gbk" {
		log.Fatalf("[*] Charset must be set as 'utf-8'(default) or 'gbk'")
	}

	if args.Domain == "" && args.Connect != "" {
		addrSlice := strings.SplitN(args.Connect, ":", 2)
		args.Domain = addrSlice[0]
	}

	if err := checkOptions(args); err != nil {
		log.Fatalf("[*] Options err: %s\n", err.Error())
	}

	return args
}

func checkOptions(option *Options) error {
	var err error

	if args.Connect != "" {
		_, err = net.ResolveTCPAddr("", option.Connect)
	}

	if args.Socks5Proxy != "" {
		_, err = net.ResolveTCPAddr("", option.Socks5Proxy)
	}

	if args.HttpProxy != "" {
		_, err = net.ResolveTCPAddr("", option.HttpProxy)
	}

	if args.ReuseHost != "" {
		if addr := net.ParseIP(args.ReuseHost); addr == nil {
			err = errors.New("ReuseHost is not a valid IP addr")
		}
	}

	return err
}
