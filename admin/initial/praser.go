package initial

import (
	"flag"
	"fmt"
	"net"
	"os"

	"Stowaway/admin/printer"
)

const (
	NORMAL_ACTIVE = iota
	PROXY_ACTIVE
	NORMAL_PASSIVE
)

type Options struct {
	Mode       uint8
	Secret     string
	Listen     string
	Connect    string
	Proxy      string
	ProxyU     string
	ProxyP     string
	Downstream string
}

var Args *Options

func init() {
	Args = new(Options)

	flag.StringVar(&Args.Secret, "s", "", "Communication secret")
	flag.StringVar(&Args.Listen, "l", "", "Listen port")
	flag.StringVar(&Args.Connect, "c", "", "The node address when you actively connect to it")
	flag.StringVar(&Args.Proxy, "proxy", "", "The socks5 server ip:port you want to use")
	flag.StringVar(&Args.ProxyU, "proxyu", "", "socks5 username")
	flag.StringVar(&Args.ProxyP, "proxyp", "", "socks5 password")
	flag.StringVar(&Args.Downstream, "down", "raw", "")

	flag.Usage = newUsage
}

func newUsage() {
	fmt.Fprintf(os.Stderr, `
Usages:
	>> ./stowaway_admin -l <port> -s [secret]
	>> ./stowaway_admin -c <ip:port> -s [secret] 
	>> ./stowaway_admin -c <ip:port> -s [secret] --proxy <ip:port> --proxyu [username] --proxyp [password]
	>> ./stowaway_admin -c <ip:port> -s [secret] --rhostreuse
	>> ./stowaway_admin -c <ip:port> -s [secret] --proxy <ip:port> --proxyu [username] --proxyp [password] --rhostreuse

Options:
`)
	flag.PrintDefaults()
}

// ParseOptions Parsing user's options
func ParseOptions() *Options {
	flag.Parse()

	if Args.Listen != "" && Args.Connect == "" && Args.Proxy == "" { // ./stowaway_admin -l <port> -s [secret]
		Args.Mode = NORMAL_PASSIVE
		printer.Warning("[*] Starting admin node on port %s\n", Args.Listen)
	} else if Args.Connect != "" && Args.Listen == "" && Args.Proxy == "" { // ./stowaway_admin -c <ip:port> -s [secret]
		Args.Mode = NORMAL_ACTIVE
		printer.Warning("[*] Trying to connect node actively")
	} else if Args.Connect != "" && Args.Listen == "" && Args.Proxy != "" { // ./stowaway_admin -c <ip:port> -s [secret] --proxy <ip:port> --proxyu [username] --proxyp [password]
		Args.Mode = PROXY_ACTIVE
		printer.Warning("[*] Trying to connect node actively with proxy %s\n", Args.Proxy)
	} else { // Wrong format
		flag.Usage()
		os.Exit(1)
	}

	if err := checkOptions(Args); err != nil {
		printer.Fail("[*] Options err: %s\n", err.Error())
		os.Exit(0)
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

	return err
}
