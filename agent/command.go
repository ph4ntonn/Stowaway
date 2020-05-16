package agent

import (
	"flag"
	"log"

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

// ParseCommand 解析输入的命令
func ParseCommand() {
	flag.Parse()

	if Args.Listen != "" && Args.Reverse && Args.Monitor == "" {
		log.Printf("Starting agent node on port %s passively\n", Args.Listen)
	} else if Args.Listen != "" && Args.Reverse && Args.Monitor != "" {
		log.Fatalln("If you want to start node passively,do not set the -m option")
	} else if Args.Listen != "" && !Args.Reverse && Args.Monitor == "" && Args.ReusePort == "" {
		log.Fatalln("You should set the -m option!")
	} else if !Args.Reverse && Args.Monitor != "" {
		log.Println("Node starting......")
	} else if Args.Reconnect != "" && !Args.IsStartNode {
		log.Fatalln("Do not set the --reconnect option on simple node")
	} else if (Args.ReusePort != "" || Args.ReuseHost != "") && (Args.Monitor != "") {
		log.Fatalln("Choose one from (--report,--rehost) and -m")
	} else if Args.ReusePort != "" && Args.ReuseHost == "" && Args.Listen == "" {
		log.Fatalln("If you want to reuse port through iptable method,you should set --report and -l simultaneously,or if you want to use SO_REUSE method,you should set --report and --rehost instead")
	} else if Args.ReusePort != "" && Args.ReuseHost != "" && Args.Listen != "" {
		log.Fatalln("Should be report+rehost or report+listen")
	} else if (Args.ReusePort != "" && Args.ReuseHost != "") && (Args.Listen == "" && Args.Monitor == "") {
		log.Printf("Starting agent node reusing port %s \n", Args.ReusePort)
	} else if Args.ReusePort != "" && Args.Listen != "" && Args.ReuseHost == "" && Args.Monitor == "" {
		log.Printf("Now agent node is listening on port %s and reusing port %s", Args.Listen, Args.ReusePort)
	} else {
		log.Fatalln("Bad format! See readme!")
	}
	NewAgent(Args)
}
