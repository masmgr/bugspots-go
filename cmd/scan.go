package cmd

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/masmgr/bugspots-go/config"
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

	// Open repository
	r, err := git.PlainOpen(repoPath)
	if err != nil {
		return fmt.Errorf("invalid Git repository - please run from or specify the full path to the root of the project: %w", err)
	}

	// Resolve starting reference (branch/ref)
	fromHash, err := resolveFromHash(r, branch)
	if err != nil {
		return err
	}

	// Calculate time range
	until := time.Now()
	since := until.AddDate(-cfg.Legacy.AnalysisWindowYears, 0, 0)

	// Get commit iterator
	cIter, err := r.Log(&git.LogOptions{From: fromHash, Since: &since, Until: &until})
	if err != nil {
		return fmt.Errorf("failed to get commit log: %w", err)
	}

	// Get fixes
	fixes, err := getFixes(cIter, regex)
	if err != nil {
		return fmt.Errorf("failed to get fixes: %w", err)
	}

	// Calculate hotspots
	hotspots := scoring.CalculateLegacyHotspots(fixes, until, since)

	// Display results
	showScanResult(fixes, hotspots, cfg.Legacy.MaxHotspots, displayTimestamps)

	fmt.Fprintf(os.Stderr, "\nCompleted in %s\n", time.Since(start))
	return nil
}

func resolveFromHash(repo *git.Repository, branch string) (plumbing.Hash, error) {
	branch = strings.TrimSpace(branch)
	if branch == "" || strings.EqualFold(branch, "HEAD") {
		ref, err := repo.Head()
		if err != nil {
			return plumbing.ZeroHash, fmt.Errorf("failed to get HEAD: %w", err)
		}
		return ref.Hash(), nil
	}

	remoteRef := plumbing.ReferenceName("")
	if !strings.HasPrefix(branch, "refs/") && strings.Contains(branch, "/") {
		if parts := strings.SplitN(branch, "/", 2); len(parts) == 2 && parts[0] != "" && parts[1] != "" {
			remoteRef = plumbing.NewRemoteReferenceName(parts[0], parts[1])
		}
	}

	candidates := []plumbing.ReferenceName{
		plumbing.ReferenceName(branch),
		plumbing.NewBranchReferenceName(branch),
		plumbing.NewRemoteReferenceName("origin", branch),
		remoteRef,
		plumbing.NewTagReferenceName(branch),
	}

	for _, name := range candidates {
		if name == "" {
			continue
		}
		ref, err := repo.Reference(name, true)
		if err == nil {
			return ref.Hash(), nil
		}
	}

	h, err := repo.ResolveRevision(plumbing.Revision(branch))
	if err == nil && h != nil {
		return *h, nil
	}

	return plumbing.ZeroHash, fmt.Errorf("branch/ref not found: %q", branch)
}

func getFixes(cIter object.CommitIter, regex *regexp.Regexp) ([]scoring.LegacyFix, error) {
	var fixes []scoring.LegacyFix

	err := cIter.ForEach(func(c *object.Commit) error {
		// Check if commit message matches bugfix pattern
		if regex != nil && len(regex.FindStringSubmatch(c.Message)) == 0 {
			return nil
		}

		// Skip commits without parents (initial commit)
		if c.NumParents() == 0 {
			return nil
		}

		tree, err := c.Tree()
		if err != nil {
			return err
		}

		parent, err := c.Parent(0)
		if err != nil {
			return err
		}

		parentTree, err := parent.Tree()
		if err != nil {
			return err
		}

		changes, err := object.DiffTree(parentTree, tree)
		if err != nil {
			return err
		}

		files := make([]string, 0, len(changes))
		for _, change := range changes {
			if change.From.Name != "" {
				files = append(files, change.From.Name)
			} else {
				files = append(files, change.To.Name)
			}
		}

		// Extract first line of commit message efficiently
		message := c.Message
		if idx := strings.IndexByte(message, '\n'); idx != -1 {
			message = message[:idx]
		}

		fixes = append(fixes, scoring.LegacyFix{
			Message: message,
			Date:    c.Committer.When,
			Files:   files,
		})

		return nil
	})

	return fixes, err
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
