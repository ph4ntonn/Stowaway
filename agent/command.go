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
			Name:    "monitor",
			Aliases: []string{"m"},
			Usage:   "monitor node",
		},
	},
	Action: func(c *cli.Context) error {
		if c.Bool("debug") {
			log.SetLevel(log.DebugLevel)
		}
		if c.String("control") == "" && c.String("listen") != "" {
			log.Infof("Starting agent node on port %s without cc port\n", c.String("listen"))
		} else if c.String("control") != "" && c.String("listen") != "" {
			log.Infof("Starting agent node on port %s and cc port is %s\n", c.String("listen"), c.String("control"))
		} else if (c.String("monitor") == "" || c.String("listen") == "") && c.Bool("reverse") == false {
			log.Error("Please at least set the -m/--monitor  and -l/--listen option")
			os.Exit(1)
		}
		NewAgent(c)
		return nil
	},
}
