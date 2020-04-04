package admin

import (
	"log"
	"os"

	"github.com/urfave/cli/v2"
)

// Command  cli settings
var Command = &cli.Command{
	Name: "admin",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "secret",
			Aliases: []string{"s"},
			Usage:   "secret key",
		},
		&cli.StringFlag{
			Name:    "connect",
			Aliases: []string{"c"},
			Usage:   "connect to startnode",
		},
		&cli.StringFlag{
			Name:    "listen",
			Aliases: []string{"l"},
			Usage:   "listen port",
		},
	},
	Action: func(c *cli.Context) error {
		if c.String("listen") != "" && c.String("connect") == "" {
			log.Printf("[*]Starting admin node on port %s\n", c.String("listen"))
		} else if c.String("connect") != "" && c.String("listen") != "" {
			log.Println("[*]If you are using active connect mode, do not set -l option")
			os.Exit(1)
		} else if c.String("connect") != "" && c.String("listen") == "" {
			log.Println("[*]Trying to connect startnode actively...")
		} else {
			log.Println("[*]Please at least set the -l/--listen option")
			os.Exit(1)
		}
		NewAdmin(c)
		return nil
	},
}
