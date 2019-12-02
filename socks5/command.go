package socks5

import (
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

// Command  cli settings
var Command = &cli.Command{
	Name: "socks5",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "debug",
			Aliases: []string{"d"},
			Usage:   "debug level",
		},
		&cli.StringFlag{
			Name:    "username",
			Aliases: []string{"u"},
			Usage:   "username",
		},
		&cli.StringFlag{
			Name:    "secret",
			Aliases: []string{"s"},
			Usage:   "secret",
		},
		&cli.StringFlag{
			Name:  "protocol",
			Value: "tcp",
			Usage: "comm protocol",
		},
		&cli.StringFlag{
			Name:    "port",
			Aliases: []string{"p"},
			Usage:   "listening port",
		},
	},
	Action: func(c *cli.Context) error {
		if c.Bool("debug") {
			log.SetLevel(log.DebugLevel)
		}
		newSocks5(c)
		return nil
	},
}
