package main

import (
	"btest/cmd"
	"log"
	"os"

	cli "github.com/urfave/cli/v2"
)

func main() {

	app := &cli.App{
		Flags:  cmd.Par,
		Action: cmd.Run,
		Name:   "sudp",
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
