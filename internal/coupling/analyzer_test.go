package coupling

import (
	"math"
	"testing"
	"time"

	"github.com/masmgr/bugspots-go/config"
	"github.com/masmgr/bugspots-go/internal/git"
)

func TestNewFilePair_ConsistentOrdering(t *testing.T) {
	tests := []struct {
		name      string
		a         string
		b         string
		expectedA string
		expectedB string
	}{
		{name: "Already ordered", a: "aaa.go", b: "zzz.go", expectedA: "aaa.go", expectedB: "zzz.go"},
		{name: "Reversed input", a: "zzz.go", b: "aaa.go", expectedA: "aaa.go", expectedB: "zzz.go"},
		{name: "Same file", a: "file.go", b: "file.go", expectedA: "file.go", expectedB: "file.go"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pair := NewFilePair(tt.a, tt.b)
			if pair.FileA != tt.expectedA || pair.FileB != tt.expectedB {
				t.Errorf("NewFilePair(%q, %q) = {%q, %q}, expected {%q, %q}",
					tt.a, tt.b, pair.FileA, pair.FileB, tt.expectedA, tt.expectedB)
			}
		})
	}
}

func TestNewFilePair_Symmetry(t *testing.T) {
	pairs := [][2]string{
		{"file1.go", "file2.go"},
		{"src/main.go", "lib/utils.go"},
		{"AAA.go", "bbb.go"},
	}

	for _, p := range pairs {
		pair1 := NewFilePair(p[0], p[1])
		pair2 := NewFilePair(p[1], p[0])
		if pair1 != pair2 {
			t.Errorf("NewFilePair(%q,%q) = %v != NewFilePair(%q,%q) = %v",
				p[0], p[1], pair1, p[1], p[0], pair2)
		}
	}
}

func makeChangeSet(sha string, files ...string) git.CommitChangeSet {
	changes := make([]git.FileChange, len(files))
	for i, f := range files {
		changes[i] = git.FileChange{Path: f, Kind: git.ChangeKindModified, LinesAdded: 1, LinesDeleted: 0}
	}
	return git.CommitChangeSet{
		Commit: git.CommitInfo{
			SHA:     sha,
			When:    time.Now(),
			Author:  git.AuthorInfo{Name: "Test", Email: "test@example.com"},
			Message: "test commit",
		},
		Changes: changes,
	}
}

func defaultCouplingConfig() config.CouplingConfig {
	return config.CouplingConfig{
		MinCoCommits:        1,
		MinJaccardThreshold: 0.0,
		MaxFilesPerCommit:   50,
		TopPairs:            50,
	}
}

func TestAnalyzer_Analyze_Empty(t *testing.T) {
	analyzer := NewAnalyzer(defaultCouplingConfig())
	result := analyzer.Analyze(nil)

	if result.TotalCommits != 0 {
		t.Errorf("TotalCommits = %d, expected 0", result.TotalCommits)
	}
	if len(result.Couplings) != 0 {
		t.Errorf("Couplings count = %d, expected 0", len(result.Couplings))
	}
}

func TestAnalyzer_Analyze_SingleFileCommit(t *testing.T) {
	analyzer := NewAnalyzer(defaultCouplingConfig())
	changeSets := []git.CommitChangeSet{
		makeChangeSet("abc", "file1.go"),
	}

	result := analyzer.Analyze(changeSets)

	if len(result.Couplings) != 0 {
		t.Errorf("Couplings count = %d, expected 0 for single-file commit", len(result.Couplings))
	}
}

func TestAnalyzer_Analyze_PerfectCoupling(t *testing.T) {
	analyzer := NewAnalyzer(defaultCouplingConfig())

	// 5 commits each touching a.go and b.go
	changeSets := make([]git.CommitChangeSet, 5)
	for i := 0; i < 5; i++ {
		changeSets[i] = makeChangeSet("sha"+string(rune('0'+i)), "a.go", "b.go")
	}

	result := analyzer.Analyze(changeSets)

	if len(result.Couplings) != 1 {
		t.Fatalf("Expected 1 coupling, got %d", len(result.Couplings))
	}

	coupling := result.Couplings[0]
	if coupling.CoCommitCount != 5 {
		t.Errorf("CoCommitCount = %d, expected 5", coupling.CoCommitCount)
	}
	if math.Abs(coupling.JaccardCoefficient-1.0) > 0.001 {
		t.Errorf("JaccardCoefficient = %f, expected 1.0", coupling.JaccardCoefficient)
	}
	if math.Abs(coupling.Confidence-1.0) > 0.001 {
		t.Errorf("Confidence = %f, expected 1.0", coupling.Confidence)
	}
}

func TestAnalyzer_Analyze_PartialCoupling(t *testing.T) {
	cfg := defaultCouplingConfig()
	analyzer := NewAnalyzer(cfg)

	// FileA in 10 commits, FileB in 6 commits, together in 4 commits
	var changeSets []git.CommitChangeSet

	// 4 commits with both files
	for i := 0; i < 4; i++ {
		changeSets = append(changeSets, makeChangeSet("both"+string(rune('0'+i)), "a.go", "b.go"))
	}
	// 6 commits with only A
	for i := 0; i < 6; i++ {
		changeSets = append(changeSets, makeChangeSet("onlyA"+string(rune('0'+i)), "a.go", "other.go"))
	}
	// 2 commits with only B
	for i := 0; i < 2; i++ {
		changeSets = append(changeSets, makeChangeSet("onlyB"+string(rune('0'+i)), "b.go", "other2.go"))
	}

	result := analyzer.Analyze(changeSets)

	// Find the a.go/b.go coupling
	var found *ChangeCoupling
	for i := range result.Couplings {
		c := &result.Couplings[i]
		if (c.FileA == "a.go" && c.FileB == "b.go") || (c.FileA == "b.go" && c.FileB == "a.go") {
			found = c
			break
		}
	}

	if found == nil {
		t.Fatal("Expected to find a.go/b.go coupling")
	}

	if found.CoCommitCount != 4 {
		t.Errorf("CoCommitCount = %d, expected 4", found.CoCommitCount)
	}

	// Jaccard = 4 / (10 + 6 - 4) = 4/12 = 0.333
	expectedJaccard := 4.0 / 12.0
	if math.Abs(found.JaccardCoefficient-expectedJaccard) > 0.01 {
		t.Errorf("JaccardCoefficient = %f, expected %f", found.JaccardCoefficient, expectedJaccard)
	}
}

func TestAnalyzer_Analyze_MinCoCommitsFilter(t *testing.T) {
	cfg := defaultCouplingConfig()
	cfg.MinCoCommits = 3

	analyzer := NewAnalyzer(cfg)

	// Only 2 co-commits (below threshold)
	changeSets := []git.CommitChangeSet{
		makeChangeSet("1", "a.go", "b.go"),
		makeChangeSet("2", "a.go", "b.go"),
	}

	result := analyzer.Analyze(changeSets)

	if len(result.Couplings) != 0 {
		t.Errorf("Expected 0 couplings with MinCoCommits=3, got %d", len(result.Couplings))
	}
}

func TestAnalyzer_Analyze_MinJaccardFilter(t *testing.T) {
	cfg := defaultCouplingConfig()
	cfg.MinJaccardThreshold = 0.5

	analyzer := NewAnalyzer(cfg)

	// Create a low Jaccard scenario: A in 10 commits, B in 10 commits, together in 1
	var changeSets []git.CommitChangeSet
	changeSets = append(changeSets, makeChangeSet("both", "a.go", "b.go"))
	for i := 0; i < 9; i++ {
		changeSets = append(changeSets, makeChangeSet("onlyA"+string(rune('0'+i)), "a.go", "other.go"))
	}
	for i := 0; i < 9; i++ {
		changeSets = append(changeSets, makeChangeSet("onlyB"+string(rune('0'+i)), "b.go", "other2.go"))
	}

	result := analyzer.Analyze(changeSets)

	// a.go/b.go pair has Jaccard = 1/(10+10-1) = 1/19 ≈ 0.053, below 0.5
	for _, c := range result.Couplings {
		if (c.FileA == "a.go" && c.FileB == "b.go") || (c.FileA == "b.go" && c.FileB == "a.go") {
			t.Error("a.go/b.go pair should be filtered out by MinJaccardThreshold=0.5")
		}
	}
}

func TestAnalyzer_Analyze_MaxFilesPerCommitFilter(t *testing.T) {
	cfg := defaultCouplingConfig()
	cfg.MaxFilesPerCommit = 3

	analyzer := NewAnalyzer(cfg)

	// Commit with 5 files (exceeds max of 3)
	changeSets := []git.CommitChangeSet{
		makeChangeSet("big", "a.go", "b.go", "c.go", "d.go", "e.go"),
		makeChangeSet("big2", "a.go", "b.go", "c.go", "d.go", "e.go"),
	}

	result := analyzer.Analyze(changeSets)

	if len(result.Couplings) != 0 {
		t.Errorf("Expected 0 couplings when commits exceed MaxFilesPerCommit, got %d", len(result.Couplings))
	}
}

func TestAnalyzer_Analyze_DeletedFilesExcluded(t *testing.T) {
	analyzer := NewAnalyzer(defaultCouplingConfig())

	changeSets := []git.CommitChangeSet{
		{
			Commit: git.CommitInfo{SHA: "1", When: time.Now(), Author: git.AuthorInfo{Name: "T", Email: "t@t.com"}, Message: "test"},
			Changes: []git.FileChange{
				{Path: "a.go", Kind: git.ChangeKindModified, LinesAdded: 1},
				{Path: "deleted.go", Kind: git.ChangeKindDeleted, LinesDeleted: 10},
			},
		},
	}

	result := analyzer.Analyze(changeSets)

	// Only 1 non-deleted file, so no pairs possible
	if len(result.Couplings) != 0 {
		t.Errorf("Expected 0 couplings when deleted files are excluded, got %d", len(result.Couplings))
	}
}

func TestAnalyzer_Analyze_TopPairsLimit(t *testing.T) {
	cfg := defaultCouplingConfig()
	cfg.TopPairs = 2

	analyzer := NewAnalyzer(cfg)

	// Create 4 pairs by having 4 commits with different file combinations
	changeSets := []git.CommitChangeSet{
		makeChangeSet("1", "a.go", "b.go"),
		makeChangeSet("2", "c.go", "d.go"),
		makeChangeSet("3", "e.go", "f.go"),
		makeChangeSet("4", "g.go", "h.go"),
	}

	result := analyzer.Analyze(changeSets)

	if len(result.Couplings) > 2 {
		t.Errorf("Expected at most 2 couplings with TopPairs=2, got %d", len(result.Couplings))
	}
}

func TestAnalyzer_Analyze_SortedByJaccard(t *testing.T) {
	cfg := defaultCouplingConfig()
	analyzer := NewAnalyzer(cfg)

	// Create scenarios with different Jaccard values
	var changeSets []git.CommitChangeSet

	// Pair a/b: 3 co-commits out of 3 each → Jaccard = 1.0
	for i := 0; i < 3; i++ {
		changeSets = append(changeSets, makeChangeSet("ab"+string(rune('0'+i)), "a.go", "b.go"))
	}

	// Pair c/d: 2 co-commits, c in 4 total, d in 4 total → Jaccard = 2/(4+4-2) = 2/6
	changeSets = append(changeSets, makeChangeSet("cd1", "c.go", "d.go"))
	changeSets = append(changeSets, makeChangeSet("cd2", "c.go", "d.go"))
	changeSets = append(changeSets, makeChangeSet("c_only1", "c.go", "x.go"))
	changeSets = append(changeSets, makeChangeSet("c_only2", "c.go", "y.go"))
	changeSets = append(changeSets, makeChangeSet("d_only1", "d.go", "x.go"))
	changeSets = append(changeSets, makeChangeSet("d_only2", "d.go", "y.go"))

	result := analyzer.Analyze(changeSets)

	// Verify descending Jaccard order
	for i := 1; i < len(result.Couplings); i++ {
		if result.Couplings[i].JaccardCoefficient > result.Couplings[i-1].JaccardCoefficient {
			t.Errorf("Couplings not sorted by Jaccard descending at index %d: %f > %f",
				i, result.Couplings[i].JaccardCoefficient, result.Couplings[i-1].JaccardCoefficient)
		}
	}
}
