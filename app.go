package main

import (
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "bugspots",
		Usage: "Usage: bugspots /path/to/git/repo",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "branch",
				Aliases: []string{"b"},
				Value:   "master",
				Usage:   "branch to crawl",
			},
			&cli.StringFlag{
				Name:    "depth",
				Aliases: []string{"d"},
				Usage:   "branch to crawl",
			},
			&cli.StringFlag{
				Name:    "words",
				Aliases: []string{"w"},
				Usage:   "dbugfix indicator word list, ie: \"fixes,closed\"",
			},
			&cli.StringFlag{
				Name:    "regex",
				Aliases: []string{"r"},
				Usage:   "bugfix indicator regex, ie: \"fix(es|ed)?\" or \"/fixes #(\\d+)/i\"",
			},
			&cli.BoolFlag{
				Name:  "display-timestamps",
				Usage: "show timestamps of each identified fix commit",
			},
		},

		Action: func(c *cli.Context) error {
			Scan(
				c.Args().Get(0),
				c.String("branch"),
				c.Int("depth"),
				getRegexp(c),
			)
			return nil
		},
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
		r_str = "\b(fix(es|ed)?|close(s|d)?)\b"
	}

	var r *regexp.Regexp
	if r_str != "" {
		r, _ = regexp.Compile(r_str)
		return r
	}
	return nil
}
