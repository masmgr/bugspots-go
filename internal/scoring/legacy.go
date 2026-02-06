package scoring

import (
	"math"
	"sort"
	"time"
)

// LegacySigmoidScore implements the original bugspots scoring algorithm.
// It calculates a score based on the recency of a fix commit using a sigmoid function.
//
// The timestamp used in the equation is normalized from 0 to 1, where
// 0 is the earliest point in the code base, and 1 is now (when the algorithm was run).
// Note that the score changes over time with this algorithm due to the moving normalization;
// it's not meant to provide some objective score, only provide a means of comparison
// between one file and another at any one point in time.
//
// Formula: 1 / (1 + exp((-12*t)+12))
// where t is normalized time from 0 to 1
func LegacySigmoidScore(currentDate time.Time, oldestDate time.Time, fixDate time.Time) float64 {
	denom := currentDate.Sub(oldestDate).Seconds()
	if denom <= 0 {
		// No time range to normalize against; treat all fixes as equally "recent".
		return 1.0
	}

	t := 1 - (float64(currentDate.Sub(fixDate).Seconds()) / denom)
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}

	return 1 / (1 + math.Exp((-12*t)+12))
}

// LegacyFix represents a detected bugfix commit in the legacy format.
type LegacyFix struct {
	Message string
	Date    time.Time
	Files   []string
}

// LegacySpot represents a file's bugspot ranking in the legacy format.
type LegacySpot struct {
	File  string
	Score float64
}

// CalculateLegacyHotspots calculates hotspot scores using the legacy algorithm.
// It takes a list of fixes and returns a map of file paths to their cumulative scores.
func CalculateLegacyHotspots(fixes []LegacyFix, until time.Time, since time.Time) map[string]float64 {
	hotspots := make(map[string]float64)

	for _, fix := range fixes {
		for _, file := range fix.Files {
			if _, exists := hotspots[file]; !exists {
				hotspots[file] = 0
			}
			hotspots[file] += LegacySigmoidScore(until, since, fix.Date)
		}
	}

	return hotspots
}

// RankLegacyHotspots converts a hotspot map to a sorted slice of LegacySpot.
// Results are sorted by score in descending order.
func RankLegacyHotspots(hotspots map[string]float64, maxSpots int) []LegacySpot {
	spots := make([]LegacySpot, 0, len(hotspots))
	for file, score := range hotspots {
		spots = append(spots, LegacySpot{File: file, Score: score})
	}

	// Sort by score descending (O(n log n) instead of O(n²) bubble sort)
	sort.Slice(spots, func(i, j int) bool {
		return spots[i].Score > spots[j].Score
	})

	if maxSpots > 0 && maxSpots < len(spots) {
		spots = spots[:maxSpots]
	}

	return spots
}
