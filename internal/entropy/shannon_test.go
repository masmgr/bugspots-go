package entropy

import (
	"math"
	"testing"

	"github.com/masmgr/bugspots-go/internal/git"
)

func TestCalculateCommitEntropy_EmptyChanges(t *testing.T) {
	calc := NewCalculator()
	result := calc.CalculateCommitEntropy(nil)
	if result != 0.0 {
		t.Errorf("CalculateCommitEntropy(nil) = %f, expected 0.0", result)
	}

	result = calc.CalculateCommitEntropy([]git.FileChange{})
	if result != 0.0 {
		t.Errorf("CalculateCommitEntropy([]) = %f, expected 0.0", result)
	}
}

func TestCalculateCommitEntropy_SingleFile(t *testing.T) {
	calc := NewCalculator()
	changes := []git.FileChange{
		{Path: "file1.go", LinesAdded: 10, LinesDeleted: 5},
	}
	result := calc.CalculateCommitEntropy(changes)
	if result != 0.0 {
		t.Errorf("CalculateCommitEntropy(single file) = %f, expected 0.0", result)
	}
}

func TestCalculateCommitEntropy_UniformDistribution(t *testing.T) {
	calc := NewCalculator()

	// Two files with equal churn → maximum entropy = 1.0
	changes := []git.FileChange{
		{Path: "file1.go", LinesAdded: 10, LinesDeleted: 10},
		{Path: "file2.go", LinesAdded: 10, LinesDeleted: 10},
	}
	result := calc.CalculateCommitEntropy(changes)
	if math.Abs(result-1.0) > 0.001 {
		t.Errorf("CalculateCommitEntropy(uniform 2 files) = %f, expected 1.0", result)
	}

	// Three files with equal churn → maximum entropy = 1.0
	changes3 := []git.FileChange{
		{Path: "file1.go", LinesAdded: 5, LinesDeleted: 5},
		{Path: "file2.go", LinesAdded: 5, LinesDeleted: 5},
		{Path: "file3.go", LinesAdded: 5, LinesDeleted: 5},
	}
	result3 := calc.CalculateCommitEntropy(changes3)
	if math.Abs(result3-1.0) > 0.001 {
		t.Errorf("CalculateCommitEntropy(uniform 3 files) = %f, expected 1.0", result3)
	}
}

func TestCalculateCommitEntropy_SkewedDistribution(t *testing.T) {
	calc := NewCalculator()

	// One file dominates → low entropy
	changes := []git.FileChange{
		{Path: "file1.go", LinesAdded: 100, LinesDeleted: 0},
		{Path: "file2.go", LinesAdded: 1, LinesDeleted: 0},
	}
	result := calc.CalculateCommitEntropy(changes)
	if result >= 0.2 {
		t.Errorf("CalculateCommitEntropy(skewed) = %f, expected < 0.2", result)
	}
}

func TestCalculateCommitEntropy_ZeroChurn(t *testing.T) {
	calc := NewCalculator()

	// Multiple files all with zero churn → 1.0
	changes := []git.FileChange{
		{Path: "file1.go", LinesAdded: 0, LinesDeleted: 0},
		{Path: "file2.go", LinesAdded: 0, LinesDeleted: 0},
	}
	result := calc.CalculateCommitEntropy(changes)
	if result != 1.0 {
		t.Errorf("CalculateCommitEntropy(zero churn) = %f, expected 1.0", result)
	}
}

func TestCalculateCommitEntropy_BoundedRange(t *testing.T) {
	calc := NewCalculator()

	testCases := [][]git.FileChange{
		{{Path: "a.go", LinesAdded: 1, LinesDeleted: 0}},
		{
			{Path: "a.go", LinesAdded: 50, LinesDeleted: 50},
			{Path: "b.go", LinesAdded: 1, LinesDeleted: 0},
		},
		{
			{Path: "a.go", LinesAdded: 10, LinesDeleted: 0},
			{Path: "b.go", LinesAdded: 10, LinesDeleted: 0},
			{Path: "c.go", LinesAdded: 10, LinesDeleted: 0},
			{Path: "d.go", LinesAdded: 10, LinesDeleted: 0},
		},
	}

	for i, changes := range testCases {
		result := calc.CalculateCommitEntropy(changes)
		if result < 0.0 || result > 1.0 {
			t.Errorf("Case %d: CalculateCommitEntropy() = %f, expected in [0, 1]", i, result)
		}
	}
}
