package aggregation

import (
	"math"
	"testing"
	"time"

	"github.com/masmgr/bugspots-go/internal/git"
)

func TestNewFileMetrics(t *testing.T) {
	fm := NewFileMetrics("test/file.go")

	if fm.Path != "test/file.go" {
		t.Errorf("Path = %q, expected %q", fm.Path, "test/file.go")
	}
	if fm.CommitCount != 0 {
		t.Errorf("CommitCount = %d, expected 0", fm.CommitCount)
	}
	if fm.AddedLines != 0 {
		t.Errorf("AddedLines = %d, expected 0", fm.AddedLines)
	}
	if fm.DeletedLines != 0 {
		t.Errorf("DeletedLines = %d, expected 0", fm.DeletedLines)
	}
	if fm.Contributors == nil {
		t.Error("Contributors map is nil, expected initialized")
	}
	if fm.ContributorCommitCounts == nil {
		t.Error("ContributorCommitCounts map is nil, expected initialized")
	}
	if fm.CommitTimes == nil {
		t.Error("CommitTimes slice is nil, expected initialized")
	}
	if len(fm.CommitTimes) != 0 {
		t.Errorf("CommitTimes length = %d, expected 0", len(fm.CommitTimes))
	}
}

func TestFileMetrics_ChurnTotal(t *testing.T) {
	tests := []struct {
		name     string
		added    int
		deleted  int
		expected int
	}{
		{name: "Both set", added: 10, deleted: 5, expected: 15},
		{name: "Zero", added: 0, deleted: 0, expected: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm := &FileMetrics{AddedLines: tt.added, DeletedLines: tt.deleted}
			result := fm.ChurnTotal()
			if result != tt.expected {
				t.Errorf("ChurnTotal() = %d, expected %d", result, tt.expected)
			}
		})
	}
}

func TestFileMetrics_ContributorCount(t *testing.T) {
	fm := NewFileMetrics("test.go")
	if fm.ContributorCount() != 0 {
		t.Errorf("ContributorCount() = %d, expected 0", fm.ContributorCount())
	}

	fm.Contributors["alice@example.com"] = struct{}{}
	fm.Contributors["bob@example.com"] = struct{}{}
	fm.Contributors["charlie@example.com"] = struct{}{}

	if fm.ContributorCount() != 3 {
		t.Errorf("ContributorCount() = %d, expected 3", fm.ContributorCount())
	}
}

func TestFileMetrics_OwnershipRatio(t *testing.T) {
	t.Run("Single contributor", func(t *testing.T) {
		fm := NewFileMetrics("test.go")
		fm.CommitCount = 5
		fm.ContributorCommitCounts["alice@example.com"] = 5
		result := fm.OwnershipRatio()
		if result != 1.0 {
			t.Errorf("OwnershipRatio() = %f, expected 1.0", result)
		}
	})

	t.Run("Two contributors", func(t *testing.T) {
		fm := NewFileMetrics("test.go")
		fm.CommitCount = 5
		fm.ContributorCommitCounts["alice@example.com"] = 3
		fm.ContributorCommitCounts["bob@example.com"] = 2
		result := fm.OwnershipRatio()
		if math.Abs(result-0.6) > 0.001 {
			t.Errorf("OwnershipRatio() = %f, expected 0.6", result)
		}
	})

	t.Run("Three contributors", func(t *testing.T) {
		fm := NewFileMetrics("test.go")
		fm.CommitCount = 10
		fm.ContributorCommitCounts["alice@example.com"] = 5
		fm.ContributorCommitCounts["bob@example.com"] = 3
		fm.ContributorCommitCounts["charlie@example.com"] = 2
		result := fm.OwnershipRatio()
		if math.Abs(result-0.5) > 0.001 {
			t.Errorf("OwnershipRatio() = %f, expected 0.5", result)
		}
	})

	t.Run("No commits", func(t *testing.T) {
		fm := NewFileMetrics("test.go")
		result := fm.OwnershipRatio()
		if result != 1.0 {
			t.Errorf("OwnershipRatio() = %f, expected 1.0 for no commits", result)
		}
	})
}

func TestFileMetrics_OwnershipRatio_Caching(t *testing.T) {
	fm := NewFileMetrics("test.go")
	fm.CommitCount = 2
	fm.ContributorCommitCounts["alice@example.com"] = 2

	// First call should compute and cache
	result1 := fm.OwnershipRatio()
	// Second call should use cache
	result2 := fm.OwnershipRatio()

	if result1 != result2 {
		t.Errorf("Cached result %f != first result %f", result2, result1)
	}

	// AddCommit should invalidate cache
	commit := git.CommitInfo{
		SHA:     "abc123",
		When:    time.Now(),
		Author:  git.AuthorInfo{Name: "Bob", Email: "bob@example.com"},
		Message: "fix",
	}
	change := git.FileChange{Path: "test.go", LinesAdded: 1, LinesDeleted: 0}
	fm.AddCommit(commit, change, false)

	result3 := fm.OwnershipRatio()
	// Now we have alice:2, bob:1, total:3 → ratio = 2/3
	if math.Abs(result3-2.0/3.0) > 0.001 {
		t.Errorf("OwnershipRatio() after AddCommit = %f, expected %f", result3, 2.0/3.0)
	}
}

func TestFileMetrics_AddCommit(t *testing.T) {
	fm := NewFileMetrics("test.go")

	t1 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)

	commit1 := git.CommitInfo{
		SHA:     "abc123",
		When:    t1,
		Author:  git.AuthorInfo{Name: "Alice", Email: "alice@example.com"},
		Message: "first commit",
	}
	change1 := git.FileChange{Path: "test.go", LinesAdded: 10, LinesDeleted: 5}
	fm.AddCommit(commit1, change1, true)

	if fm.CommitCount != 1 {
		t.Errorf("CommitCount = %d, expected 1", fm.CommitCount)
	}
	if fm.AddedLines != 10 {
		t.Errorf("AddedLines = %d, expected 10", fm.AddedLines)
	}
	if fm.DeletedLines != 5 {
		t.Errorf("DeletedLines = %d, expected 5", fm.DeletedLines)
	}
	if !fm.LastModifiedAt.Equal(t1) {
		t.Errorf("LastModifiedAt = %v, expected %v", fm.LastModifiedAt, t1)
	}
	if len(fm.CommitTimes) != 1 {
		t.Errorf("CommitTimes length = %d, expected 1", len(fm.CommitTimes))
	}

	// Second commit with later date
	commit2 := git.CommitInfo{
		SHA:     "def456",
		When:    t2,
		Author:  git.AuthorInfo{Name: "Bob", Email: "bob@example.com"},
		Message: "second commit",
	}
	change2 := git.FileChange{Path: "test.go", LinesAdded: 3, LinesDeleted: 2}
	fm.AddCommit(commit2, change2, true)

	if fm.CommitCount != 2 {
		t.Errorf("CommitCount = %d, expected 2", fm.CommitCount)
	}
	if fm.AddedLines != 13 {
		t.Errorf("AddedLines = %d, expected 13", fm.AddedLines)
	}
	if fm.DeletedLines != 7 {
		t.Errorf("DeletedLines = %d, expected 7", fm.DeletedLines)
	}
	if !fm.LastModifiedAt.Equal(t2) {
		t.Errorf("LastModifiedAt = %v, expected %v (should be most recent)", fm.LastModifiedAt, t2)
	}
	if fm.ContributorCount() != 2 {
		t.Errorf("ContributorCount() = %d, expected 2", fm.ContributorCount())
	}
	if len(fm.CommitTimes) != 2 {
		t.Errorf("CommitTimes length = %d, expected 2", len(fm.CommitTimes))
	}
}

func TestFileMetrics_AddCommit_CommitTimesDisabled(t *testing.T) {
	fm := NewFileMetrics("test.go")
	commit := git.CommitInfo{
		SHA:     "abc123",
		When:    time.Now(),
		Author:  git.AuthorInfo{Name: "Alice", Email: "alice@example.com"},
		Message: "commit",
	}
	change := git.FileChange{Path: "test.go", LinesAdded: 1, LinesDeleted: 0}

	fm.AddCommit(commit, change, false)

	if len(fm.CommitTimes) != 0 {
		t.Errorf("CommitTimes length = %d, expected 0 when collectCommitTimes=false", len(fm.CommitTimes))
	}
}

func TestFileMetricsAggregator_Process(t *testing.T) {
	agg := NewFileMetricsAggregator()

	changeSets := []git.CommitChangeSet{
		{
			Commit: git.CommitInfo{
				SHA:     "abc123",
				When:    time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				Author:  git.AuthorInfo{Name: "Alice", Email: "alice@example.com"},
				Message: "first",
			},
			Changes: []git.FileChange{
				{Path: "file1.go", Kind: git.ChangeKindModified, LinesAdded: 10, LinesDeleted: 5},
				{Path: "file2.go", Kind: git.ChangeKindModified, LinesAdded: 3, LinesDeleted: 1},
			},
		},
		{
			Commit: git.CommitInfo{
				SHA:     "def456",
				When:    time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
				Author:  git.AuthorInfo{Name: "Bob", Email: "bob@example.com"},
				Message: "second",
			},
			Changes: []git.FileChange{
				{Path: "file1.go", Kind: git.ChangeKindModified, LinesAdded: 5, LinesDeleted: 2},
			},
		},
	}

	metrics := agg.Process(changeSets)

	if len(metrics) != 2 {
		t.Fatalf("Expected 2 files in metrics, got %d", len(metrics))
	}

	file1 := metrics["file1.go"]
	if file1 == nil {
		t.Fatal("file1.go not found in metrics")
	}
	if file1.CommitCount != 2 {
		t.Errorf("file1.go CommitCount = %d, expected 2", file1.CommitCount)
	}
	if file1.AddedLines != 15 {
		t.Errorf("file1.go AddedLines = %d, expected 15", file1.AddedLines)
	}

	file2 := metrics["file2.go"]
	if file2 == nil {
		t.Fatal("file2.go not found in metrics")
	}
	if file2.CommitCount != 1 {
		t.Errorf("file2.go CommitCount = %d, expected 1", file2.CommitCount)
	}
}

func TestFileMetricsAggregator_Process_DeletedFiles(t *testing.T) {
	agg := NewFileMetricsAggregator()

	changeSets := []git.CommitChangeSet{
		{
			Commit: git.CommitInfo{
				SHA:     "abc123",
				When:    time.Now(),
				Author:  git.AuthorInfo{Name: "Alice", Email: "alice@example.com"},
				Message: "delete file",
			},
			Changes: []git.FileChange{
				{Path: "deleted.go", Kind: git.ChangeKindDeleted, LinesAdded: 0, LinesDeleted: 50},
				{Path: "kept.go", Kind: git.ChangeKindModified, LinesAdded: 10, LinesDeleted: 0},
			},
		},
	}

	metrics := agg.Process(changeSets)

	if _, exists := metrics["deleted.go"]; exists {
		t.Error("Deleted file should not appear in metrics")
	}
	if _, exists := metrics["kept.go"]; !exists {
		t.Error("Modified file should appear in metrics")
	}
}

func TestFileMetricsAggregator_Process_Renames(t *testing.T) {
	agg := NewFileMetricsAggregator()

	changeSets := []git.CommitChangeSet{
		{
			Commit: git.CommitInfo{
				SHA:     "abc123",
				When:    time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				Author:  git.AuthorInfo{Name: "Alice", Email: "alice@example.com"},
				Message: "first",
			},
			Changes: []git.FileChange{
				{Path: "old.go", Kind: git.ChangeKindModified, LinesAdded: 10, LinesDeleted: 5},
			},
		},
		{
			Commit: git.CommitInfo{
				SHA:     "def456",
				When:    time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
				Author:  git.AuthorInfo{Name: "Bob", Email: "bob@example.com"},
				Message: "rename",
			},
			Changes: []git.FileChange{
				{Path: "new.go", OldPath: "old.go", Kind: git.ChangeKindRenamed, LinesAdded: 2, LinesDeleted: 1},
			},
		},
	}

	metrics := agg.Process(changeSets)

	if _, exists := metrics["old.go"]; exists {
		t.Error("Old path should be removed after rename")
	}

	newMetrics := metrics["new.go"]
	if newMetrics == nil {
		t.Fatal("new.go not found in metrics after rename")
	}

	// Should have merged metrics: commit from old.go + rename commit
	if newMetrics.CommitCount != 2 {
		t.Errorf("new.go CommitCount = %d, expected 2 (merged)", newMetrics.CommitCount)
	}
}

func TestFileMetricsAggregator_Process_Renames_ReverseOrder(t *testing.T) {
	agg := NewFileMetricsAggregator()

	// Simulate history being processed newest-first (rename first, then older old-path changes).
	changeSets := []git.CommitChangeSet{
		{
			Commit: git.CommitInfo{
				SHA:     "def456",
				When:    time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
				Author:  git.AuthorInfo{Name: "Bob", Email: "bob@example.com"},
				Message: "rename",
			},
			Changes: []git.FileChange{
				{Path: "new.go", OldPath: "old.go", Kind: git.ChangeKindRenamed, LinesAdded: 2, LinesDeleted: 1},
			},
		},
		{
			Commit: git.CommitInfo{
				SHA:     "abc123",
				When:    time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				Author:  git.AuthorInfo{Name: "Alice", Email: "alice@example.com"},
				Message: "older change",
			},
			Changes: []git.FileChange{
				{Path: "old.go", Kind: git.ChangeKindModified, LinesAdded: 10, LinesDeleted: 5},
			},
		},
	}

	metrics := agg.Process(changeSets)

	if _, exists := metrics["old.go"]; exists {
		t.Error("Old path should not appear in metrics after rename aliasing")
	}

	newMetrics := metrics["new.go"]
	if newMetrics == nil {
		t.Fatal("new.go not found in metrics after reverse-order rename")
	}

	if newMetrics.CommitCount != 2 {
		t.Errorf("new.go CommitCount = %d, expected 2 (merged)", newMetrics.CommitCount)
	}
	if newMetrics.AddedLines != 12 {
		t.Errorf("new.go AddedLines = %d, expected 12 (merged)", newMetrics.AddedLines)
	}
	if newMetrics.DeletedLines != 6 {
		t.Errorf("new.go DeletedLines = %d, expected 6 (merged)", newMetrics.DeletedLines)
	}
}
