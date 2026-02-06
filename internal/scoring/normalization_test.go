package scoring

import (
	"math"
	"testing"
)

func TestMinMax_Range(t *testing.T) {
	tests := []struct {
		name     string
		min      float64
		max      float64
		expected float64
	}{
		{name: "Positive range", min: 0, max: 10, expected: 10},
		{name: "Same values", min: 5, max: 5, expected: 0},
		{name: "Negative range", min: -10, max: -2, expected: 8},
		{name: "Zero range", min: 0, max: 0, expected: 0},
		{name: "Large range", min: 0, max: 1e6, expected: 1e6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := MinMax{Min: tt.min, Max: tt.max}
			result := m.Range()
			if result != tt.expected {
				t.Errorf("Range() = %f, expected %f", result, tt.expected)
			}
		})
	}
}

func TestMinMax_IsSingleValue(t *testing.T) {
	tests := []struct {
		name     string
		min      float64
		max      float64
		expected bool
	}{
		{name: "Same values", min: 5, max: 5, expected: true},
		{name: "Within tolerance", min: 1.0, max: 1.0 + 1e-11, expected: true},
		{name: "Outside tolerance", min: 1.0, max: 1.0 + 1e-9, expected: false},
		{name: "Different values", min: 0, max: 1, expected: false},
		{name: "Both zero", min: 0, max: 0, expected: true},
		{name: "Negative same", min: -3, max: -3, expected: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := MinMax{Min: tt.min, Max: tt.max}
			result := m.IsSingleValue()
			if result != tt.expected {
				t.Errorf("IsSingleValue() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestClamp(t *testing.T) {
	tests := []struct {
		name     string
		value    float64
		expected float64
	}{
		{name: "Value in range", value: 0.5, expected: 0.5},
		{name: "Value at zero", value: 0.0, expected: 0.0},
		{name: "Value at one", value: 1.0, expected: 1.0},
		{name: "Value below zero", value: -0.5, expected: 0.0},
		{name: "Value above one", value: 1.5, expected: 1.0},
		{name: "Large negative", value: -100, expected: 0.0},
		{name: "Large positive", value: 100, expected: 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := clamp(tt.value)
			if result != tt.expected {
				t.Errorf("clamp(%f) = %f, expected %f", tt.value, result, tt.expected)
			}

			// Verify exported Clamp matches unexported clamp
			exported := Clamp(tt.value)
			if exported != result {
				t.Errorf("Clamp(%f) = %f, but clamp(%f) = %f", tt.value, exported, tt.value, result)
			}
		})
	}
}

func TestNormLog(t *testing.T) {
	tests := []struct {
		name      string
		value     float64
		minMax    MinMax
		expected  float64
		tolerance float64
	}{
		{name: "Min value", value: 0, minMax: MinMax{0, 10}, expected: 0.0, tolerance: 0.001},
		{name: "Max value", value: 10, minMax: MinMax{0, 10}, expected: 1.0, tolerance: 0.001},
		{name: "Single value positive", value: 5, minMax: MinMax{5, 5}, expected: 1.0, tolerance: 0.001},
		{name: "Single value zero", value: 0, minMax: MinMax{0, 0}, expected: 0.0, tolerance: 0.001},
		{name: "Single value both zero check", value: 0, minMax: MinMax{3, 3}, expected: 0.0, tolerance: 0.001},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormLog(tt.value, tt.minMax)
			if math.Abs(result-tt.expected) > tt.tolerance {
				t.Errorf("NormLog(%f, {%f,%f}) = %f, expected %f (±%f)",
					tt.value, tt.minMax.Min, tt.minMax.Max, result, tt.expected, tt.tolerance)
			}
		})
	}

	// Mid value should be between 0 and 1
	t.Run("Mid value between 0 and 1", func(t *testing.T) {
		result := NormLog(5, MinMax{0, 10})
		if result <= 0.0 || result >= 1.0 {
			t.Errorf("NormLog(5, {0,10}) = %f, expected between 0 and 1", result)
		}
	})

	// Value above max should be clamped to 1
	t.Run("Value above max clamped", func(t *testing.T) {
		result := NormLog(20, MinMax{0, 10})
		if result != 1.0 {
			t.Errorf("NormLog(20, {0,10}) = %f, expected 1.0", result)
		}
	})
}

func TestNormLog_Monotonicity(t *testing.T) {
	minMax := MinMax{Min: 0, Max: 100}
	values := []float64{0, 10, 25, 50, 75, 100}

	for i := 1; i < len(values); i++ {
		prev := NormLog(values[i-1], minMax)
		curr := NormLog(values[i], minMax)
		if curr < prev {
			t.Errorf("NormLog(%f) = %f < NormLog(%f) = %f, expected monotonically increasing",
				values[i], curr, values[i-1], prev)
		}
	}
}

func TestNormMinMax(t *testing.T) {
	tests := []struct {
		name      string
		value     float64
		minMax    MinMax
		expected  float64
		tolerance float64
	}{
		{name: "Min value", value: 0, minMax: MinMax{0, 10}, expected: 0.0, tolerance: 0.001},
		{name: "Max value", value: 10, minMax: MinMax{0, 10}, expected: 1.0, tolerance: 0.001},
		{name: "Mid value", value: 5, minMax: MinMax{0, 10}, expected: 0.5, tolerance: 0.001},
		{name: "Quarter value", value: 2.5, minMax: MinMax{0, 10}, expected: 0.25, tolerance: 0.001},
		{name: "Single value positive", value: 5, minMax: MinMax{5, 5}, expected: 1.0, tolerance: 0.001},
		{name: "Single value zero", value: 0, minMax: MinMax{0, 0}, expected: 0.0, tolerance: 0.001},
		{name: "Negative range", value: -5, minMax: MinMax{-10, 0}, expected: 0.5, tolerance: 0.001},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormMinMax(tt.value, tt.minMax)
			if math.Abs(result-tt.expected) > tt.tolerance {
				t.Errorf("NormMinMax(%f, {%f,%f}) = %f, expected %f (±%f)",
					tt.value, tt.minMax.Min, tt.minMax.Max, result, tt.expected, tt.tolerance)
			}
		})
	}
}

func TestRecencyDecay(t *testing.T) {
	tests := []struct {
		name      string
		daysSince float64
		halfLife  int
		expected  float64
		tolerance float64
	}{
		{name: "At zero days", daysSince: 0, halfLife: 30, expected: 1.0, tolerance: 0.001},
		{name: "At half-life", daysSince: 30, halfLife: 30, expected: 0.5, tolerance: 0.001},
		{name: "At double half-life", daysSince: 60, halfLife: 30, expected: 0.25, tolerance: 0.001},
		{name: "Negative daysSince treated as zero", daysSince: -10, halfLife: 30, expected: 1.0, tolerance: 0.001},
		{name: "Zero halfLife defaults to 30", daysSince: 30, halfLife: 0, expected: 0.5, tolerance: 0.001},
		{name: "Negative halfLife defaults to 30", daysSince: 30, halfLife: -5, expected: 0.5, tolerance: 0.001},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RecencyDecay(tt.daysSince, tt.halfLife)
			if math.Abs(result-tt.expected) > tt.tolerance {
				t.Errorf("RecencyDecay(%f, %d) = %f, expected %f (±%f)",
					tt.daysSince, tt.halfLife, result, tt.expected, tt.tolerance)
			}
		})
	}

	// Very old should approach zero
	t.Run("Very old approaches zero", func(t *testing.T) {
		result := RecencyDecay(365, 30)
		if result >= 0.01 {
			t.Errorf("RecencyDecay(365, 30) = %f, expected < 0.01", result)
		}
	})
}

func TestRecencyDecay_MonotonicDecrease(t *testing.T) {
	days := []float64{0, 10, 30, 60, 90, 180, 365}
	halfLife := 30

	for i := 1; i < len(days); i++ {
		prev := RecencyDecay(days[i-1], halfLife)
		curr := RecencyDecay(days[i], halfLife)
		if curr >= prev {
			t.Errorf("RecencyDecay(%f, %d) = %f >= RecencyDecay(%f, %d) = %f, expected monotonic decrease",
				days[i], halfLife, curr, days[i-1], halfLife, prev)
		}
	}
}
