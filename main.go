package main

import (
	"log"
	"os"

	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Version = "0.1.3"
	app.Name = "fastc"
	app.Usage = "configures asterisk using json"
	app.Commands = []cli.Command{
		{
			Name:    "dongles",
			Aliases: []string{"d"},
			Usage:   "configures asterisk dongles with json",
			Action:  Dongles,
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatalf("fconf: %v", err)
	}
}
