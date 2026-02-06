package main

import (
	"log"
	"os"

	"github.com/masmgr/bugspots-go/cmd"
	"github.com/urfave/cli/v2"
)

func main() {
	// Build the app with subcommands
	app := cmd.App()

	// Add legacy flags to the root command for backward compatibility
	// These flags are used when running `bugspots /path/to/repo` without a subcommand
	app.Flags = append(app.Flags,
		&cli.StringFlag{
			Name:    "branch",
			Aliases: []string{"b"},
			Usage:   "Branch to analyze (legacy mode)",
		},
		&cli.IntFlag{
			Name:    "depth",
			Aliases: []string{"d"},
			Usage:   "Depth of commits to analyze (legacy mode, not implemented)",
		},
		&cli.StringFlag{
			Name:    "words",
			Aliases: []string{"w"},
			Usage:   "Bugfix indicator word list, e.g., \"fixes,closed\" (legacy mode)",
		},
		&cli.StringFlag{
			Name:    "regex",
			Aliases: []string{"r"},
			Usage:   "Bugfix indicator regex pattern (legacy mode)",
		},
		&cli.BoolFlag{
			Name:  "display-timestamps",
			Usage: "Show timestamps of each identified fix commit (legacy mode)",
		},
	)

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
