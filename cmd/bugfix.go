package cmd

import (
	"fmt"

	"github.com/masmgr/bugspots-go/config"
	"github.com/masmgr/bugspots-go/internal/aggregation"
	"github.com/masmgr/bugspots-go/internal/bugfix"
	"github.com/masmgr/bugspots-go/internal/git"
	"github.com/urfave/cli/v2"
)

func resolveBugPatterns(c *cli.Context, cfg *config.Config) []string {
	patterns := c.StringSlice("bug-patterns")
	if len(patterns) > 0 {
		return patterns
	}
	return cfg.Bugfix.Patterns
}

func detectAndApplyBugfixes(
	changeSets []git.CommitChangeSet,
	metrics map[string]*aggregation.FileMetrics,
	aggregator *aggregation.FileMetricsAggregator,
	patterns []string,
) (*bugfix.BugfixResult, error) {
	detector, err := bugfix.NewDetector(patterns)
	if err != nil {
		return nil, fmt.Errorf("invalid bug pattern: %w", err)
	}

	result := detector.Detect(changeSets)
	aggregation.ApplyBugfixCounts(metrics, aggregator, result.FileBugfixCounts)
	return result, nil
}
