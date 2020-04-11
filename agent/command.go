package agent

import (
	"log"
	"os"

	"github.com/urfave/cli/v2"
)

// Command  cli settings
var Flags = []cli.Flag{
	&cli.StringFlag{
		Name:    "secret",
		Aliases: []string{"s"},
		Usage:   "secret key",
	},
	&cli.StringFlag{
		Name:    "listen",
		Aliases: []string{"l"},
		Usage:   "listen port",
	},
	&cli.BoolFlag{
		Name:    "startnode",
		Aliases: []string{"Startnode"},
		Usage:   "act as startnode",
	},
	&cli.BoolFlag{
		Name:    "reverse",
		Aliases: []string{"r"},
		Usage:   "connect to others actively",
	},
	&cli.StringFlag{
		Name:    "reconnect",
		Aliases: []string{"Reconnect"},
		Usage:   "reconnect to admin node",
	},
	&cli.StringFlag{
		Name:    "monitor",
		Aliases: []string{"m"},
		Usage:   "monitor node",
	},
	&cli.StringFlag{
		Name:  "rehost",
		Usage: "specify reuse host",
	},
	&cli.StringFlag{
		Name:  "report",
		Usage: "specify reuse port",
	},
	&cli.BoolFlag{
		Name:  "rhostreuse",
		Usage: "remote host reusing port",
	},
}

func Action(c *cli.Context) error {
	if c.String("listen") != "" && c.Bool("reverse") && c.String("monitor") == "" {
		log.Printf("Starting agent node on port %s passively\n", c.String("listen"))
	} else if c.String("listen") != "" && c.Bool("reverse") && c.String("monitor") != "" {
		log.Println("If you want to start node passively,do not set the -m option")
		os.Exit(0)
	} else if c.String("listen") != "" && !c.Bool("reverse") && c.String("monitor") == "" {
		log.Println("You should set the -m option!")
		os.Exit(0)
	} else if !c.Bool("reverse") && c.String("monitor") != "" {
		log.Println("Node starting......")
	} else if c.String("reconnect") != "" && !c.Bool("startnode") {
		log.Println("Do not set the --reconnect option on simple node")
		os.Exit(0)
	} else if (c.String("rehost") == "" && c.String("report") != "") || (c.String("rehost") != "" && c.String("report") == "") {
		log.Println("Need rehost/report option")
		os.Exit(0)
	} else if (c.String("report") != "" || c.String("rehost") != "") && (c.String("listen") != "" || c.String("monitor") != "") {
		log.Println("Choose one from (--report,--rehost) and -l and -m")
		os.Exit(0)
	} else if (c.String("report") != "" && c.String("rehost") != "") && (c.String("listen") == "" && c.String("monitor") == "") {
		log.Printf("Starting agent node reusing port %s \n", c.String("report"))
	} else {
		log.Println("Bad format! See readme!")
		os.Exit(0)
	}
	NewAgent(c)
	return nil
}
