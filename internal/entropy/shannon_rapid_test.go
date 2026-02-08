package entropy

import (
	"fmt"
	"math"
	"testing"

	"pgregory.net/rapid"

	"github.com/masmgr/bugspots-go/internal/git"
)

// --- Generators ---

func genFileChange() *rapid.Generator[git.FileChange] {
	return rapid.Custom(func(t *rapid.T) git.FileChange {
		return git.FileChange{
			Path:         fmt.Sprintf("file%d.go", rapid.IntRange(0, 100).Draw(t, "id")),
			LinesAdded:   rapid.IntRange(0, 1000).Draw(t, "added"),
			LinesDeleted: rapid.IntRange(0, 1000).Draw(t, "deleted"),
			Kind:         git.ChangeKindModified,
		}
	})
}

func genFileChanges() *rapid.Generator[[]git.FileChange] {
	return rapid.SliceOfN(genFileChange(), 0, 50)
}

// --- Property Tests ---

func TestRapidEntropy_OutputBounds(t *testing.T) {
	calc := NewCalculator()

	rapid.Check(t, func(t *rapid.T) {
		changes := genFileChanges().Draw(t, "changes")

		result := calc.CalculateCommitEntropy(changes)

		if result < 0.0 || result > 1.0 {
			t.Fatalf("CalculateCommitEntropy returned %f, expected in [0,1]", result)
		}
	})
}

func TestRapidEntropy_UniformMaximal(t *testing.T) {
	calc := NewCalculator()

	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(2, 20).Draw(t, "n")
		churn := rapid.IntRange(1, 500).Draw(t, "churn")

		changes := make([]git.FileChange, n)
		for i := 0; i < n; i++ {
			changes[i] = git.FileChange{
				Path:         fmt.Sprintf("file%d.go", i),
				LinesAdded:   churn,
				LinesDeleted: 0,
				Kind:         git.ChangeKindModified,
			}
		}

		result := calc.CalculateCommitEntropy(changes)

		if math.Abs(result-1.0) > 0.001 {
			t.Fatalf("Uniform distribution with %d files (churn=%d) gave entropy=%f, expected ≈ 1.0",
				n, churn, result)
		}
	})
}

func TestRapidEntropy_PermutationInvariant(t *testing.T) {
	calc := NewCalculator()

	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(2, 20).Draw(t, "n")
		changes := make([]git.FileChange, n)
		for i := 0; i < n; i++ {
			changes[i] = git.FileChange{
				Path:         fmt.Sprintf("file%d.go", i),
				LinesAdded:   rapid.IntRange(0, 500).Draw(t, fmt.Sprintf("added%d", i)),
				LinesDeleted: rapid.IntRange(0, 500).Draw(t, fmt.Sprintf("deleted%d", i)),
				Kind:         git.ChangeKindModified,
			}
		}

		original := calc.CalculateCommitEntropy(changes)

		// Reverse the slice
		reversed := make([]git.FileChange, n)
		for i := 0; i < n; i++ {
			reversed[i] = changes[n-1-i]
		}
		reversedResult := calc.CalculateCommitEntropy(reversed)

		if math.Abs(original-reversedResult) > 1e-10 {
			t.Fatalf("Permutation changed entropy: original=%f, reversed=%f", original, reversedResult)
		}
	})
}

func TestRapidEntropy_ScaleInvariant(t *testing.T) {
	calc := NewCalculator()

	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(2, 10).Draw(t, "n")
		k := rapid.IntRange(2, 100).Draw(t, "k")

		changes := make([]git.FileChange, n)
		scaled := make([]git.FileChange, n)
		hasChurn := false
		for i := 0; i < n; i++ {
			added := rapid.IntRange(0, 100).Draw(t, fmt.Sprintf("added%d", i))
			deleted := rapid.IntRange(0, 100).Draw(t, fmt.Sprintf("deleted%d", i))
			if added+deleted > 0 {
				hasChurn = true
			}
			changes[i] = git.FileChange{
				Path: fmt.Sprintf("file%d.go", i), LinesAdded: added, LinesDeleted: deleted,
				Kind: git.ChangeKindModified,
			}
			scaled[i] = git.FileChange{
				Path: fmt.Sprintf("file%d.go", i), LinesAdded: added * k, LinesDeleted: deleted * k,
				Kind: git.ChangeKindModified,
			}
		}

		if !hasChurn {
			return // Both would return 1.0 (zero churn), skip
		}

		original := calc.CalculateCommitEntropy(changes)
		scaledResult := calc.CalculateCommitEntropy(scaled)

		if math.Abs(original-scaledResult) > 1e-9 {
			t.Fatalf("Scale invariance violated: original=%f, scaled(k=%d)=%f", original, k, scaledResult)
		}
	})
}

func TestRapidEntropy_SingleFileZero(t *testing.T) {
	calc := NewCalculator()

	rapid.Check(t, func(t *rapid.T) {
		change := genFileChange().Draw(t, "change")
		changes := []git.FileChange{change}

		result := calc.CalculateCommitEntropy(changes)

		if result != 0.0 {
			t.Fatalf("Single file entropy = %f, expected 0.0", result)
		}
	})
}
