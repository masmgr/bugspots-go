package burst

import (
	"math"
	"testing"
	"time"
)

func TestNewCalculator(t *testing.T) {
	tests := []struct {
		name       string
		windowDays int
	}{
		{name: "Positive window", windowDays: 14},
		{name: "Zero defaults", windowDays: 0},
		{name: "Negative defaults", windowDays: -5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calc := NewCalculator(tt.windowDays)
			if calc == nil {
				t.Error("NewCalculator returned nil")
			}
		})
	}
}

func TestCalculateBurstScore_Empty(t *testing.T) {
	calc := NewCalculator(7)
	result := calc.CalculateBurstScore(nil)
	if result != 0.0 {
		t.Errorf("CalculateBurstScore(nil) = %f, expected 0.0", result)
	}

	result = calc.CalculateBurstScore([]time.Time{})
	if result != 0.0 {
		t.Errorf("CalculateBurstScore([]) = %f, expected 0.0", result)
	}
}

func TestCalculateBurstScore_SingleCommit(t *testing.T) {
	calc := NewCalculator(7)
	times := []time.Time{time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}
	result := calc.CalculateBurstScore(times)
	if result != 1.0 {
		t.Errorf("CalculateBurstScore(single) = %f, expected 1.0", result)
	}
}

func TestCalculateBurstScore_AllInOneWindow(t *testing.T) {
	calc := NewCalculator(7)
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	times := []time.Time{
		base,
		base.Add(1 * 24 * time.Hour),
		base.Add(2 * 24 * time.Hour),
		base.Add(3 * 24 * time.Hour),
		base.Add(4 * 24 * time.Hour),
	}
	result := calc.CalculateBurstScore(times)
	if math.Abs(result-1.0) > 0.001 {
		t.Errorf("CalculateBurstScore(all in window) = %f, expected 1.0", result)
	}
}

func TestCalculateBurstScore_SpreadAcross(t *testing.T) {
	calc := NewCalculator(7)
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// 10 commits, each 30 days apart → max 1 per window → score = 0.1
	times := make([]time.Time, 10)
	for i := 0; i < 10; i++ {
		times[i] = base.Add(time.Duration(i*30) * 24 * time.Hour)
	}

	result := calc.CalculateBurstScore(times)
	if math.Abs(result-0.1) > 0.001 {
		t.Errorf("CalculateBurstScore(spread) = %f, expected 0.1", result)
	}
}

func TestCalculateBurstScore_TwoClusters(t *testing.T) {
	calc := NewCalculator(7)
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// 5 commits in first week, 5 commits 6 months later
	times := make([]time.Time, 10)
	for i := 0; i < 5; i++ {
		times[i] = base.Add(time.Duration(i) * 24 * time.Hour)
	}
	for i := 5; i < 10; i++ {
		times[i] = base.Add(time.Duration(180+(i-5)) * 24 * time.Hour)
	}

	result := calc.CalculateBurstScore(times)
	if math.Abs(result-0.5) > 0.001 {
		t.Errorf("CalculateBurstScore(two clusters) = %f, expected 0.5", result)
	}
}

func TestCalculateBurstScore_DescendingOrder(t *testing.T) {
	calc := NewCalculator(7)
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Descending order (newest first, as git reader provides)
	times := []time.Time{
		base.Add(4 * 24 * time.Hour),
		base.Add(3 * 24 * time.Hour),
		base.Add(2 * 24 * time.Hour),
		base.Add(1 * 24 * time.Hour),
		base,
	}
	result := calc.CalculateBurstScore(times)
	if math.Abs(result-1.0) > 0.001 {
		t.Errorf("CalculateBurstScore(descending) = %f, expected 1.0", result)
	}
}

func TestCalculateBurstScore_UnsortedOrder(t *testing.T) {
	calc := NewCalculator(7)
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Unsorted order
	times := []time.Time{
		base.Add(3 * 24 * time.Hour),
		base,
		base.Add(4 * 24 * time.Hour),
		base.Add(1 * 24 * time.Hour),
		base.Add(2 * 24 * time.Hour),
	}
	result := calc.CalculateBurstScore(times)
	if math.Abs(result-1.0) > 0.001 {
		t.Errorf("CalculateBurstScore(unsorted) = %f, expected 1.0", result)
	}
}

func TestCalculateBurstScore_DoesNotMutateInput(t *testing.T) {
	calc := NewCalculator(7)
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Descending order input
	original := []time.Time{
		base.Add(4 * 24 * time.Hour),
		base.Add(3 * 24 * time.Hour),
		base.Add(2 * 24 * time.Hour),
		base.Add(1 * 24 * time.Hour),
		base,
	}

	// Copy for comparison
	inputCopy := make([]time.Time, len(original))
	copy(inputCopy, original)

	calc.CalculateBurstScore(original)

	for i := range original {
		if !original[i].Equal(inputCopy[i]) {
			t.Errorf("Input was mutated at index %d: got %v, expected %v", i, original[i], inputCopy[i])
		}
	}
}

func TestIsSortedAscending(t *testing.T) {
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		times    []time.Time
		expected bool
	}{
		{
			name:     "Ascending",
			times:    []time.Time{base, base.Add(time.Hour), base.Add(2 * time.Hour)},
			expected: true,
		},
		{
			name:     "Descending",
			times:    []time.Time{base.Add(2 * time.Hour), base.Add(time.Hour), base},
			expected: false,
		},
		{
			name:     "Single element",
			times:    []time.Time{base},
			expected: true,
		},
		{
			name:     "Empty",
			times:    []time.Time{},
			expected: true,
		},
		{
			name:     "Unsorted",
			times:    []time.Time{base, base.Add(2 * time.Hour), base.Add(time.Hour)},
			expected: false,
		},
		{
			name:     "Equal times",
			times:    []time.Time{base, base, base},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSortedAscending(tt.times)
			if result != tt.expected {
				t.Errorf("isSortedAscending() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestIsSortedDescending(t *testing.T) {
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		times    []time.Time
		expected bool
	}{
		{
			name:     "Descending",
			times:    []time.Time{base.Add(2 * time.Hour), base.Add(time.Hour), base},
			expected: true,
		},
		{
			name:     "Ascending",
			times:    []time.Time{base, base.Add(time.Hour), base.Add(2 * time.Hour)},
			expected: false,
		},
		{
			name:     "Single element",
			times:    []time.Time{base},
			expected: true,
		},
		{
			name:     "Empty",
			times:    []time.Time{},
			expected: true,
		},
		{
			name:     "Unsorted",
			times:    []time.Time{base, base.Add(2 * time.Hour), base.Add(time.Hour)},
			expected: false,
		},
		{
			name:     "Equal times",
			times:    []time.Time{base, base, base},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSortedDescending(tt.times)
			if result != tt.expected {
				t.Errorf("isSortedDescending() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestReverse(t *testing.T) {
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	t1 := base
	t2 := base.Add(time.Hour)
	t3 := base.Add(2 * time.Hour)

	t.Run("Three elements", func(t *testing.T) {
		times := []time.Time{t1, t2, t3}
		reverse(times)
		if !times[0].Equal(t3) || !times[1].Equal(t2) || !times[2].Equal(t1) {
			t.Errorf("reverse([t1,t2,t3]) = [%v,%v,%v], expected [t3,t2,t1]", times[0], times[1], times[2])
		}
	})

	t.Run("Two elements", func(t *testing.T) {
		times := []time.Time{t1, t2}
		reverse(times)
		if !times[0].Equal(t2) || !times[1].Equal(t1) {
			t.Errorf("reverse([t1,t2]) = [%v,%v], expected [t2,t1]", times[0], times[1])
		}
	})

	t.Run("Single element", func(t *testing.T) {
		times := []time.Time{t1}
		reverse(times)
		if !times[0].Equal(t1) {
			t.Errorf("reverse([t1]) changed the element")
		}
	})
}
