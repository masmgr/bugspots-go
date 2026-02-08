package cmd

import (
	"time"

	"github.com/masmgr/bugspots-go/config"
	"github.com/masmgr/bugspots-go/internal/aggregation"
	"github.com/masmgr/bugspots-go/internal/git"
	"github.com/masmgr/bugspots-go/internal/output"
	"github.com/masmgr/bugspots-go/internal/scoring"
	"github.com/urfave/cli/v2"
)

// CommitsCmd returns the commits command.
func CommitsCmd() *cli.Command {
	flags := append(commonFlags(),
		&cli.StringFlag{
			Name:    "risk-level",
			Aliases: []string{"l"},
			Usage:   "Filter by minimum risk level (high, medium, all)",
			Value:   "all",
		},
	)

	return &cli.Command{
		Name:    "commits",
		Aliases: []string{"c"},
		Usage:   "Analyze commit risk using JIT defect prediction",
		Flags:   flags,
		Action:  commitsAction,
	}
}

func commitsAction(c *cli.Context) error {
	return executeWithContext(c, git.ChangeDetailFull, func(ctx *CommandContext, c *cli.Context) error {
		// Calculate commit metrics
		calculator := aggregation.NewCommitMetricsCalculator()
		metrics := calculator.CalculateAll(ctx.ChangeSets)

		// Calculate risk scores
		explain := c.Bool("explain")
		scorer := scoring.NewCommitScorer(ctx.Config.CommitScoring)
		items := scorer.ScoreAndRank(metrics, explain)

		// Filter by risk level
		riskLevel := parseRiskLevel(c.String("risk-level"))
		if riskLevel != "" {
			items = scoring.FilterByRiskLevel(items, riskLevel)
		}

		// Create report
		report := &output.CommitAnalysisReport{
			RepoPath:    ctx.RepoPath,
			Since:       ctx.Since,
			Until:       ctx.Until,
			GeneratedAt: time.Now(),
			Items:       items,
		}

		// Output results
		opts := OutputOptions(c)
		writer := output.NewCommitReportWriter(opts.Format)
		if err := writer.Write(report, opts); err != nil {
			return err
		}

		return nil
	})
}

func parseRiskLevel(s string) config.RiskLevel {
	switch s {
	case "high":
		return config.RiskLevelHigh
	case "medium":
		return config.RiskLevelMedium
	default:
		return "" // all
	}
}
