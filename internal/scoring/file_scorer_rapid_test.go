package scoring

import (
	"fmt"
	"testing"
	"time"

	"pgregory.net/rapid"

	"github.com/masmgr/bugspots-go/config"
	"github.com/masmgr/bugspots-go/internal/aggregation"
)

// --- Generators ---

func genFileMetrics() *rapid.Generator[*aggregation.FileMetrics] {
	return rapid.Custom(func(t *rapid.T) *aggregation.FileMetrics {
		path := fmt.Sprintf("pkg%d/file%d.go",
			rapid.IntRange(0, 5).Draw(t, "pkg"),
			rapid.IntRange(0, 20).Draw(t, "file"))
		fm := aggregation.NewFileMetrics(path)
		fm.CommitCount = rapid.IntRange(1, 100).Draw(t, "commitCount")
		fm.AddedLines = rapid.IntRange(0, 5000).Draw(t, "added")
		fm.DeletedLines = rapid.IntRange(0, 5000).Draw(t, "deleted")
		fm.LastModifiedAt = time.Now().Add(-time.Duration(rapid.IntRange(0, 365).Draw(t, "daysAgo")) * 24 * time.Hour)
		fm.BurstScore = rapid.Float64Range(0, 1).Draw(t, "burst")
		fm.BugfixCount = rapid.IntRange(0, fm.CommitCount).Draw(t, "bugfix")
		fm.FileSize = rapid.IntRange(0, 10000).Draw(t, "size")

		// Add contributors (at least 1)
		contributorCount := rapid.IntRange(1, 5).Draw(t, "contributors")
		totalAssigned := 0
		for i := 0; i < contributorCount; i++ {
			email := fmt.Sprintf("dev%d@example.com", i)
			fm.Contributors[email] = struct{}{}
			commits := 1
			if i < contributorCount-1 {
				commits = rapid.IntRange(1, max(1, fm.CommitCount-totalAssigned-(contributorCount-i-1))).Draw(t, fmt.Sprintf("commits%d", i))
			} else {
				commits = fm.CommitCount - totalAssigned
			}
			if commits < 1 {
				commits = 1
			}
			fm.ContributorCommitCounts[email] = commits
			totalAssigned += commits
		}

		return fm
	})
}

func genFileMetricsMap() *rapid.Generator[map[string]*aggregation.FileMetrics] {
	return rapid.Custom(func(t *rapid.T) map[string]*aggregation.FileMetrics {
		count := rapid.IntRange(1, 20).Draw(t, "count")
		metrics := make(map[string]*aggregation.FileMetrics, count)
		for i := 0; i < count; i++ {
			fm := genFileMetrics().Draw(t, fmt.Sprintf("fm%d", i))
			// Ensure unique paths
			fm.Path = fmt.Sprintf("pkg%d/file%d.go", i/5, i)
			metrics[fm.Path] = fm
		}
		return metrics
	})
}

// --- Property Tests ---

func TestRapidFromMetrics_MinLeqMax(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		metrics := genFileMetricsMap().Draw(t, "metrics")

		ctx := FromMetrics(metrics)

		if ctx.CommitCount.Min > ctx.CommitCount.Max {
			t.Fatalf("CommitCount Min(%f) > Max(%f)", ctx.CommitCount.Min, ctx.CommitCount.Max)
		}
		if ctx.ChurnTotal.Min > ctx.ChurnTotal.Max {
			t.Fatalf("ChurnTotal Min(%f) > Max(%f)", ctx.ChurnTotal.Min, ctx.ChurnTotal.Max)
		}
		if ctx.BugfixCount.Min > ctx.BugfixCount.Max {
			t.Fatalf("BugfixCount Min(%f) > Max(%f)", ctx.BugfixCount.Min, ctx.BugfixCount.Max)
		}
		if ctx.FileSize.Min > ctx.FileSize.Max {
			t.Fatalf("FileSize Min(%f) > Max(%f)", ctx.FileSize.Min, ctx.FileSize.Max)
		}
	})
}

func TestRapidScoreAndRank_OutputBounds(t *testing.T) {
	scorer := NewFileScorer(config.DefaultConfig().Scoring)

	rapid.Check(t, func(t *rapid.T) {
		metrics := genFileMetricsMap().Draw(t, "metrics")
		now := time.Now()

		items := scorer.ScoreAndRank(metrics, false, now)

		for i, item := range items {
			if item.RiskScore < 0.0 || item.RiskScore > 1.0 {
				t.Fatalf("items[%d].RiskScore=%f, expected in [0,1]", i, item.RiskScore)
			}
		}
	})
}

func TestRapidScoreAndRank_SortedDescending(t *testing.T) {
	scorer := NewFileScorer(config.DefaultConfig().Scoring)

	rapid.Check(t, func(t *rapid.T) {
		metrics := genFileMetricsMap().Draw(t, "metrics")
		now := time.Now()

		items := scorer.ScoreAndRank(metrics, false, now)

		for i := 1; i < len(items); i++ {
			if items[i].RiskScore > items[i-1].RiskScore {
				t.Fatalf("Not sorted descending at index %d: %f > %f",
					i, items[i].RiskScore, items[i-1].RiskScore)
			}
		}
	})
}

func TestRapidScoreAndRank_ExplainBreakdown(t *testing.T) {
	scorer := NewFileScorer(config.DefaultConfig().Scoring)

	rapid.Check(t, func(t *rapid.T) {
		metrics := genFileMetricsMap().Draw(t, "metrics")
		now := time.Now()

		withExplain := scorer.ScoreAndRank(metrics, true, now)
		for i, item := range withExplain {
			if item.Breakdown == nil {
				t.Fatalf("items[%d].Breakdown is nil with explain=true", i)
			}
		}

		withoutExplain := scorer.ScoreAndRank(metrics, false, now)
		for i, item := range withoutExplain {
			if item.Breakdown != nil {
				t.Fatalf("items[%d].Breakdown is non-nil with explain=false", i)
			}
		}
	})
}

func TestRapidScoreAndRank_LengthPreserved(t *testing.T) {
	scorer := NewFileScorer(config.DefaultConfig().Scoring)

	rapid.Check(t, func(t *rapid.T) {
		metrics := genFileMetricsMap().Draw(t, "metrics")
		now := time.Now()

		items := scorer.ScoreAndRank(metrics, false, now)

		if len(items) != len(metrics) {
			t.Fatalf("len(items)=%d != len(metrics)=%d", len(items), len(metrics))
		}
	})
}
