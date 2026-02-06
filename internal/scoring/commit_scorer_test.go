package scoring

import (
	"testing"
	"time"

	"github.com/masmgr/bugspots-go/config"
	"github.com/masmgr/bugspots-go/internal/aggregation"
	"github.com/masmgr/bugspots-go/internal/git"
)

func TestCommitContextFromMetrics_Empty(t *testing.T) {
	ctx := CommitContextFromMetrics(nil)

	if ctx.FileCount.Min != 0 || ctx.FileCount.Max != 0 {
		t.Errorf("FileCount = {%f, %f}, expected {0, 0}", ctx.FileCount.Min, ctx.FileCount.Max)
	}
	if ctx.DirectoryCount.Min != 0 || ctx.DirectoryCount.Max != 0 {
		t.Errorf("DirectoryCount = {%f, %f}, expected {0, 0}", ctx.DirectoryCount.Min, ctx.DirectoryCount.Max)
	}
	if ctx.SubsystemCount.Min != 0 || ctx.SubsystemCount.Max != 0 {
		t.Errorf("SubsystemCount = {%f, %f}, expected {0, 0}", ctx.SubsystemCount.Min, ctx.SubsystemCount.Max)
	}
	if ctx.TotalChurn.Min != 0 || ctx.TotalChurn.Max != 0 {
		t.Errorf("TotalChurn = {%f, %f}, expected {0, 0}", ctx.TotalChurn.Min, ctx.TotalChurn.Max)
	}
}

func TestCommitContextFromMetrics_Multiple(t *testing.T) {
	metrics := []aggregation.CommitMetrics{
		{FileCount: 1, DirectoryCount: 1, SubsystemCount: 1, LinesAdded: 5, LinesDeleted: 5},
		{FileCount: 5, DirectoryCount: 3, SubsystemCount: 2, LinesAdded: 25, LinesDeleted: 25},
		{FileCount: 10, DirectoryCount: 5, SubsystemCount: 3, LinesAdded: 50, LinesDeleted: 50},
	}

	ctx := CommitContextFromMetrics(metrics)

	if ctx.FileCount.Min != 1 || ctx.FileCount.Max != 10 {
		t.Errorf("FileCount = {%f, %f}, expected {1, 10}", ctx.FileCount.Min, ctx.FileCount.Max)
	}
	if ctx.DirectoryCount.Min != 1 || ctx.DirectoryCount.Max != 5 {
		t.Errorf("DirectoryCount = {%f, %f}, expected {1, 5}", ctx.DirectoryCount.Min, ctx.DirectoryCount.Max)
	}
	if ctx.SubsystemCount.Min != 1 || ctx.SubsystemCount.Max != 3 {
		t.Errorf("SubsystemCount = {%f, %f}, expected {1, 3}", ctx.SubsystemCount.Min, ctx.SubsystemCount.Max)
	}
	if ctx.TotalChurn.Min != 10 || ctx.TotalChurn.Max != 100 {
		t.Errorf("TotalChurn = {%f, %f}, expected {10, 100}", ctx.TotalChurn.Min, ctx.TotalChurn.Max)
	}
}

func TestCommitScorer_ScoreAndRank_Empty(t *testing.T) {
	scorer := NewCommitScorer(config.DefaultConfig().CommitScoring)
	result := scorer.ScoreAndRank(nil, false)
	if result != nil {
		t.Errorf("ScoreAndRank(nil) = %v, expected nil", result)
	}
}

func TestCommitScorer_ScoreAndRank_Ordering(t *testing.T) {
	scorer := NewCommitScorer(config.DefaultConfig().CommitScoring)

	metrics := []aggregation.CommitMetrics{
		{
			SHA:            "small",
			When:           time.Now(),
			Author:         git.AuthorInfo{Name: "A", Email: "a@example.com"},
			Message:        "small change",
			FileCount:      1,
			DirectoryCount: 1,
			SubsystemCount: 1,
			LinesAdded:     1,
			LinesDeleted:   0,
			ChangeEntropy:  0.0,
		},
		{
			SHA:            "large",
			When:           time.Now(),
			Author:         git.AuthorInfo{Name: "B", Email: "b@example.com"},
			Message:        "large risky change",
			FileCount:      20,
			DirectoryCount: 10,
			SubsystemCount: 5,
			LinesAdded:     500,
			LinesDeleted:   200,
			ChangeEntropy:  0.9,
		},
	}

	items := scorer.ScoreAndRank(metrics, false)

	if len(items) != 2 {
		t.Fatalf("Expected 2 items, got %d", len(items))
	}

	// Large risky change should be first (higher score)
	if items[0].Metrics.SHA != "large" {
		t.Errorf("Expected 'large' commit first, got %q", items[0].Metrics.SHA)
	}
	if items[0].RiskScore <= items[1].RiskScore {
		t.Errorf("First item score %f should be > second item score %f",
			items[0].RiskScore, items[1].RiskScore)
	}
}

func TestCommitScorer_ScoreAndRank_ExplainBreakdown(t *testing.T) {
	scorer := NewCommitScorer(config.DefaultConfig().CommitScoring)

	metrics := []aggregation.CommitMetrics{
		{
			SHA:            "abc123",
			FileCount:      5,
			DirectoryCount: 3,
			SubsystemCount: 2,
			LinesAdded:     50,
			LinesDeleted:   20,
			ChangeEntropy:  0.5,
		},
	}

	// With explain=true
	items := scorer.ScoreAndRank(metrics, true)
	if len(items) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(items))
	}
	if items[0].Breakdown == nil {
		t.Error("Breakdown should not be nil when explain=true")
	}

	// With explain=false
	items = scorer.ScoreAndRank(metrics, false)
	if items[0].Breakdown != nil {
		t.Error("Breakdown should be nil when explain=false")
	}
}

func TestCommitScorer_ScoreAndRank_ScoreBounded(t *testing.T) {
	scorer := NewCommitScorer(config.DefaultConfig().CommitScoring)

	metrics := []aggregation.CommitMetrics{
		{FileCount: 1, LinesAdded: 1, ChangeEntropy: 0.0},
		{FileCount: 100, DirectoryCount: 50, SubsystemCount: 20, LinesAdded: 10000, LinesDeleted: 5000, ChangeEntropy: 1.0},
	}

	items := scorer.ScoreAndRank(metrics, false)
	for _, item := range items {
		if item.RiskScore < 0 || item.RiskScore > 1 {
			t.Errorf("Score %f is outside [0, 1] range", item.RiskScore)
		}
	}
}

func TestFilterByRiskLevel(t *testing.T) {
	items := []CommitRiskItem{
		{RiskScore: 0.9, RiskLevel: config.RiskLevelHigh},
		{RiskScore: 0.5, RiskLevel: config.RiskLevelMedium},
		{RiskScore: 0.2, RiskLevel: config.RiskLevelLow},
	}

	tests := []struct {
		name     string
		level    config.RiskLevel
		expected int
	}{
		{name: "Filter high only", level: config.RiskLevelHigh, expected: 1},
		{name: "Filter medium+", level: config.RiskLevelMedium, expected: 2},
		{name: "Filter low (all)", level: config.RiskLevelLow, expected: 3},
		{name: "Empty filter", level: "", expected: 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterByRiskLevel(items, tt.level)
			if len(result) != tt.expected {
				t.Errorf("FilterByRiskLevel(%q) returned %d items, expected %d",
					tt.level, len(result), tt.expected)
			}
		})
	}

	// Empty input
	t.Run("Empty input", func(t *testing.T) {
		result := FilterByRiskLevel(nil, config.RiskLevelHigh)
		if len(result) != 0 {
			t.Errorf("FilterByRiskLevel(nil) returned %d items, expected 0", len(result))
		}
	})
}
