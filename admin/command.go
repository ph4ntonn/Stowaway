package admin

import (
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

// Command  cli settings
var Command = &cli.Command{
	Name: "admin",
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
	},
	Action: func(c *cli.Context) error {
		if c.Bool("debug") {
			log.SetLevel(log.DebugLevel)
		}
		if c.String("control") == "" && c.String("listen") != "" {
			log.Infof("Starting admin node on port %s without cc port\n", c.String("listen"))
		} else if c.String("control") != "" && c.String("listen") != "" {
			log.Infof("Starting admin node on port %s and cc port is %s\n", c.String("listen"), c.String("control"))
		} else {
			log.Error("Please at least set the -l/--listen option")
			os.Exit(1)
		}
		NewAdmin(c)
		return nil
	},
}
