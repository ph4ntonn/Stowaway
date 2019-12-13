package client

import (
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
		&cli.BoolFlag{
			Name:    "heartbeat",
			Aliases: []string{"Heartbeat"},
			Usage:   "turn on heartbeat function",
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
		if c.String("secret") != "" {
			log.Infof("Start connection with secret : %s\n", c.String("secret"))
		} else {
			log.Infof("Connection started!")
		}
		newClient(c)
		return nil
	},
}
