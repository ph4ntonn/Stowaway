package admin

import (
	"Stowaway/common"
	"flag"
	"log"
	"os"
)

var Args *common.AdminOptions

func init() {
	Args = new(common.AdminOptions)
	flag.StringVar(&Args.Secret, "s", "", "Remote `ip` address.")
	flag.StringVar(&Args.Listen, "l", "", "Remote `ip` address.")
	flag.StringVar(&Args.Connect, "c", "", "Remote `ip` address.")
	flag.BoolVar(&Args.Rhostreuse, "rhostreuse", false, "")
}
func ParseCommand() {

	flag.Parse()

	if Args.Listen != "" && Args.Connect == "" {
		log.Printf("[*]Starting admin node on port %s\n", Args.Listen)
	} else if Args.Connect != "" && Args.Listen != "" {
		log.Println("[*]If you are using active connect mode, do not set -l option")
		os.Exit(0)
	} else if Args.Connect != "" && Args.Listen == "" {
		log.Println("[*]Trying to connect startnode actively...")
	} else {
		log.Println("Bad format! See readme!")
		os.Exit(0)
	}
	NewAdmin(Args)
}
