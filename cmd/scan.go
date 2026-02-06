package cmd

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/masmgr/bugspots-go/config"
	gitpkg "github.com/masmgr/bugspots-go/internal/git"
	"github.com/masmgr/bugspots-go/internal/scoring"
	"github.com/urfave/cli/v2"
)

// ScanCmd creates the scan command for legacy bugspots analysis.
func ScanCmd() *cli.Command {
	return &cli.Command{
		Name:      "scan",
		Usage:     "Classic bugspots analysis (legacy compatibility)",
		ArgsUsage: "[repository path]",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "branch",
				Aliases: []string{"b"},
				Value:   "",
				Usage:   "Branch to analyze (default: from config or 'master')",
			},
			&cli.IntFlag{
				Name:    "depth",
				Aliases: []string{"d"},
				Usage:   "Depth of commits to analyze (not implemented)",
			},
			&cli.StringFlag{
				Name:    "words",
				Aliases: []string{"w"},
				Usage:   "Bugfix indicator word list, e.g., \"fixes,closed\"",
			},
			&cli.StringFlag{
				Name:    "regex",
				Aliases: []string{"r"},
				Usage:   "Bugfix indicator regex pattern",
			},
			&cli.BoolFlag{
				Name:  "display-timestamps",
				Usage: "Show timestamps of each identified fix commit",
			},
			&cli.StringFlag{
				Name:  "config",
				Usage: "Path to configuration file",
			},
		},
		Action: scanAction,
	}
}

func scanAction(c *cli.Context) error {
	// Load configuration
	cfg, err := loadConfig(c)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get repository path
	repoPath := "."
	if c.NArg() > 0 {
		repoPath = c.Args().Get(0)
	}

	// Get branch (from flag or config)
	branch := c.String("branch")
	if branch == "" {
		branch = cfg.Legacy.DefaultBranch
	}

	// Build regex pattern
	regex, err := buildBugfixRegex(c, cfg)
	if err != nil {
		return fmt.Errorf("failed to build regex: %w", err)
	}

	// Run the scan
	return runScan(repoPath, branch, regex, cfg, c.Bool("display-timestamps"))
}

func buildBugfixRegex(c *cli.Context, cfg *config.Config) (*regexp.Regexp, error) {
	var pattern string

	if words := c.String("words"); words != "" {
		// Convert word list to regex pattern
		pattern = convertToRegex(words)
	} else if regexStr := c.String("regex"); regexStr != "" {
		pattern = regexStr
	} else {
		pattern = cfg.Legacy.DefaultBugfixRegex
	}

	if pattern == "" {
		return nil, nil
	}

	return regexp.Compile(pattern)
}

// convertToRegex converts a comma-separated word list to a regex pattern.
func convertToRegex(words string) string {
	parts := strings.Split(words, ",")
	tokens := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		tokens = append(tokens, regexp.QuoteMeta(p))
	}
	return strings.Join(tokens, "|")
}

func runScan(repoPath string, branch string, regex *regexp.Regexp, cfg *config.Config, displayTimestamps bool) error {
	start := time.Now()
	color.Green("Scanning %v repo", repoPath)

	// Calculate time range
	until := time.Now()
	since := until.AddDate(-cfg.Legacy.AnalysisWindowYears, 0, 0)

	// Read commit history using HistoryReader
	reader, err := gitpkg.NewHistoryReader(gitpkg.ReadOptions{
		RepoPath:     repoPath,
		Branch:       branch,
		Since:        &since,
		Until:        &until,
		Include:      cfg.Filters.Include,
		Exclude:      cfg.Filters.Exclude,
		DetailLevel:  gitpkg.ChangeDetailPathsOnly,
		RenameDetect: gitpkg.RenameDetectSimple,
	})
	if err != nil {
		return fmt.Errorf("invalid Git repository - please run from or specify the full path to the root of the project: %w", err)
	}

	changeSets, err := reader.ReadChanges(context.Background())
	if err != nil {
		return fmt.Errorf("failed to read history: %w", err)
	}

	// Filter commits by bugfix regex and build fixes list
	fixes := getFixes(changeSets, regex)

	// Calculate hotspots
	hotspots := scoring.CalculateLegacyHotspots(fixes, until, since)

	// Display results
	showScanResult(fixes, hotspots, cfg.Legacy.MaxHotspots, displayTimestamps)

	fmt.Fprintf(os.Stderr, "\nCompleted in %s\n", time.Since(start))
	return nil
}

func getFixes(changeSets []gitpkg.CommitChangeSet, regex *regexp.Regexp) []scoring.LegacyFix {
	var fixes []scoring.LegacyFix

	for _, cs := range changeSets {
		// Check if commit message matches bugfix pattern
		if regex != nil && !regex.MatchString(cs.Commit.Message) {
			continue
		}

		files := make([]string, 0, len(cs.Changes))
		for _, change := range cs.Changes {
			if change.Path != "" {
				files = append(files, change.Path)
			}
		}

		if len(files) == 0 {
			continue
		}

		fixes = append(fixes, scoring.LegacyFix{
			Message: cs.Commit.Message,
			Date:    cs.Commit.When,
			Files:   files,
		})
	}

	return fixes
}

func showScanResult(fixes []scoring.LegacyFix, hotspots map[string]float64, maxSpots int, displayTimestamps bool) {
	spots := scoring.RankLegacyHotspots(hotspots, maxSpots)

	// Sort fixes by date (newest first) for display
	sortedFixes := make([]scoring.LegacyFix, len(fixes))
	copy(sortedFixes, fixes)
	sort.Slice(sortedFixes, func(i, j int) bool {
		return sortedFixes[i].Date.After(sortedFixes[j].Date)
	})

	fmt.Print("\t")
	color.Yellow("Found %v bugfix commits, with %v hotspots:", len(fixes), len(spots))
	fmt.Println("")

	colorTitle := color.New(color.FgGreen).Add(color.Underline)

	fmt.Print("\t")
	colorTitle.Println("Fixes:")
	for _, fix := range sortedFixes {
		fmt.Print("\t\t")
		var buff strings.Builder
		buff.WriteString("- ")
		if displayTimestamps {
			buff.WriteString(fix.Date.Format("2006-01-02 15:04:05"))
			buff.WriteString(" ")
		}
		buff.WriteString(fix.Message)
		color.Yellow(buff.String())
	}

	fmt.Println("")
	fmt.Print("\t")
	colorTitle.Println("Hotspots:")

	colorSpot := color.New(color.FgRed)
	colorScore := color.New(color.FgYellow)

	for _, spot := range spots {
		fmt.Print("\t\t")
		colorSpot.Print(color.RedString("%v", spot.File))
		colorScore.Println(color.YellowString(" - %.3f", spot.Score))
	}
}
