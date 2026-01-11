package main

import (
	"math"
	"testing"
	"time"
)

// TestCalcScore tests the CalcScore function with various time ranges
func TestCalcScore(t *testing.T) {
	tests := []struct {
		name       string
		current    time.Time
		oldest     time.Time
		fixDate    time.Time
		expected   float64
		tolerance  float64
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
			result := CalcScore(tt.current, tt.oldest, tt.fixDate)

			if math.Abs(result-tt.expected) > tt.tolerance {
				t.Errorf("CalcScore() = %f, expected %f (±%f)", result, tt.expected, tt.tolerance)
			}

			// Verify result is between 0 and 1 (sigmoid always returns this)
			if result < 0 || result > 1 {
				t.Errorf("CalcScore() returned %f, expected value between 0 and 1", result)
			}
		})
	}
}

// TestCalcScore_SigmoidProperties tests mathematical properties of the sigmoid function
func TestCalcScore_SigmoidProperties(t *testing.T) {
	current := time.Date(2025, 1, 11, 0, 0, 0, 0, time.UTC)
	oldest := time.Date(2022, 1, 11, 0, 0, 0, 0, time.UTC)

	// Recent fixes should have higher scores than older fixes
	recentFix := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	oldFix := time.Date(2022, 6, 11, 0, 0, 0, 0, time.UTC)

	recentScore := CalcScore(current, oldest, recentFix)
	oldScore := CalcScore(current, oldest, oldFix)

	if recentScore <= oldScore {
		t.Errorf("Recent fix score (%f) should be higher than old fix score (%f)", recentScore, oldScore)
	}
}

// TestMinInt tests the minInt helper function
func TestMinInt(t *testing.T) {
	tests := []struct {
		a        int
		b        int
		expected int
	}{
		{1, 2, 1},
		{2, 1, 1},
		{5, 5, 5},
		{0, 10, 0},
		{-1, 1, -1},
		{-5, -2, -5},
		{100, 50, 50},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := minInt(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("minInt(%d, %d) = %d, expected %d", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

// TestMinInt_Symmetry tests that minInt produces same result regardless of argument order
func TestMinInt_Symmetry(t *testing.T) {
	tests := []struct {
		a int
		b int
	}{
		{10, 20},
		{-5, 10},
		{0, 0},
		{1000, 500},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result1 := minInt(tt.a, tt.b)
			result2 := minInt(tt.b, tt.a)

			if result1 != result2 {
				t.Errorf("minInt(%d, %d) = %d, but minInt(%d, %d) = %d (should be equal)",
					tt.a, tt.b, result1, tt.b, tt.a, result2)
			}
		})
	}
}
