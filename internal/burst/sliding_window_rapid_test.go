package burst

import (
	"fmt"
	"math"
	"testing"
	"time"

	"pgregory.net/rapid"
)

// --- Generators ---

func genCommitTimes() *rapid.Generator[[]time.Time] {
	return rapid.Custom(func(t *rapid.T) []time.Time {
		count := rapid.IntRange(0, 100).Draw(t, "count")
		base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		times := make([]time.Time, count)
		for i := 0; i < count; i++ {
			dayOffset := rapid.IntRange(0, 365).Draw(t, fmt.Sprintf("day%d", i))
			hourOffset := rapid.IntRange(0, 23).Draw(t, fmt.Sprintf("hour%d", i))
			times[i] = base.Add(time.Duration(dayOffset)*24*time.Hour + time.Duration(hourOffset)*time.Hour)
		}
		return times
	})
}

// --- Property Tests ---

func TestRapidBurst_OutputBounds(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		windowDays := rapid.IntRange(1, 30).Draw(t, "windowDays")
		calc := NewCalculator(windowDays)
		times := genCommitTimes().Draw(t, "times")

		result := calc.CalculateBurstScore(times)

		if result < 0.0 || result > 1.0 {
			t.Fatalf("CalculateBurstScore returned %f, expected in [0,1]", result)
		}
	})
}

func TestRapidBurst_EmptyZero(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		windowDays := rapid.IntRange(1, 30).Draw(t, "windowDays")
		calc := NewCalculator(windowDays)

		result := calc.CalculateBurstScore(nil)
		if result != 0.0 {
			t.Fatalf("Empty input gave %f, expected 0.0", result)
		}

		result = calc.CalculateBurstScore([]time.Time{})
		if result != 0.0 {
			t.Fatalf("Empty slice gave %f, expected 0.0", result)
		}
	})
}

func TestRapidBurst_SingleOne(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		windowDays := rapid.IntRange(1, 30).Draw(t, "windowDays")
		calc := NewCalculator(windowDays)
		base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		dayOffset := rapid.IntRange(0, 365).Draw(t, "dayOffset")
		single := []time.Time{base.Add(time.Duration(dayOffset) * 24 * time.Hour)}

		result := calc.CalculateBurstScore(single)

		if result != 1.0 {
			t.Fatalf("Single commit gave %f, expected 1.0", result)
		}
	})
}

func TestRapidBurst_SortOrderInvariant(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		windowDays := rapid.IntRange(1, 30).Draw(t, "windowDays")
		calc := NewCalculator(windowDays)

		count := rapid.IntRange(2, 50).Draw(t, "count")
		base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

		ascending := make([]time.Time, count)
		for i := 0; i < count; i++ {
			dayOffset := rapid.IntRange(0, 365).Draw(t, fmt.Sprintf("day%d", i))
			ascending[i] = base.Add(time.Duration(dayOffset) * 24 * time.Hour)
		}

		// Create reversed copy
		descending := make([]time.Time, count)
		for i := 0; i < count; i++ {
			descending[i] = ascending[count-1-i]
		}

		resAsc := calc.CalculateBurstScore(ascending)
		resDesc := calc.CalculateBurstScore(descending)

		if math.Abs(resAsc-resDesc) > 1e-10 {
			t.Fatalf("Sort order affected result: ascending=%f, descending=%f", resAsc, resDesc)
		}
	})
}

func TestRapidBurst_NonMutating(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		windowDays := rapid.IntRange(1, 30).Draw(t, "windowDays")
		calc := NewCalculator(windowDays)
		times := genCommitTimes().Draw(t, "times")

		if len(times) == 0 {
			return
		}

		// Save copies
		original := make([]time.Time, len(times))
		copy(original, times)

		calc.CalculateBurstScore(times)

		for i := range times {
			if !times[i].Equal(original[i]) {
				t.Fatalf("Input mutated at index %d: got %v, expected %v", i, times[i], original[i])
			}
		}
	})
}
