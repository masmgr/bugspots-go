package bugfix

import (
	"testing"
	"time"

	"github.com/masmgr/bugspots-go/internal/git"
)

func TestNewDetector_ValidPatterns(t *testing.T) {
	patterns := []string{`\bfix(ed|es)?\b`, `\bbug\b`, `\bhotfix\b`}
	d, err := NewDetector(patterns)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(d.patterns) != 3 {
		t.Errorf("expected 3 compiled patterns, got %d", len(d.patterns))
	}
}

func TestNewDetector_InvalidPattern(t *testing.T) {
	patterns := []string{`[invalid`}
	_, err := NewDetector(patterns)
	if err == nil {
		t.Fatal("expected error for invalid pattern, got nil")
	}
}

func TestNewDetector_EmptyPatterns(t *testing.T) {
	d, err := NewDetector([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(d.patterns) != 0 {
		t.Errorf("expected 0 compiled patterns, got %d", len(d.patterns))
	}
}

func TestNewDetector_SkipsBlankPatterns(t *testing.T) {
	d, err := NewDetector([]string{"fix", "", "  ", "bug"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(d.patterns) != 2 {
		t.Errorf("expected 2 compiled patterns, got %d", len(d.patterns))
	}
}

func TestIsBugfix(t *testing.T) {
	d, err := NewDetector([]string{`\bfix(ed|es)?\b`, `\bbug\b`, `\bhotfix\b`})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tests := []struct {
		name    string
		message string
		want    bool
	}{
		{"matches fix", "fix: resolve null pointer", true},
		{"matches fixed", "fixed login issue", true},
		{"matches fixes", "fixes #123", true},
		{"matches bug", "bug in auth module", true},
		{"matches hotfix", "hotfix for production crash", true},
		{"case insensitive", "FIX: resolve issue", true},
		{"case insensitive mixed", "Fixed Login Issue", true},
		{"no match", "add new feature", false},
		{"no match refactor", "refactor: clean up code", false},
		{"partial word no match", "prefix fixation suffix", false},
		{"empty message", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := d.IsBugfix(tt.message)
			if got != tt.want {
				t.Errorf("IsBugfix(%q) = %v, want %v", tt.message, got, tt.want)
			}
		})
	}
}

func TestIsBugfix_NoPatterns(t *testing.T) {
	d, _ := NewDetector([]string{})
	if d.IsBugfix("fix something") {
		t.Error("expected false when no patterns configured")
	}
}

func makeChangeSets() []git.CommitChangeSet {
	now := time.Now()
	return []git.CommitChangeSet{
		{
			Commit: git.CommitInfo{
				SHA:     "aaa111",
				When:    now,
				Author:  git.AuthorInfo{Name: "Alice", Email: "alice@example.com"},
				Message: "fix: resolve null pointer in auth",
			},
			Changes: []git.FileChange{
				{Path: "auth/login.go", Kind: git.ChangeKindModified},
				{Path: "auth/session.go", Kind: git.ChangeKindModified},
			},
		},
		{
			Commit: git.CommitInfo{
				SHA:     "bbb222",
				When:    now,
				Author:  git.AuthorInfo{Name: "Bob", Email: "bob@example.com"},
				Message: "feat: add user profile page",
			},
			Changes: []git.FileChange{
				{Path: "user/profile.go", Kind: git.ChangeKindAdded},
			},
		},
		{
			Commit: git.CommitInfo{
				SHA:     "ccc333",
				When:    now,
				Author:  git.AuthorInfo{Name: "Charlie", Email: "charlie@example.com"},
				Message: "bug: incorrect validation logic",
			},
			Changes: []git.FileChange{
				{Path: "auth/login.go", Kind: git.ChangeKindModified},
				{Path: "validation/rules.go", Kind: git.ChangeKindModified},
				{Path: "old/file.go", Kind: git.ChangeKindDeleted},
			},
		},
	}
}

func TestDetect(t *testing.T) {
	d, err := NewDetector([]string{`\bfix\b`, `\bbug\b`})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result := d.Detect(makeChangeSets())

	// Should detect 2 bugfix commits (aaa111 and ccc333)
	if result.TotalBugfixes != 2 {
		t.Errorf("TotalBugfixes = %d, want 2", result.TotalBugfixes)
	}

	if _, ok := result.BugfixCommits["aaa111"]; !ok {
		t.Error("expected aaa111 to be a bugfix commit")
	}
	if _, ok := result.BugfixCommits["bbb222"]; ok {
		t.Error("expected bbb222 to NOT be a bugfix commit")
	}
	if _, ok := result.BugfixCommits["ccc333"]; !ok {
		t.Error("expected ccc333 to be a bugfix commit")
	}

	// auth/login.go should have count 2 (from aaa111 and ccc333)
	if result.FileBugfixCounts["auth/login.go"] != 2 {
		t.Errorf("FileBugfixCounts[auth/login.go] = %d, want 2", result.FileBugfixCounts["auth/login.go"])
	}

	// auth/session.go should have count 1 (from aaa111)
	if result.FileBugfixCounts["auth/session.go"] != 1 {
		t.Errorf("FileBugfixCounts[auth/session.go] = %d, want 1", result.FileBugfixCounts["auth/session.go"])
	}

	// validation/rules.go should have count 1 (from ccc333)
	if result.FileBugfixCounts["validation/rules.go"] != 1 {
		t.Errorf("FileBugfixCounts[validation/rules.go] = %d, want 1", result.FileBugfixCounts["validation/rules.go"])
	}

	// user/profile.go should NOT be in the map (not a bugfix)
	if result.FileBugfixCounts["user/profile.go"] != 0 {
		t.Errorf("FileBugfixCounts[user/profile.go] = %d, want 0", result.FileBugfixCounts["user/profile.go"])
	}

	// old/file.go should NOT be counted (deleted)
	if result.FileBugfixCounts["old/file.go"] != 0 {
		t.Errorf("FileBugfixCounts[old/file.go] = %d, want 0 (deleted files should be skipped)", result.FileBugfixCounts["old/file.go"])
	}
}

func TestDetect_NoPatterns(t *testing.T) {
	d, _ := NewDetector([]string{})
	result := d.Detect(makeChangeSets())

	if result.TotalBugfixes != 0 {
		t.Errorf("TotalBugfixes = %d, want 0", result.TotalBugfixes)
	}
	if len(result.BugfixCommits) != 0 {
		t.Errorf("BugfixCommits length = %d, want 0", len(result.BugfixCommits))
	}
	if len(result.FileBugfixCounts) != 0 {
		t.Errorf("FileBugfixCounts length = %d, want 0", len(result.FileBugfixCounts))
	}
}

func TestDetect_MultiplePatterns(t *testing.T) {
	// Only "hotfix" pattern, should match nothing in our test data
	d, _ := NewDetector([]string{`\bhotfix\b`})
	result := d.Detect(makeChangeSets())
	if result.TotalBugfixes != 0 {
		t.Errorf("TotalBugfixes = %d, want 0", result.TotalBugfixes)
	}

	// Add "fix" pattern, should match aaa111
	d, _ = NewDetector([]string{`\bhotfix\b`, `\bfix\b`})
	result = d.Detect(makeChangeSets())
	if result.TotalBugfixes != 1 {
		t.Errorf("TotalBugfixes = %d, want 1", result.TotalBugfixes)
	}

	// Add "bug" pattern, should match aaa111 and ccc333
	d, _ = NewDetector([]string{`\bhotfix\b`, `\bfix\b`, `\bbug\b`})
	result = d.Detect(makeChangeSets())
	if result.TotalBugfixes != 2 {
		t.Errorf("TotalBugfixes = %d, want 2", result.TotalBugfixes)
	}
}

func TestDetect_EmptyChangeSets(t *testing.T) {
	d, _ := NewDetector([]string{`\bfix\b`})
	result := d.Detect([]git.CommitChangeSet{})

	if result.TotalBugfixes != 0 {
		t.Errorf("TotalBugfixes = %d, want 0", result.TotalBugfixes)
	}
}
