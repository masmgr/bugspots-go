package cmd

import (
	"fmt"
	"time"

	"github.com/masmgr/bugspots-go/internal/coupling"
	"github.com/masmgr/bugspots-go/internal/git"
	"github.com/masmgr/bugspots-go/internal/output"
	"github.com/urfave/cli/v2"
)

// CouplingCmd returns the coupling command.
func CouplingCmd() *cli.Command {
	flags := append(commonFlags(),
		&cli.IntFlag{
			Name:  "min-co-commits",
			Usage: "Minimum number of co-commits to consider a coupling",
			Value: 3,
		},
		&cli.Float64Flag{
			Name:  "min-jaccard",
			Usage: "Minimum Jaccard coefficient threshold",
			Value: 0.1,
		},
		&cli.IntFlag{
			Name:  "max-files",
			Usage: "Maximum files per commit to consider (skip large refactoring commits)",
			Value: 50,
		},
		&cli.IntFlag{
			Name:  "top-pairs",
			Usage: "Number of top coupled pairs to report",
			Value: 50,
		},
	)

	return &cli.Command{
		Name:    "coupling",
		Aliases: []string{"cp"},
		Usage:   "Analyze file change coupling patterns",
		Flags:   flags,
		Action:  couplingAction,
	}
}

func couplingAction(c *cli.Context) error {
	// Load configuration
	cfg, err := loadConfig(c)
	if err != nil {
		return err
	}

	// Override config from CLI flags
	if minCoCommits := c.Int("min-co-commits"); minCoCommits > 0 {
		cfg.Coupling.MinCoCommits = minCoCommits
	}
	if minJaccard := c.Float64("min-jaccard"); minJaccard > 0 {
		cfg.Coupling.MinJaccardThreshold = minJaccard
	}
	if maxFiles := c.Int("max-files"); maxFiles > 0 {
		cfg.Coupling.MaxFilesPerCommit = maxFiles
	}
	if topPairs := c.Int("top-pairs"); topPairs > 0 {
		cfg.Coupling.TopPairs = topPairs
	}

	// Parse date flags
	since, err := parseDateFlag(c.String("since"))
	if err != nil {
		return err
	}
	until, err := parseDateFlag(c.String("until"))
	if err != nil {
		return err
	}

	untilTime := time.Now()
	if until != nil {
		untilTime = *until
	}

	// Set up Git reader
	repoPath := c.String("repo")
	reader, err := git.NewHistoryReader(git.ReadOptions{
		RepoPath: repoPath,
		Branch:   c.String("branch"),
		Since:    since,
		Until:    until,
		Include:  cfg.Filters.Include,
		Exclude:  cfg.Filters.Exclude,
	})
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	// Read commit changes
	changeSets, err := reader.ReadChanges()
	if err != nil {
		return fmt.Errorf("failed to read history: %w", err)
	}

	if len(changeSets) == 0 {
		fmt.Println("No commits found in the specified range.")
		return nil
	}

	// Analyze coupling
	analyzer := coupling.NewAnalyzer(cfg.Coupling)
	result := analyzer.Analyze(changeSets)

	// Create report
	report := &output.CouplingAnalysisReport{
		RepoPath:    repoPath,
		Since:       since,
		Until:       untilTime,
		GeneratedAt: time.Now(),
		Result:      result,
	}

	// Output results
	format := getOutputFormat(c.String("format"))
	writer := output.NewCouplingReportWriter(format)
	return writer.Write(report, output.OutputOptions{
		Format:     format,
		Top:        c.Int("top"),
		OutputPath: c.String("output"),
		Explain:    c.Bool("explain"),
	})
}
