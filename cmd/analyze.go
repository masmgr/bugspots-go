package cmd

import (
	"time"

	"github.com/masmgr/bugspots-go/internal/aggregation"
	"github.com/masmgr/bugspots-go/internal/burst"
	"github.com/masmgr/bugspots-go/internal/output"
	"github.com/masmgr/bugspots-go/internal/scoring"
	"github.com/urfave/cli/v2"
)

// AnalyzeCmd returns the analyze command.
func AnalyzeCmd() *cli.Command {
	flags := append(commonFlags(),
		&cli.IntFlag{
			Name:  "half-life",
			Usage: "Half-life in days for recency decay",
			Value: 30,
		},
		&cli.IntFlag{
			Name:  "window-days",
			Usage: "Window size in days for burst detection",
			Value: 7,
		},
	)

	return &cli.Command{
		Name:    "analyze",
		Aliases: []string{"a"},
		Usage:   "Analyze file hotspots using 5-factor scoring",
		Flags:   flags,
		Action:  analyzeAction,
	}
}

func analyzeAction(c *cli.Context) error {
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
	if halfLife := c.Int("half-life"); halfLife > 0 {
		ctx.Config.Scoring.HalfLifeDays = halfLife
	}
	if windowDays := c.Int("window-days"); windowDays > 0 {
		ctx.Config.Burst.WindowDays = windowDays
	}

	// Aggregate file metrics
	aggregator := aggregation.NewFileMetricsAggregator()
	metrics := aggregator.Process(ctx.ChangeSets)

	// Calculate burst scores
	burstCalc := burst.NewCalculator(ctx.Config.Burst.WindowDays)
	burstCalc.Compute(metrics)

	// Calculate risk scores
	explain := c.Bool("explain")
	scorer := scoring.NewFileScorer(ctx.Config.Scoring)
	items := scorer.ScoreAndRank(metrics, explain, ctx.Until)

	// Create report
	report := &output.FileAnalysisReport{
		RepoPath:    ctx.RepoPath,
		Since:       ctx.Since,
		Until:       ctx.Until,
		GeneratedAt: time.Now(),
		Items:       items,
	}

	// Output results
	opts := OutputOptions(c)
	writer := output.NewFileReportWriter(opts.Format)
	return writer.Write(report, opts)
}
