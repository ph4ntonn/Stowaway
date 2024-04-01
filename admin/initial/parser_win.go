//go:build windows

package initial

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strings"

	"Stowaway/admin/printer"
)

const (
	NORMAL_ACTIVE = iota
	NORMAL_PASSIVE
	SOCKS5_PROXY_ACTIVE
	HTTP_PROXY_ACTIVE
)

type Options struct {
	Mode         uint8
	Secret       string
	Listen       string
	Connect      string
	Socks5Proxy  string
	Socks5ProxyU string
	Socks5ProxyP string
	HttpProxy    string
	Downstream   string
	Domain       string
	TlsEnable    bool
	Heartbeat    bool
}

var args *Options

func init() {
	args = new(Options)

	flag.StringVar(&args.Secret, "s", "", "Communication secret")
	flag.StringVar(&args.Listen, "l", "", "Listen port")
	flag.StringVar(&args.Connect, "c", "", "The node address when you actively connect to it")
	flag.StringVar(&args.Socks5Proxy, "socks5-proxy", "", "The socks5 server ip:port you want to use")
	flag.StringVar(&args.Socks5ProxyU, "socks5-proxyu", "", "socks5 username")
	flag.StringVar(&args.Socks5ProxyP, "socks5-proxyp", "", "socks5 password")
	flag.StringVar(&args.HttpProxy, "http-proxy", "", "The http proxy server ip:port you want to use")
	flag.StringVar(&args.Downstream, "down", "raw", "Downstream data type you want to use")
	flag.StringVar(&args.Domain, "domain", "", "Domain name for TLS SNI/WS")
	flag.BoolVar(&args.TlsEnable, "tls-enable", false, "Encrypt connection by TLS")
	flag.BoolVar(&args.Heartbeat, "heartbeat", false, "Send heartbeat packet to first agent")

	flag.Usage = newUsage
}

func newUsage() {
	fmt.Fprintf(os.Stderr, `
Usages:
	>> ./stowaway_admin -l <port> -s [secret]
	>> ./stowaway_admin -c <ip:port> -s [secret] 
	>> ./stowaway_admin -c <ip:port> -s [secret] --socks5-proxy <ip:port> --socks5-proxyu [username] --socks5-proxyp [password]
`)
	flag.PrintDefaults()
}

// ParseOptions Parsing user's options
func ParseOptions() *Options {
	flag.Parse()

	if args.Listen != "" && args.Connect == "" && args.Socks5Proxy == "" && args.HttpProxy == "" { // ./stowaway_admin -l <port> -s [secret]
		args.Mode = NORMAL_PASSIVE
		printer.Warning("[*] Starting admin node on port %s\r\n", args.Listen)
	} else if args.Connect != "" && args.Listen == "" && args.Socks5Proxy == "" && args.HttpProxy == "" { // ./stowaway_admin -c <ip:port> -s [secret]
		args.Mode = NORMAL_ACTIVE
		printer.Warning("[*] Trying to connect node actively")
	} else if args.Connect != "" && args.Listen == "" && args.Socks5Proxy != "" && args.HttpProxy == "" { // ./stowaway_admin -c <ip:port> -s [secret] --proxy <ip:port> --proxyu [username] --proxyp [password]
		args.Mode = SOCKS5_PROXY_ACTIVE
		printer.Warning("[*] Trying to connect node actively via socks5 proxy %s\r\n", args.Socks5Proxy)
	} else if args.Connect != "" && args.Listen == "" && args.Socks5Proxy == "" && args.HttpProxy != "" {
		args.Mode = HTTP_PROXY_ACTIVE
		printer.Warning("[*] Trying to connect node actively via http proxy %s\r\n", args.HttpProxy)
	} else { // Wrong format
		flag.Usage()
		os.Exit(0)
	}

	if args.Domain == "" && args.Connect != "" {
		addrSlice := strings.SplitN(args.Connect, ":", 2)
		args.Domain = addrSlice[0]
	}

	if err := checkOptions(args); err != nil {
		printer.Fail("[*] Options err: %s\r\n", err.Error())
		os.Exit(0)
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

	return err
}
