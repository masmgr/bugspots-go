package scoring

import (
	"fmt"
	"testing"
	"time"

	"github.com/masmgr/bugspots-go/config"
	"github.com/masmgr/bugspots-go/internal/aggregation"
	"github.com/masmgr/bugspots-go/internal/git"
	"pgregory.net/rapid"
)

// --- Generators ---

func genCommitMetrics() *rapid.Generator[aggregation.CommitMetrics] {
	return rapid.Custom(func(t *rapid.T) aggregation.CommitMetrics {
		return aggregation.CommitMetrics{
			SHA:            fmt.Sprintf("sha%d", rapid.IntRange(0, 100000).Draw(t, "sha")),
			When:           time.Now(),
			Author:         git.AuthorInfo{Name: "Test", Email: "test@example.com"},
			Message:        "test commit",
			FileCount:      rapid.IntRange(1, 50).Draw(t, "files"),
			DirectoryCount: rapid.IntRange(1, 20).Draw(t, "dirs"),
			SubsystemCount: rapid.IntRange(1, 10).Draw(t, "subsystems"),
			LinesAdded:     rapid.IntRange(0, 5000).Draw(t, "added"),
			LinesDeleted:   rapid.IntRange(0, 5000).Draw(t, "deleted"),
			ChangeEntropy:  rapid.Float64Range(0, 1).Draw(t, "entropy"),
		}
	})
}

func genCommitMetricsSlice() *rapid.Generator[[]aggregation.CommitMetrics] {
	return rapid.Custom(func(t *rapid.T) []aggregation.CommitMetrics {
		count := rapid.IntRange(1, 20).Draw(t, "count")
		metrics := make([]aggregation.CommitMetrics, count)
		for i := 0; i < count; i++ {
			metrics[i] = genCommitMetrics().Draw(t, fmt.Sprintf("cm%d", i))
		}
		return metrics
	})
}

// --- Property Tests ---

func TestRapidCommitContext_MinLeqMax(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		metrics := genCommitMetricsSlice().Draw(t, "metrics")

		ctx := CommitContextFromMetrics(metrics)

		if ctx.FileCount.Min > ctx.FileCount.Max {
			t.Fatalf("FileCount Min(%f) > Max(%f)", ctx.FileCount.Min, ctx.FileCount.Max)
		}
		if ctx.DirectoryCount.Min > ctx.DirectoryCount.Max {
			t.Fatalf("DirectoryCount Min(%f) > Max(%f)", ctx.DirectoryCount.Min, ctx.DirectoryCount.Max)
		}
		if ctx.SubsystemCount.Min > ctx.SubsystemCount.Max {
			t.Fatalf("SubsystemCount Min(%f) > Max(%f)", ctx.SubsystemCount.Min, ctx.SubsystemCount.Max)
		}
		if ctx.TotalChurn.Min > ctx.TotalChurn.Max {
			t.Fatalf("TotalChurn Min(%f) > Max(%f)", ctx.TotalChurn.Min, ctx.TotalChurn.Max)
		}
	})
}

func TestRapidCommitScoreAndRank_OutputBounds(t *testing.T) {
	scorer := NewCommitScorer(config.DefaultConfig().CommitScoring)

	rapid.Check(t, func(t *rapid.T) {
		metrics := genCommitMetricsSlice().Draw(t, "metrics")

		items := scorer.ScoreAndRank(metrics, false)

		for i, item := range items {
			if item.RiskScore < 0.0 || item.RiskScore > 1.0 {
				t.Fatalf("items[%d].RiskScore=%f, expected in [0,1]", i, item.RiskScore)
			}
		}
	})
}

func TestRapidCommitScoreAndRank_SortedDescending(t *testing.T) {
	scorer := NewCommitScorer(config.DefaultConfig().CommitScoring)

	rapid.Check(t, func(t *rapid.T) {
		metrics := genCommitMetricsSlice().Draw(t, "metrics")

		items := scorer.ScoreAndRank(metrics, false)

		for i := 1; i < len(items); i++ {
			if items[i].RiskScore > items[i-1].RiskScore {
				t.Fatalf("Not sorted descending at index %d: %f > %f",
					i, items[i].RiskScore, items[i-1].RiskScore)
			}
		}
	})
}

func TestRapidCommitScoreAndRank_RiskLevelConsistent(t *testing.T) {
	cfg := config.DefaultConfig().CommitScoring
	scorer := NewCommitScorer(cfg)

	rapid.Check(t, func(t *rapid.T) {
		metrics := genCommitMetricsSlice().Draw(t, "metrics")

		items := scorer.ScoreAndRank(metrics, false)

		for i, item := range items {
			expected := cfg.Thresholds.Classify(item.RiskScore)
			if item.RiskLevel != expected {
				t.Fatalf("items[%d]: RiskLevel=%q but Classify(%f)=%q",
					i, item.RiskLevel, item.RiskScore, expected)
			}
		}
	})
}

func TestRapidFilterByRiskLevel_Subset(t *testing.T) {
	scorer := NewCommitScorer(config.DefaultConfig().CommitScoring)

	rapid.Check(t, func(t *rapid.T) {
		metrics := genCommitMetricsSlice().Draw(t, "metrics")
		items := scorer.ScoreAndRank(metrics, false)

		levels := []config.RiskLevel{
			config.RiskLevelHigh,
			config.RiskLevelMedium,
			config.RiskLevelLow,
			"",
		}
		level := levels[rapid.IntRange(0, len(levels)-1).Draw(t, "levelIdx")]

		filtered := FilterByRiskLevel(items, level)

		if len(filtered) > len(items) {
			t.Fatalf("FilterByRiskLevel returned %d items, more than original %d",
				len(filtered), len(items))
		}
	})
}

func TestRapidFilterByRiskLevel_OrderPreserved(t *testing.T) {
	scorer := NewCommitScorer(config.DefaultConfig().CommitScoring)

	rapid.Check(t, func(t *rapid.T) {
		metrics := genCommitMetricsSlice().Draw(t, "metrics")
		items := scorer.ScoreAndRank(metrics, false)

		levels := []config.RiskLevel{
			config.RiskLevelHigh,
			config.RiskLevelMedium,
			config.RiskLevelLow,
		}
		level := levels[rapid.IntRange(0, len(levels)-1).Draw(t, "levelIdx")]

		filtered := FilterByRiskLevel(items, level)

		// Verify filtered items appear in same relative order
		filterIdx := 0
		for _, item := range items {
			if filterIdx >= len(filtered) {
				break
			}
			if item.Metrics.SHA == filtered[filterIdx].Metrics.SHA {
				filterIdx++
			}
		}
		if filterIdx != len(filtered) {
			t.Fatalf("Filtered items are not in original order")
		}
	})
}
