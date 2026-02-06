package entropy

import (
	"math"

	"github.com/masmgr/bugspots-go/internal/git"
)

// Calculator calculates Shannon entropy for change distribution within a commit.
// Based on Hassan (2009) "Predicting Faults Using the Complexity of Code Changes".
type Calculator struct{}

// NewCalculator creates a new entropy calculator.
func NewCalculator() *Calculator {
	return &Calculator{}
}

// CalculateCommitEntropy calculates the normalized Shannon entropy for a commit's changes.
// Returns a value between 0 and 1:
//   - 0 = focused change (single file or all changes in one file)
//   - 1 = highly dispersed change (changes evenly distributed)
func (c *Calculator) CalculateCommitEntropy(changes []git.FileChange) float64 {
	if len(changes) == 0 {
		return 0.0
	}

	if len(changes) == 1 {
		// Single file change has no distribution, entropy is 0
		return 0.0
	}

	// Calculate total churn across all files
	totalChurn := 0
	for _, change := range changes {
		totalChurn += change.Churn()
	}

	if totalChurn == 0 {
		// No actual changes, treat as uniform distribution
		return 1.0
	}

	// Calculate Shannon entropy: -Σ(p_i × log2(p_i))
	entropy := 0.0
	for _, change := range changes {
		churn := change.Churn()
		if churn > 0 {
			p := float64(churn) / float64(totalChurn)
			entropy -= p * math.Log2(p)
		}
	}

	// Normalize by maximum possible entropy (log2(n) for n files)
	// This gives us a value between 0 and 1
	maxEntropy := math.Log2(float64(len(changes)))
	if maxEntropy <= 0 {
		return 0.0
	}

	normalizedEntropy := entropy / maxEntropy

	// Clamp to [0, 1] for safety
	if normalizedEntropy < 0 {
		return 0.0
	}
	if normalizedEntropy > 1 {
		return 1.0
	}
	return normalizedEntropy
}
