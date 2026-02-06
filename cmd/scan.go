package cmd

import (
	"fmt"
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

// LegacyScanFlags returns the flags for the legacy scan command.
// These are shared between ScanCmd and the root app (for backward compatibility).
func LegacyScanFlags() []cli.Flag {
	return []cli.Flag{
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
	}
}

// ScanCmd creates the scan command for legacy bugspots analysis.
func ScanCmd() *cli.Command {
	return &cli.Command{
		Name:      "scan",
		Usage:     "Classic bugspots analysis (legacy compatibility)",
		ArgsUsage: "[repository path]",
		Flags:     LegacyScanFlags(),
		Action:    scanAction,
	}
}

func scanAction(c *cli.Context) error {
	// Legacy: repo path from positional arg
	if c.NArg() > 0 && !c.IsSet("repo") {
		_ = c.Set("repo", c.Args().Get(0))
	}

	// Load configuration
	cfg, err := loadConfig(c)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Apply default branch from config if not specified via flag
	if !c.IsSet("branch") {
		_ = c.Set("branch", cfg.Legacy.DefaultBranch)
	}

	// Apply default since date from config's analysis window
	if !c.IsSet("since") {
		since := time.Now().AddDate(-cfg.Legacy.AnalysisWindowYears, 0, 0)
		_ = c.Set("since", since.Format("2006-01-02"))
	}

	ctx, err := NewCommandContextWithGitDetail(c, gitpkg.ChangeDetailPathsOnly)
	if err != nil {
		return err
	}
	defer ctx.LogCompletion()

	color.Green("Scanning %v repo", ctx.RepoPath)

	// Build regex pattern
	regex, err := buildBugfixRegex(c, ctx.Config)
	if err != nil {
		return fmt.Errorf("failed to build regex: %w", err)
	}

	// Filter commits by bugfix regex and build fixes list
	fixes := getFixes(ctx.ChangeSets, regex)

	// Calculate hotspots
	since := ctx.Until.AddDate(-ctx.Config.Legacy.AnalysisWindowYears, 0, 0)
	if ctx.Since != nil {
		since = *ctx.Since
	}
	hotspots := scoring.CalculateLegacyHotspots(fixes, ctx.Until, since)

	// Display results
	showScanResult(fixes, hotspots, ctx.Config.Legacy.MaxHotspots, c.Bool("display-timestamps"))

	return nil
}

func buildBugfixRegex(c *cli.Context, cfg *config.Config) (*regexp.Regexp, error) {
	var pattern string

	if words := c.String("words"); words != "" {
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
