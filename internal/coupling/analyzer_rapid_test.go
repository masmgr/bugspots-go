package coupling

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/masmgr/bugspots-go/config"
	"github.com/masmgr/bugspots-go/internal/git"
	"pgregory.net/rapid"
)

// --- Generators ---

func genFilePath() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		return fmt.Sprintf("%s.go", rapid.StringMatching(`[a-z]{1,8}`).Draw(t, "name"))
	})
}

func genCouplingChangeSets() *rapid.Generator[[]git.CommitChangeSet] {
	return rapid.Custom(func(t *rapid.T) []git.CommitChangeSet {
		count := rapid.IntRange(1, 20).Draw(t, "commits")
		sets := make([]git.CommitChangeSet, count)
		for i := 0; i < count; i++ {
			fileCount := rapid.IntRange(2, 10).Draw(t, fmt.Sprintf("files%d", i))
			changes := make([]git.FileChange, fileCount)
			for j := 0; j < fileCount; j++ {
				changes[j] = git.FileChange{
					Path:         fmt.Sprintf("file%d.go", rapid.IntRange(0, 20).Draw(t, fmt.Sprintf("path%d_%d", i, j))),
					LinesAdded:   rapid.IntRange(1, 100).Draw(t, fmt.Sprintf("added%d_%d", i, j)),
					LinesDeleted: rapid.IntRange(0, 50).Draw(t, fmt.Sprintf("deleted%d_%d", i, j)),
					Kind:         git.ChangeKindModified,
				}
			}
			sets[i] = git.CommitChangeSet{
				Commit: git.CommitInfo{
					SHA:     fmt.Sprintf("sha%d", i),
					When:    time.Now(),
					Author:  git.AuthorInfo{Name: "Test", Email: "test@example.com"},
					Message: "test commit",
				},
				Changes: changes,
			}
		}
		return sets
	})
}

// --- FilePair Property Tests ---

func TestRapidFilePair_Commutative(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		a := genFilePath().Draw(t, "a")
		b := genFilePath().Draw(t, "b")

		pair1 := NewFilePair(a, b)
		pair2 := NewFilePair(b, a)

		if pair1 != pair2 {
			t.Fatalf("NewFilePair not commutative: (%q,%q)=%v, (%q,%q)=%v",
				a, b, pair1, b, a, pair2)
		}
	})
}

func TestRapidFilePair_Idempotent(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		a := genFilePath().Draw(t, "a")
		b := genFilePath().Draw(t, "b")

		pair := NewFilePair(a, b)
		repaired := NewFilePair(pair.FileA, pair.FileB)

		if pair != repaired {
			t.Fatalf("NewFilePair not idempotent: pair=%v, re-paired=%v", pair, repaired)
		}
	})
}

func TestRapidFilePair_LexOrder(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		a := genFilePath().Draw(t, "a")
		b := genFilePath().Draw(t, "b")

		pair := NewFilePair(a, b)

		if strings.ToLower(pair.FileA) > strings.ToLower(pair.FileB) {
			t.Fatalf("FilePair not in lexicographic order: FileA=%q, FileB=%q", pair.FileA, pair.FileB)
		}
	})
}

// --- Analyze Property Tests ---

func TestRapidAnalyze_JaccardBounds(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		cfg := config.CouplingConfig{
			MinCoCommits:        1,
			MinJaccardThreshold: 0.0,
			MaxFilesPerCommit:   50,
			TopPairs:            100,
		}
		analyzer := NewAnalyzer(cfg)
		changeSets := genCouplingChangeSets().Draw(t, "changeSets")

		result := analyzer.Analyze(changeSets)

		for i, c := range result.Couplings {
			if c.JaccardCoefficient < 0.0 || c.JaccardCoefficient > 1.0 {
				t.Fatalf("Coupling[%d] Jaccard=%f, expected in [0,1]", i, c.JaccardCoefficient)
			}
		}
	})
}

func TestRapidAnalyze_ConfidenceBounds(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		cfg := config.CouplingConfig{
			MinCoCommits:        1,
			MinJaccardThreshold: 0.0,
			MaxFilesPerCommit:   50,
			TopPairs:            100,
		}
		analyzer := NewAnalyzer(cfg)
		changeSets := genCouplingChangeSets().Draw(t, "changeSets")

		result := analyzer.Analyze(changeSets)

		for i, c := range result.Couplings {
			if c.Confidence < 0.0 || c.Confidence > 1.0 {
				t.Fatalf("Coupling[%d] Confidence=%f, expected in [0,1]", i, c.Confidence)
			}
		}
	})
}

func TestRapidAnalyze_SortedDescending(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		cfg := config.CouplingConfig{
			MinCoCommits:        1,
			MinJaccardThreshold: 0.0,
			MaxFilesPerCommit:   50,
			TopPairs:            100,
		}
		analyzer := NewAnalyzer(cfg)
		changeSets := genCouplingChangeSets().Draw(t, "changeSets")

		result := analyzer.Analyze(changeSets)

		for i := 1; i < len(result.Couplings); i++ {
			if result.Couplings[i].JaccardCoefficient > result.Couplings[i-1].JaccardCoefficient {
				t.Fatalf("Couplings not sorted by Jaccard descending at index %d: %f > %f",
					i, result.Couplings[i].JaccardCoefficient, result.Couplings[i-1].JaccardCoefficient)
			}
		}
	})
}

func TestRapidAnalyze_TopPairsLimit(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		topPairs := rapid.IntRange(1, 10).Draw(t, "topPairs")
		cfg := config.CouplingConfig{
			MinCoCommits:        1,
			MinJaccardThreshold: 0.0,
			MaxFilesPerCommit:   50,
			TopPairs:            topPairs,
		}
		analyzer := NewAnalyzer(cfg)
		changeSets := genCouplingChangeSets().Draw(t, "changeSets")

		result := analyzer.Analyze(changeSets)

		if len(result.Couplings) > topPairs {
			t.Fatalf("len(Couplings)=%d exceeds TopPairs=%d", len(result.Couplings), topPairs)
		}
	})
}
