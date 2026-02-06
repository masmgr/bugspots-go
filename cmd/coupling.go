package cmd

import (
	"time"

	"github.com/masmgr/bugspots-go/internal/coupling"
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
	// Create command context (handles config, dates, git reader)
	ctx, err := NewCommandContext(c)
	if err != nil {
		return err
	}

	if !ctx.HasCommits() {
		ctx.PrintNoCommitsMessage()
		return nil
	}

	// Override config from CLI flags
	if minCoCommits := c.Int("min-co-commits"); minCoCommits > 0 {
		ctx.Config.Coupling.MinCoCommits = minCoCommits
	}
	if minJaccard := c.Float64("min-jaccard"); minJaccard > 0 {
		ctx.Config.Coupling.MinJaccardThreshold = minJaccard
	}
	if maxFiles := c.Int("max-files"); maxFiles > 0 {
		ctx.Config.Coupling.MaxFilesPerCommit = maxFiles
	}
	if topPairs := c.Int("top-pairs"); topPairs > 0 {
		ctx.Config.Coupling.TopPairs = topPairs
	}

	// Analyze coupling
	analyzer := coupling.NewAnalyzer(ctx.Config.Coupling)
	result := analyzer.Analyze(ctx.ChangeSets)

	// Create report
	report := &output.CouplingAnalysisReport{
		RepoPath:    ctx.RepoPath,
		Since:       ctx.Since,
		Until:       ctx.Until,
		GeneratedAt: time.Now(),
		Result:      result,
	}

	// Output results
	opts := OutputOptions(c)
	writer := output.NewCouplingReportWriter(opts.Format)
	return writer.Write(report, opts)
}
