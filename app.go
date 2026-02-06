package main

import (
	"log"
	"os"

	"github.com/masmgr/bugspots-go/cmd"
)

func main() {
	// Build the app with subcommands
	app := cmd.App()

	// Add legacy flags to the root command for backward compatibility.
	// These flags are used when running `bugspots /path/to/repo` without a subcommand.
	app.Flags = append(app.Flags, cmd.LegacyScanFlags()...)

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
