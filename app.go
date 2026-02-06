package main

import (
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/masmgr/bugspots-go/cmd"
	"github.com/urfave/cli/v2"
)

func main() {
	// Build the new app with subcommands
	app := cmd.App()

	// Add legacy flags to the root command for backward compatibility
	app.Flags = append(app.Flags,
		&cli.StringFlag{
			Name:    "branch",
			Aliases: []string{"b"},
			Value:   "master",
			Usage:   "branch to crawl (legacy mode)",
		},
		&cli.StringFlag{
			Name:    "depth",
			Aliases: []string{"d"},
			Usage:   "depth (legacy mode)",
		},
		&cli.StringFlag{
			Name:    "words",
			Aliases: []string{"w"},
			Usage:   "bugfix indicator word list, ie: \"fixes,closed\" (legacy mode)",
		},
		&cli.StringFlag{
			Name:    "regex",
			Aliases: []string{"r"},
			Usage:   "bugfix indicator regex (legacy mode)",
		},
		&cli.BoolFlag{
			Name:  "display-timestamps",
			Usage: "show timestamps of each identified fix commit (legacy mode)",
		},
	)

	// Override the default action for legacy support
	app.Action = func(c *cli.Context) error {
		// If a subcommand was invoked, this won't be called
		// If no args, show help
		if c.NArg() == 0 {
			return cli.ShowAppHelp(c)
		}

		// Legacy mode: treat first arg as repo path
		Scan(
			c.Args().Get(0),
			c.String("branch"),
			c.Int("depth"),
			getRegexp(c),
		)
		return nil
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func convertToRegex(words string) string {
	return strings.Join(strings.Split(words, ","), "|")
}

func getRegexp(c *cli.Context) *regexp.Regexp {
	var r_str string
	if len(c.String("words")) > 0 {
		r_str = convertToRegex(c.String("words"))
	} else if len(c.String("regex")) > 0 {
		r_str = c.String("regex")
	} else {
		r_str = `\b(fix(es|ed)?|close(s|d)?)\b`
	}

	var r *regexp.Regexp
	if r_str != "" {
		r, _ = regexp.Compile(r_str)
		return r
	}
	return nil
}
