package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/masmgr/bugspots-go/internal/aggregation"
	"github.com/masmgr/bugspots-go/internal/bugfix"
	"github.com/masmgr/bugspots-go/internal/burst"
	"github.com/masmgr/bugspots-go/internal/complexity"
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
		&cli.StringSliceFlag{
			Name:  "bug-patterns",
			Usage: "Regex patterns for bugfix commit detection (can be specified multiple times)",
		},
		&cli.StringFlag{
			Name:  "diff",
			Usage: "Analyze only files changed between refs (e.g., origin/main...HEAD)",
		},
		&cli.Float64Flag{
			Name:  "ci-threshold",
			Usage: "Exit with non-zero status if any file exceeds this risk score",
		},
		&cli.BoolFlag{
			Name:  "include-complexity",
			Usage: "Include file complexity (line count) in scoring",
		},
	)

	return &cli.Command{
		Name:    "analyze",
		Aliases: []string{"a"},
		Usage:   "Analyze file hotspots using 6-factor scoring",
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
	defer ctx.LogCompletion()

	if !ctx.HasCommits() {
		ctx.PrintNoCommitsMessage()
		return nil
	}

	// Override config from CLI flags
	ctx.ApplyCLIOverrides(c)

	// Aggregate file metrics
	aggregator := aggregation.NewFileMetricsAggregator()
	metrics := aggregator.Process(ctx.ChangeSets)

	// Detect bugfix commits and apply counts
	bugPatterns := c.StringSlice("bug-patterns")
	if len(bugPatterns) == 0 {
		bugPatterns = ctx.Config.Bugfix.Patterns
	}
	if len(bugPatterns) > 0 {
		detector, err := bugfix.NewDetector(bugPatterns)
		if err != nil {
			return fmt.Errorf("invalid bug pattern: %w", err)
		}
		result := detector.Detect(ctx.ChangeSets)
		aggregation.ApplyBugfixCounts(metrics, aggregator, result.FileBugfixCounts)
	}

	// Calculate burst scores
	burstCalc := burst.NewCalculator(ctx.Config.Burst.WindowDays)
	burstCalc.Compute(metrics)

	// Measure file complexity (line counts) if requested
	if c.Bool("include-complexity") {
		pathSet := make(map[string]struct{}, len(metrics))
		for p := range metrics {
			pathSet[p] = struct{}{}
		}
		branch := ctx.Branch
		if branch == "" {
			branch = "HEAD"
		}
		lineCounts, err := complexity.FileLineCounts(context.Background(), ctx.RepoPath, branch, pathSet)
		if err != nil {
			return fmt.Errorf("failed to measure file complexity: %w", err)
		}
		for path, count := range lineCounts {
			if fm, ok := metrics[path]; ok {
				fm.FileSize = count
			}
		}
	} else {
		// Zero out complexity weight when not measuring
		ctx.Config.Scoring.Weights.Complexity = 0
	}

	// Calculate risk scores
	explain := c.Bool("explain")
	scorer := scoring.NewFileScorer(ctx.Config.Scoring)
	items := scorer.ScoreAndRank(metrics, explain, ctx.Until)

	// Filter by diff if specified
	if diffSpec := c.String("diff"); diffSpec != "" {
		diffResult, err := git.ReadDiff(context.Background(), git.DiffOptions{
			RepoPath: ctx.RepoPath,
			DiffSpec: diffSpec,
		})
		if err != nil {
			return fmt.Errorf("failed to read diff: %w", err)
		}
		items = filterByDiff(items, diffResult)
	}

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
	if err := writer.Write(report, opts); err != nil {
		return err
	}

	// Check CI threshold
	if threshold := c.Float64("ci-threshold"); threshold > 0 && len(items) > 0 {
		if items[0].RiskScore >= threshold {
			return fmt.Errorf("risk threshold exceeded: %s has score %.4f (threshold: %.4f)",
				items[0].Path, items[0].RiskScore, threshold)
		}
	}

	return nil
}

// filterByDiff filters scored items to only include files present in the diff result.
func filterByDiff(items []scoring.FileRiskItem, diff *git.DiffResult) []scoring.FileRiskItem {
	pathSet := make(map[string]struct{}, len(diff.ChangedFiles))
	for _, f := range diff.ChangedFiles {
		pathSet[f.Path] = struct{}{}
		if f.OldPath != "" {
			pathSet[f.OldPath] = struct{}{}
		}
	}

	filtered := make([]scoring.FileRiskItem, 0, len(pathSet))
	for _, item := range items {
		if _, ok := pathSet[item.Path]; ok {
			filtered = append(filtered, item)
		}
	}
	return filtered
}
