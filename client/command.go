package client

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

// Command  cli settings
var Command = &cli.Command{
	Name: "connect",
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
			Name:    "protocol",
			Aliases: []string{"p"},
			Value:   "tcp",
			Usage:   "comm protocol",
		},
		&cli.StringSliceFlag{
			Name:    "tunnel",
			Aliases: []string{"t"},
			Usage:   "create tunnel",
		},
	},
	Action: func(c *cli.Context) error {
		if c.Bool("debug") {
			log.SetLevel(log.DebugLevel)
		}
		newClient(c)
		if c.String("secret") != "" {
			fmt.Printf("Start connection with secret : %s\n", c.String("secret"))
		}
		return nil
	},
}
