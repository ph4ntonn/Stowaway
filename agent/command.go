package agent

import (
	"flag"
	"log"
	"os"

	"Stowaway/utils"
)

var Args *utils.AgentOptions

func init() {
	Args = new(utils.AgentOptions)
	flag.StringVar(&Args.Secret, "s", "", "")
	flag.StringVar(&Args.Listen, "l", "", "")
	flag.StringVar(&Args.Reconnect, "reconnect", "", "")
	flag.BoolVar(&Args.Reverse, "r", false, "")
	flag.StringVar(&Args.Monitor, "m", "", "")
	flag.BoolVar(&Args.IsStartNode, "startnode", false, "")
	flag.StringVar(&Args.ReuseHost, "rehost", "", "")
	flag.StringVar(&Args.ReusePort, "report", "", "")
	flag.BoolVar(&Args.RhostReuse, "rhostreuse", false, "")

	flag.Usage = func() {}
}

func ParseCommand() {
	flag.Parse()

	if Args.Listen != "" && Args.Reverse && Args.Monitor == "" {
		log.Printf("Starting agent node on port %s passively\n", Args.Listen)
	} else if Args.Listen != "" && Args.Reverse && Args.Monitor != "" {
		log.Println("If you want to start node passively,do not set the -m option")
		os.Exit(0)
	} else if Args.Listen != "" && !Args.Reverse && Args.Monitor == "" && Args.ReusePort == "" {
		log.Println("You should set the -m option!")
		os.Exit(0)
	} else if !Args.Reverse && Args.Monitor != "" {
		log.Println("Node starting......")
	} else if Args.Reconnect != "" && !Args.IsStartNode {
		log.Println("Do not set the --reconnect option on simple node")
		os.Exit(0)
	} else if (Args.ReusePort != "" || Args.ReuseHost != "") && (Args.Monitor != "") {
		log.Println("Choose one from (--report,--rehost) and -m")
		os.Exit(0)
	} else if Args.ReusePort != "" && Args.ReuseHost == "" && Args.Listen == "" {
		log.Println("If you want to reuse port through iptable method,you should set --report and -l simultaneously,or if you want to use SO_REUSE method,you should set --report and --rehost instead")
		os.Exit(0)
	} else if Args.ReusePort != "" && Args.ReuseHost != "" && Args.Listen != "" {
		log.Println("Should be report+rehost or report+listen")
		os.Exit(0)
	} else if (Args.ReusePort != "" && Args.ReuseHost != "") && (Args.Listen == "" && Args.Monitor == "") {
		log.Printf("Starting agent node reusing port %s \n", Args.ReusePort)
	} else if Args.ReusePort != "" && Args.Listen != "" && Args.ReuseHost == "" && Args.Monitor == "" {
		log.Printf("Now agent node is listening on port %s and reusing port %s", Args.Listen, Args.ReusePort)
	} else {
		log.Println("Bad format! See readme!")
		os.Exit(0)
	}
	NewAgent(Args)
}
