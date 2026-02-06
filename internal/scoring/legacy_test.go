package scoring

import (
	"math"
	"testing"
	"time"
)

// TestLegacySigmoidScore tests the LegacySigmoidScore function with various time ranges
func TestLegacySigmoidScore(t *testing.T) {
	tests := []struct {
		name      string
		current   time.Time
		oldest    time.Time
		fixDate   time.Time
		expected  float64
		tolerance float64
	}{
		{
			name:      "Fix at current date (most recent, t=1)",
			current:   time.Date(2025, 1, 11, 0, 0, 0, 0, time.UTC),
			oldest:    time.Date(2022, 1, 11, 0, 0, 0, 0, time.UTC),
			fixDate:   time.Date(2025, 1, 11, 0, 0, 0, 0, time.UTC),
			expected:  0.5, // Sigmoid at t=1 equals 0.5 (1/(1+exp(-12+12)))
			tolerance: 0.001,
		},
		{
			name:      "Fix at oldest date (t=0)",
			current:   time.Date(2025, 1, 11, 0, 0, 0, 0, time.UTC),
			oldest:    time.Date(2022, 1, 11, 0, 0, 0, 0, time.UTC),
			fixDate:   time.Date(2022, 1, 11, 0, 0, 0, 0, time.UTC),
			expected:  0.000061, // Sigmoid at t=0 equals ~0 (1/(1+exp(12)))
			tolerance: 0.0001,
		},
		{
			name:      "Fix at 2/3 of time span (1 year ago out of 3 years)",
			current:   time.Date(2025, 1, 11, 0, 0, 0, 0, time.UTC),
			oldest:    time.Date(2022, 1, 11, 0, 0, 0, 0, time.UTC),
			fixDate:   time.Date(2024, 1, 11, 0, 0, 0, 0, time.UTC),
			expected:  0.017858, // t ≈ 0.667, sigmoid at 0.667 = 1/(1+exp(4))
			tolerance: 0.001,
		},
		{
			name:      "Fix at 1/3 of time span (2 years ago out of 3 years)",
			current:   time.Date(2025, 1, 11, 0, 0, 0, 0, time.UTC),
			oldest:    time.Date(2022, 1, 11, 0, 0, 0, 0, time.UTC),
			fixDate:   time.Date(2023, 1, 11, 0, 0, 0, 0, time.UTC),
			expected:  0.000335, // t ≈ 0.333, sigmoid at 0.333
			tolerance: 0.001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := LegacySigmoidScore(tt.current, tt.oldest, tt.fixDate)

			if math.Abs(result-tt.expected) > tt.tolerance {
				t.Errorf("LegacySigmoidScore() = %f, expected %f (±%f)", result, tt.expected, tt.tolerance)
			}

			// Verify result is between 0 and 1 (sigmoid always returns this)
			if result < 0 || result > 1 {
				t.Errorf("LegacySigmoidScore() returned %f, expected value between 0 and 1", result)
			}
		})
	}
}

// TestLegacySigmoidScore_SigmoidProperties tests mathematical properties of the sigmoid function
func TestLegacySigmoidScore_SigmoidProperties(t *testing.T) {
	current := time.Date(2025, 1, 11, 0, 0, 0, 0, time.UTC)
	oldest := time.Date(2022, 1, 11, 0, 0, 0, 0, time.UTC)

	// Recent fixes should have higher scores than older fixes
	recentFix := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	oldFix := time.Date(2022, 6, 11, 0, 0, 0, 0, time.UTC)

	recentScore := LegacySigmoidScore(current, oldest, recentFix)
	oldScore := LegacySigmoidScore(current, oldest, oldFix)

	if recentScore <= oldScore {
		t.Errorf("Recent fix score (%f) should be higher than old fix score (%f)", recentScore, oldScore)
	}
}

// TestCalculateLegacyHotspots tests the hotspot calculation
func TestCalculateLegacyHotspots(t *testing.T) {
	until := time.Date(2025, 1, 11, 0, 0, 0, 0, time.UTC)
	since := time.Date(2022, 1, 11, 0, 0, 0, 0, time.UTC)

	fixes := []LegacyFix{
		{
			Message: "fix: bug in file1",
			Date:    time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC),
			Files:   []string{"file1.go", "file2.go"},
		},
		{
			Message: "fix: another bug in file1",
			Date:    time.Date(2024, 11, 1, 0, 0, 0, 0, time.UTC),
			Files:   []string{"file1.go"},
		},
	}

	hotspots := CalculateLegacyHotspots(fixes, until, since)

	// file1.go should have a higher score because it was in both fixes
	if hotspots["file1.go"] <= hotspots["file2.go"] {
		t.Errorf("file1.go score (%f) should be higher than file2.go score (%f)",
			hotspots["file1.go"], hotspots["file2.go"])
	}

	// file1.go should appear in the hotspots
	if _, exists := hotspots["file1.go"]; !exists {
		t.Error("file1.go should be in hotspots")
	}

	// file2.go should appear in the hotspots
	if _, exists := hotspots["file2.go"]; !exists {
		t.Error("file2.go should be in hotspots")
	}
}

// TestRankLegacyHotspots tests the hotspot ranking
func TestRankLegacyHotspots(t *testing.T) {
	hotspots := map[string]float64{
		"file1.go": 0.5,
		"file2.go": 0.8,
		"file3.go": 0.2,
	}

	spots := RankLegacyHotspots(hotspots, 0)

	// Should be sorted by score descending
	if spots[0].File != "file2.go" {
		t.Errorf("Expected file2.go to be first, got %s", spots[0].File)
	}
	if spots[1].File != "file1.go" {
		t.Errorf("Expected file1.go to be second, got %s", spots[1].File)
	}
	if spots[2].File != "file3.go" {
		t.Errorf("Expected file3.go to be third, got %s", spots[2].File)
	}
}

// TestRankLegacyHotspots_MaxSpots tests the maxSpots limit
func TestRankLegacyHotspots_MaxSpots(t *testing.T) {
	hotspots := map[string]float64{
		"file1.go": 0.5,
		"file2.go": 0.8,
		"file3.go": 0.2,
	}

	spots := RankLegacyHotspots(hotspots, 2)

	if len(spots) != 2 {
		t.Errorf("Expected 2 spots, got %d", len(spots))
	}
}
