package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/masmgr/bugspots-go/config"
	"github.com/masmgr/bugspots-go/internal/output"
	"github.com/urfave/cli/v2"
)

// App creates the CLI application.
func App() *cli.App {
	return &cli.App{
		Name:    "bugspots",
		Usage:   "Bug prediction tool for Git repositories",
		Version: "2.0.0",
		Commands: []*cli.Command{
			AnalyzeCmd(),
			CommitsCmd(),
			CouplingCmd(),
			ScanCmd(),
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "Path to configuration file",
			},
		},
		Action: legacyAction,
	}
}

// Common flags shared across commands
func commonFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:    "repo",
			Aliases: []string{"r"},
			Usage:   "Path to Git repository",
			Value:   ".",
		},
		&cli.StringFlag{
			Name:  "since",
			Usage: "Analyze commits since this date (YYYY-MM-DD)",
		},
		&cli.StringFlag{
			Name:  "until",
			Usage: "Analyze commits until this date (YYYY-MM-DD)",
		},
		&cli.StringFlag{
			Name:    "branch",
			Aliases: []string{"b"},
			Usage:   "Branch to analyze",
		},
		&cli.StringFlag{
			Name:  "rename-detect",
			Usage: "Rename detection mode (auto, off, simple, aggressive)",
			Value: "simple",
		},
		&cli.StringSliceFlag{
			Name:  "include",
			Usage: "Glob patterns to include (can be specified multiple times)",
		},
		&cli.StringSliceFlag{
			Name:  "exclude",
			Usage: "Glob patterns to exclude (can be specified multiple times)",
		},
		&cli.StringFlag{
			Name:    "format",
			Aliases: []string{"f"},
			Usage:   "Output format (console, json, csv, markdown, ci)",
			Value:   "console",
		},
		&cli.IntFlag{
			Name:    "top",
			Aliases: []string{"n"},
			Usage:   "Number of top results to show",
			Value:   50,
		},
		&cli.StringFlag{
			Name:    "output",
			Aliases: []string{"o"},
			Usage:   "Output file path (default: stdout)",
		},
		&cli.BoolFlag{
			Name:  "explain",
			Usage: "Show score breakdown",
		},
	}
}

// parseDateFlag parses a date string flag.
func parseDateFlag(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil, fmt.Errorf("invalid date format: %s (expected YYYY-MM-DD)", s)
	}
	return &t, nil
}

// getOutputFormat parses the output format flag.
func getOutputFormat(s string) output.OutputFormat {
	switch s {
	case "json":
		return output.FormatJSON
	case "csv":
		return output.FormatCSV
	case "markdown", "md":
		return output.FormatMarkdown
	case "ci", "ndjson":
		return output.FormatCI
	default:
		return output.FormatConsole
	}
}

// loadConfig loads configuration from file or defaults.
func loadConfig(c *cli.Context) (*config.Config, error) {
	configPath := c.String("config")
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Apply filter overrides from CLI
	if includes := c.StringSlice("include"); len(includes) > 0 {
		cfg.Filters.Include = includes
	}
	if excludes := c.StringSlice("exclude"); len(excludes) > 0 {
		cfg.Filters.Exclude = excludes
	}

	return cfg, nil
}

// legacyAction handles the default (legacy) command behavior.
// When a repository path is provided as an argument, it runs the scan command.
func legacyAction(c *cli.Context) error {
	// If no args and no subcommand, show help
	if c.NArg() == 0 {
		return cli.ShowAppHelp(c)
	}

	// Legacy mode: treat first arg as repo path and run scan
	// This maintains backward compatibility with the original bugspots CLI
	return ScanCmd().Action(c)
}

// Run executes the CLI application.
func Run() {
	if err := App().Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
