package admin

import (
	"flag"
	"log"

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

// ParseCommand 解析命令行
func ParseCommand() {

	flag.Parse()

	if Args.Listen != "" && Args.Connect == "" {
		log.Printf("[*]Starting admin node on port %s\n", Args.Listen)
	} else if Args.Connect != "" && Args.Listen != "" {
		log.Fatalln("[*]If you are using active connect mode, do not set -l option")
	} else if Args.Connect != "" && Args.Listen == "" {
		log.Println("[*]Trying to connect startnode actively...")
	} else {
		log.Fatalln("Bad format! See readme!")
	}
	NewAdmin(Args)
}
