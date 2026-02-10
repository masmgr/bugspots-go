package cmd

import (
	"time"

	"github.com/urfave/cli/v2"

	"github.com/masmgr/bugspots-go/internal/coupling"
	"github.com/masmgr/bugspots-go/internal/git"
	"github.com/masmgr/bugspots-go/internal/output"
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
	return executeWithContext(c, git.ChangeDetailPathsOnly, func(ctx *CommandContext, c *cli.Context) error {
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
		if err := writeCouplingReport(c, report); err != nil {
			return err
		}

		return nil
	})
}
