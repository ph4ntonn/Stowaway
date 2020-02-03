package agent

import (
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

// Command  cli settings
var Command = &cli.Command{
	Name: "agent",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "debug",
			Aliases: []string{"d"},
			Usage:   "debug level",
		},
		&cli.StringFlag{
			Name:    "secret",
			Aliases: []string{"s"},
			Usage:   "secret key",
		},
		&cli.StringFlag{
			Name:    "control",
			Aliases: []string{"cc"},
			Usage:   "set cc port",
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
		&cli.BoolFlag{
			Name:    "single",
			Aliases: []string{"Single"},
			Usage:   "If only startnode",
		},
	},
	Action: func(c *cli.Context) error {
		if c.Bool("debug") {
			log.SetLevel(log.DebugLevel)
		}
		if c.String("listen") != "" && c.Bool("reverse") && c.String("monitor") == "" {
			log.Infof("Starting agent node on port %s passively\n", c.String("listen"))
		} else if c.String("listen") != "" && c.Bool("reverse") && c.String("monitor") != "" {
			log.Error("If you want to start node passively,do not set the -m option")
			os.Exit(1)
		} else if c.String("listen") != "" && !c.Bool("reverse") && c.String("monitor") == "" {
			log.Error("You should set the -m option!")
			os.Exit(1)
		} else if c.String("listen") == "" && !c.Bool("reverse") && c.String("monitor") != "" {
			log.Error("You should set the -l option!")
			os.Exit(1)
		} else if c.String("listen") != "" && !c.Bool("reverse") && c.String("monitor") != "" {
			log.Infof("Starting agent node on port %s \n", c.String("listen"))
		} else if c.String("reconnect") != "" && !c.Bool("startnode") {
			log.Error("Do not set the --reconnect option on simple node")
			os.Exit(1)
		} else {
			log.Error("Bad format! See readme!")
			os.Exit(1)
		}
		NewAgent(c)
		return nil
	},
}
