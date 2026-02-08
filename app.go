package main

import (
	"log"
	"os"

	"github.com/masmgr/bugspots-go/cmd"
)

func main() {
	// Build the app with subcommands
	app := cmd.App()

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
