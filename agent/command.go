package agent

import (
	"log"
	"os"

	"github.com/urfave/cli/v2"
)

// Command  cli settings
var Command = &cli.Command{
	Name: "agent",
	Flags: []cli.Flag{
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
		if c.String("listen") != "" && c.Bool("reverse") && c.String("monitor") == "" {
			log.Printf("Starting agent node on port %s passively\n", c.String("listen"))
		} else if c.String("listen") != "" && c.Bool("reverse") && c.String("monitor") != "" {
			log.Println("If you want to start node passively,do not set the -m option")
			os.Exit(1)
		} else if c.String("listen") != "" && !c.Bool("reverse") && c.String("monitor") == "" {
			log.Println("You should set the -m option!")
			os.Exit(1)
		} else if !c.Bool("reverse") && c.String("monitor") != "" {
			log.Println("Startnode starting......")
		} else if c.String("reconnect") != "" && !c.Bool("startnode") {
			log.Println("Do not set the --reconnect option on simple node")
			os.Exit(1)
		} else {
			log.Println("Bad format! See readme!")
			os.Exit(1)
		}
		NewAgent(c)
		return nil
	},
}
