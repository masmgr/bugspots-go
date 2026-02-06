package cmd

import (
	"fmt"
	"time"

	"github.com/masmgr/bugspots-go/internal/aggregation"
	"github.com/masmgr/bugspots-go/internal/burst"
	"github.com/masmgr/bugspots-go/internal/git"
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
	// Load configuration
	cfg, err := loadConfig(c)
	if err != nil {
		return err
	}

	// Override config from CLI flags
	if halfLife := c.Int("half-life"); halfLife > 0 {
		cfg.Scoring.HalfLifeDays = halfLife
	}
	if windowDays := c.Int("window-days"); windowDays > 0 {
		cfg.Burst.WindowDays = windowDays
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

	// Aggregate file metrics
	aggregator := aggregation.NewFileMetricsAggregator()
	metrics := aggregator.Process(changeSets)

	// Calculate burst scores
	burstCalc := burst.NewCalculator(cfg.Burst.WindowDays)
	burstCalc.Compute(metrics)

	// Calculate risk scores
	explain := c.Bool("explain")
	scorer := scoring.NewFileScorer(cfg.Scoring)
	items := scorer.ScoreAndRank(metrics, explain, untilTime)

	// Create report
	report := &output.FileAnalysisReport{
		RepoPath:    repoPath,
		Since:       since,
		Until:       untilTime,
		GeneratedAt: time.Now(),
		Items:       items,
	}

	// Output results
	format := getOutputFormat(c.String("format"))
	writer := output.NewFileReportWriter(format)
	return writer.Write(report, output.OutputOptions{
		Format:     format,
		Top:        c.Int("top"),
		OutputPath: c.String("output"),
		Explain:    explain,
	})
}
