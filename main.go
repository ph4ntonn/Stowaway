package main

import (
	"Stowaway/admin"
	"Stowaway/agent"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

const version = "1.1"

func init() {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "01/02 15:04:05",
	})
}

func main() {
	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Println("helloworld")
	}
	app := &cli.App{}
	app.Name = "Stowaway"
	app.Commands = []*cli.Command{
		agent.Command,
		admin.Command,
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
