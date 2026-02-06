package burst

import (
	"sort"
	"time"

	"github.com/masmgr/bugspots-go/internal/aggregation"
)

// Calculator calculates burst scores using a sliding window approach.
// Burst score = (max commits in any window) / (total commits)
type Calculator struct {
	windowDays int
}

// NewCalculator creates a new burst score calculator.
func NewCalculator(windowDays int) *Calculator {
	if windowDays <= 0 {
		windowDays = 7
	}
	return &Calculator{windowDays: windowDays}
}

// Compute calculates burst scores for all file metrics.
func (c *Calculator) Compute(metrics map[string]*aggregation.FileMetrics) {
	for _, fm := range metrics {
		fm.BurstScore = c.CalculateBurstScore(fm.CommitTimes)
	}
}

// CalculateBurstScore calculates the burst score for a single file.
// Uses a two-pointer sliding window algorithm for O(n) complexity.
func (c *Calculator) CalculateBurstScore(commitTimes []time.Time) float64 {
	if len(commitTimes) == 0 {
		return 0.0
	}

	if len(commitTimes) == 1 {
		// Single commit is maximally "bursty"
		return 1.0
	}

	// Make a copy to avoid mutating the original slice
	times := make([]time.Time, len(commitTimes))
	copy(times, commitTimes)

	// Ensure ascending order for sliding-window math
	if !isSortedAscending(times) {
		if isSortedDescending(times) {
			// Commits from Git reader arrive in descending order (newest first).
			// Reverse is O(n) vs O(n log n) sorting.
			reverse(times)
		} else {
			sort.Slice(times, func(i, j int) bool {
				return times[i].Before(times[j])
			})
		}
	}

	windowDuration := time.Duration(c.windowDays) * 24 * time.Hour
	maxInWindow := 1

	// Two-pointer sliding window: O(n) algorithm
	left := 0
	for right := 0; right < len(times); right++ {
		// Move left pointer to maintain window constraint
		for times[right].Sub(times[left]) > windowDuration {
			left++
		}

		countInWindow := right - left + 1
		if countInWindow > maxInWindow {
			maxInWindow = countInWindow
		}
	}

	// Burst score = proportion of commits in the densest window
	burstScore := float64(maxInWindow) / float64(len(times))

	// Clamp to [0, 1]
	if burstScore < 0 {
		return 0.0
	}
	if burstScore > 1 {
		return 1.0
	}
	return burstScore
}

// isSortedAscending checks if the slice is sorted in ascending order.
func isSortedAscending(times []time.Time) bool {
	for i := 1; i < len(times); i++ {
		if times[i].Before(times[i-1]) {
			return false
		}
	}
	return true
}

// isSortedDescending checks if the slice is sorted in descending order.
func isSortedDescending(times []time.Time) bool {
	for i := 1; i < len(times); i++ {
		if times[i].After(times[i-1]) {
			return false
		}
	}
	return true
}

// reverse reverses a slice of times in place.
func reverse(times []time.Time) {
	for i, j := 0, len(times)-1; i < j; i, j = i+1, j-1 {
		times[i], times[j] = times[j], times[i]
	}
}
