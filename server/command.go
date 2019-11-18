package server

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

// Command  cli settings
var Command = &cli.Command{
	Name: "listen",
	// Flags: []cli.Flag{f},
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
		&cli.StringFlag{
			Name:    "port",
			Aliases: []string{"p"},
			Value:   "tcp",
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
