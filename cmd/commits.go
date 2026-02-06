package cmd

import (
	"fmt"
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
	// Load configuration
	cfg, err := loadConfig(c)
	if err != nil {
		return err
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

	// Calculate commit metrics
	calculator := aggregation.NewCommitMetricsCalculator()
	metrics := calculator.CalculateAll(changeSets)

	// Calculate risk scores
	explain := c.Bool("explain")
	scorer := scoring.NewCommitScorer(cfg.CommitScoring)
	items := scorer.ScoreAndRank(metrics, explain)

	// Filter by risk level
	riskLevel := parseRiskLevel(c.String("risk-level"))
	if riskLevel != "" {
		items = scoring.FilterByRiskLevel(items, riskLevel)
	}

	// Create report
	report := &output.CommitAnalysisReport{
		RepoPath:    repoPath,
		Since:       since,
		Until:       untilTime,
		GeneratedAt: time.Now(),
		Items:       items,
	}

	// Output results
	format := getOutputFormat(c.String("format"))
	writer := output.NewCommitReportWriter(format)
	return writer.Write(report, output.OutputOptions{
		Format:     format,
		Top:        c.Int("top"),
		OutputPath: c.String("output"),
		Explain:    explain,
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
