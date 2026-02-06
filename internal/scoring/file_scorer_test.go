package scoring

import (
	"testing"
	"time"

	"github.com/masmgr/bugspots-go/config"
	"github.com/masmgr/bugspots-go/internal/aggregation"
)

func TestFromMetrics_Empty(t *testing.T) {
	ctx := FromMetrics(map[string]*aggregation.FileMetrics{})

	if ctx.CommitCount.Min != 0 || ctx.CommitCount.Max != 0 {
		t.Errorf("CommitCount = {%f, %f}, expected {0, 0}", ctx.CommitCount.Min, ctx.CommitCount.Max)
	}
	if ctx.ChurnTotal.Min != 0 || ctx.ChurnTotal.Max != 0 {
		t.Errorf("ChurnTotal = {%f, %f}, expected {0, 0}", ctx.ChurnTotal.Min, ctx.ChurnTotal.Max)
	}
}

func TestFromMetrics_Multiple(t *testing.T) {
	metrics := map[string]*aggregation.FileMetrics{
		"file1.go": {CommitCount: 2, AddedLines: 10, DeletedLines: 5},
		"file2.go": {CommitCount: 10, AddedLines: 100, DeletedLines: 50},
		"file3.go": {CommitCount: 5, AddedLines: 30, DeletedLines: 20},
	}

	ctx := FromMetrics(metrics)

	if ctx.CommitCount.Min != 2 || ctx.CommitCount.Max != 10 {
		t.Errorf("CommitCount = {%f, %f}, expected {2, 10}", ctx.CommitCount.Min, ctx.CommitCount.Max)
	}
	if ctx.ChurnTotal.Min != 15 || ctx.ChurnTotal.Max != 150 {
		t.Errorf("ChurnTotal = {%f, %f}, expected {15, 150}", ctx.ChurnTotal.Min, ctx.ChurnTotal.Max)
	}
}

func TestFileScorer_ScoreAndRank_Empty(t *testing.T) {
	scorer := NewFileScorer(config.DefaultConfig().Scoring)
	result := scorer.ScoreAndRank(nil, false, time.Now())
	if result != nil {
		t.Errorf("ScoreAndRank(nil) = %v, expected nil", result)
	}
}

func TestFileScorer_ScoreAndRank_Ordering(t *testing.T) {
	scorer := NewFileScorer(config.DefaultConfig().Scoring)
	now := time.Now()

	metrics := map[string]*aggregation.FileMetrics{
		"hot.go": {
			CommitCount:             20,
			AddedLines:              500,
			DeletedLines:            200,
			LastModifiedAt:          now.Add(-24 * time.Hour),
			Contributors:            map[string]struct{}{"a": {}, "b": {}, "c": {}},
			ContributorCommitCounts: map[string]int{"a": 10, "b": 6, "c": 4},
			BurstScore:              0.8,
			CommitTimes:             []time.Time{},
		},
		"cold.go": {
			CommitCount:             1,
			AddedLines:              5,
			DeletedLines:            0,
			LastModifiedAt:          now.Add(-365 * 24 * time.Hour),
			Contributors:            map[string]struct{}{"a": {}},
			ContributorCommitCounts: map[string]int{"a": 1},
			BurstScore:              0.1,
			CommitTimes:             []time.Time{},
		},
	}

	items := scorer.ScoreAndRank(metrics, false, now)

	if len(items) != 2 {
		t.Fatalf("Expected 2 items, got %d", len(items))
	}

	if items[0].Path != "hot.go" {
		t.Errorf("Expected 'hot.go' first, got %q", items[0].Path)
	}
	if items[0].RiskScore <= items[1].RiskScore {
		t.Errorf("First item score %f should be > second item score %f",
			items[0].RiskScore, items[1].RiskScore)
	}
}

func TestFileScorer_ScoreAndRank_ExplainBreakdown(t *testing.T) {
	scorer := NewFileScorer(config.DefaultConfig().Scoring)
	now := time.Now()

	metrics := map[string]*aggregation.FileMetrics{
		"test.go": {
			CommitCount:             5,
			AddedLines:              50,
			DeletedLines:            20,
			LastModifiedAt:          now.Add(-7 * 24 * time.Hour),
			Contributors:            map[string]struct{}{"a": {}},
			ContributorCommitCounts: map[string]int{"a": 5},
			BurstScore:              0.5,
			CommitTimes:             []time.Time{},
		},
	}

	// With explain=true
	items := scorer.ScoreAndRank(metrics, true, now)
	if len(items) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(items))
	}
	if items[0].Breakdown == nil {
		t.Error("Breakdown should not be nil when explain=true")
	}

	// With explain=false
	items = scorer.ScoreAndRank(metrics, false, now)
	if items[0].Breakdown != nil {
		t.Error("Breakdown should be nil when explain=false")
	}
}

func TestFileScorer_ScoreAndRank_RecencyEffect(t *testing.T) {
	scorer := NewFileScorer(config.DefaultConfig().Scoring)
	now := time.Now()

	metrics := map[string]*aggregation.FileMetrics{
		"recent.go": {
			CommitCount:             5,
			AddedLines:              50,
			DeletedLines:            20,
			LastModifiedAt:          now.Add(-1 * 24 * time.Hour),
			Contributors:            map[string]struct{}{"a": {}},
			ContributorCommitCounts: map[string]int{"a": 5},
			BurstScore:              0.5,
			CommitTimes:             []time.Time{},
		},
		"old.go": {
			CommitCount:             5,
			AddedLines:              50,
			DeletedLines:            20,
			LastModifiedAt:          now.Add(-180 * 24 * time.Hour),
			Contributors:            map[string]struct{}{"a": {}},
			ContributorCommitCounts: map[string]int{"a": 5},
			BurstScore:              0.5,
			CommitTimes:             []time.Time{},
		},
	}

	items := scorer.ScoreAndRank(metrics, true, now)

	var recentScore, oldScore float64
	for _, item := range items {
		if item.Path == "recent.go" {
			recentScore = item.RiskScore
		}
		if item.Path == "old.go" {
			oldScore = item.RiskScore
		}
	}

	if recentScore <= oldScore {
		t.Errorf("Recent file score %f should be > old file score %f", recentScore, oldScore)
	}
}
