package scoring

import "math"

// MinMax represents a min/max range for normalization.
type MinMax struct {
	Min float64
	Max float64
}

// Range returns max - min.
func (m MinMax) Range() float64 {
	return m.Max - m.Min
}

// IsSingleValue returns true if min equals max (within tolerance).
func (m MinMax) IsSingleValue() bool {
	return math.Abs(m.Max-m.Min) < 1e-10
}

// NormLog normalizes a value using logarithmic scaling.
// Formula: (log(1+x) - log(1+min)) / (log(1+max) - log(1+min))
func NormLog(value float64, minMax MinMax) float64 {
	if minMax.IsSingleValue() {
		if value > 0 {
			return 1.0
		}
		return 0.0
	}

	logValue := math.Log(1 + value)
	logMin := math.Log(1 + minMax.Min)
	logMax := math.Log(1 + minMax.Max)

	rng := logMax - logMin
	if rng <= 0 {
		if value > 0 {
			return 1.0
		}
		return 0.0
	}

	normalized := (logValue - logMin) / rng
	return clamp(normalized)
}

// NormMinMax normalizes a value using linear min-max scaling.
// Formula: (x - min) / (max - min)
func NormMinMax(value float64, minMax MinMax) float64 {
	if minMax.IsSingleValue() {
		if value > 0 {
			return 1.0
		}
		return 0.0
	}

	normalized := (value - minMax.Min) / minMax.Range()
	return clamp(normalized)
}

// RecencyDecay calculates recency decay based on days since last modification.
// Formula: exp(-ln(2) * days / halfLifeDays)
// Result is 0.5 when days == halfLifeDays.
func RecencyDecay(daysSince float64, halfLifeDays int) float64 {
	if daysSince < 0 {
		daysSince = 0
	}

	if halfLifeDays <= 0 {
		halfLifeDays = 30
	}

	// Using natural log of 2 for proper half-life calculation
	// exp(-ln(2) * days / halfLife) = 0.5 when days = halfLife
	decayConstant := math.Ln2 / float64(halfLifeDays)
	decay := math.Exp(-decayConstant * daysSince)

	return clamp(decay)
}

// clamp constrains a value between 0 and 1.
func clamp(value float64) float64 {
	return math.Max(0.0, math.Min(1.0, value))
}

// Clamp is the exported version of clamp for use by other packages.
func Clamp(value float64) float64 {
	return clamp(value)
}
