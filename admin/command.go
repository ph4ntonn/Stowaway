package admin

import (
	"flag"
	"log"
	"os"

	"Stowaway/utils"
)

var Args *utils.AdminOptions

func init() {
	Args = new(utils.AdminOptions)
	flag.StringVar(&Args.Secret, "s", "", "Communication secret")
	flag.StringVar(&Args.Listen, "l", "", "Listen port")
	flag.StringVar(&Args.Connect, "c", "", "The startnode address when you actively connect to it")
	flag.BoolVar(&Args.Rhostreuse, "rhostreuse", false, "If the startnode is reusing port")
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
