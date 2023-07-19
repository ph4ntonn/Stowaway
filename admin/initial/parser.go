// +build !windows

package initial

import (
	"flag"
	"fmt"
	"net"
	"os"

	"Stowaway/admin/printer"

	"github.com/nsf/termbox-go"
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
}

var Args *Options

func init() {
	Args = new(Options)

	flag.StringVar(&Args.Secret, "s", "", "Communication secret")
	flag.StringVar(&Args.Listen, "l", "", "Listen port")
	flag.StringVar(&Args.Connect, "c", "", "The node address when you actively connect to it")
	flag.StringVar(&Args.Socks5Proxy, "socks5-proxy", "", "The socks5 server ip:port you want to use")
	flag.StringVar(&Args.Socks5ProxyU, "socks5-proxyu", "", "socks5 username")
	flag.StringVar(&Args.Socks5ProxyP, "socks5-proxyp", "", "socks5 password")
	flag.StringVar(&Args.HttpProxy, "http-proxy", "", "The http proxy server ip:port you want to use")
	flag.StringVar(&Args.Downstream, "down", "raw", "")

	flag.Usage = newUsage
}

func newUsage() {
	termbox.Close()

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

	if Args.Listen != "" && Args.Connect == "" && Args.Socks5Proxy == "" && Args.HttpProxy == "" { // ./stowaway_admin -l <port> -s [secret]
		Args.Mode = NORMAL_PASSIVE
		printer.Warning("[*] Starting admin node on port %s\r\n", Args.Listen)
	} else if Args.Connect != "" && Args.Listen == "" && Args.Socks5Proxy == "" && Args.HttpProxy == "" { // ./stowaway_admin -c <ip:port> -s [secret]
		Args.Mode = NORMAL_ACTIVE
		printer.Warning("[*] Trying to connect node actively")
	} else if Args.Connect != "" && Args.Listen == "" && Args.Socks5Proxy != "" && Args.HttpProxy == "" { // ./stowaway_admin -c <ip:port> -s [secret] --proxy <ip:port> --proxyu [username] --proxyp [password]
		Args.Mode = SOCKS5_PROXY_ACTIVE
		printer.Warning("[*] Trying to connect node actively via socks5 proxy %s\r\n", Args.Socks5Proxy)
	} else if Args.Connect != "" && Args.Listen == "" && Args.Socks5Proxy == "" && Args.HttpProxy != "" {
		Args.Mode = HTTP_PROXY_ACTIVE
		printer.Warning("[*] Trying to connect node actively via http proxy %s\r\n", Args.HttpProxy)
	} else { // Wrong format
		flag.Usage()
		os.Exit(0)
	}

	if err := checkOptions(Args); err != nil {
		termbox.Close()
		printer.Fail("[*] Options err: %s\r\n", err.Error())
		os.Exit(0)
	}

	return Args
}

func checkOptions(option *Options) error {
	var err error

	if Args.Connect != "" {
		_, err = net.ResolveTCPAddr("", option.Connect)
	}

	if Args.Socks5Proxy != "" {
		_, err = net.ResolveTCPAddr("", option.Socks5Proxy)
	}

	if Args.HttpProxy != "" {
		_, err = net.ResolveTCPAddr("", option.HttpProxy)
	}

	return err
}
