package server

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

// Command  cli settings
var Command = &cli.Command{
	Name: "listen",
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
			Name:  "protocol",
			Value: "tcp",
			Usage: "comm protocol",
		},
		&cli.BoolFlag{
			Name:    "heartbeat",
			Aliases: []string{"Heartbeat"},
			Usage:   "turn on heartbeat function",
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
		newServer(c)
		if c.String("secret") != "" {
			fmt.Printf("Start listening with secret %s on port %s\n", c.String("secret"), c.String("port"))
		}
		return nil
	},
}
