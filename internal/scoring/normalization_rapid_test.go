package scoring

import (
	"math"
	"testing"

	"pgregory.net/rapid"
)

// --- Generators ---

func genMinMax() *rapid.Generator[MinMax] {
	return rapid.Custom(func(t *rapid.T) MinMax {
		min := rapid.Float64Range(0, 1000).Draw(t, "min")
		max := rapid.Float64Range(min, min+1000).Draw(t, "max")
		return MinMax{Min: min, Max: max}
	})
}

func genNonSingleMinMax() *rapid.Generator[MinMax] {
	return rapid.Custom(func(t *rapid.T) MinMax {
		min := rapid.Float64Range(0, 1000).Draw(t, "min")
		spread := rapid.Float64Range(1, 1000).Draw(t, "spread")
		return MinMax{Min: min, Max: min + spread}
	})
}

// --- NormLog ---

func TestRapidNormLog_OutputBounds(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		mm := genMinMax().Draw(t, "minMax")
		value := rapid.Float64Range(0, mm.Max*2+1).Draw(t, "value")

		result := NormLog(value, mm)

		if result < 0.0 || result > 1.0 {
			t.Fatalf("NormLog(%f, {%f,%f}) = %f, expected in [0,1]", value, mm.Min, mm.Max, result)
		}
	})
}

func TestRapidNormLog_Monotonicity(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		mm := genNonSingleMinMax().Draw(t, "minMax")
		x := rapid.Float64Range(mm.Min, mm.Max).Draw(t, "x")
		y := rapid.Float64Range(x, mm.Max).Draw(t, "y")

		resX := NormLog(x, mm)
		resY := NormLog(y, mm)

		if resY < resX-1e-10 {
			t.Fatalf("NormLog not monotonic: NormLog(%f)=%f > NormLog(%f)=%f for mm={%f,%f}",
				x, resX, y, resY, mm.Min, mm.Max)
		}
	})
}

func TestRapidNormLog_BoundaryMin(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		mm := genNonSingleMinMax().Draw(t, "minMax")

		result := NormLog(mm.Min, mm)

		if math.Abs(result) > 0.01 {
			t.Fatalf("NormLog(min=%f, {%f,%f}) = %f, expected ≈ 0", mm.Min, mm.Min, mm.Max, result)
		}
	})
}

func TestRapidNormLog_BoundaryMax(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		mm := genNonSingleMinMax().Draw(t, "minMax")

		result := NormLog(mm.Max, mm)

		if math.Abs(result-1.0) > 0.01 {
			t.Fatalf("NormLog(max=%f, {%f,%f}) = %f, expected ≈ 1", mm.Max, mm.Min, mm.Max, result)
		}
	})
}

// --- NormMinMax ---

func TestRapidNormMinMax_OutputBounds(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		mm := genMinMax().Draw(t, "minMax")
		value := rapid.Float64Range(0, mm.Max*2+1).Draw(t, "value")

		result := NormMinMax(value, mm)

		if result < 0.0 || result > 1.0 {
			t.Fatalf("NormMinMax(%f, {%f,%f}) = %f, expected in [0,1]", value, mm.Min, mm.Max, result)
		}
	})
}

func TestRapidNormMinMax_Monotonicity(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		mm := genNonSingleMinMax().Draw(t, "minMax")
		x := rapid.Float64Range(mm.Min, mm.Max).Draw(t, "x")
		y := rapid.Float64Range(x, mm.Max).Draw(t, "y")

		resX := NormMinMax(x, mm)
		resY := NormMinMax(y, mm)

		if resY < resX-1e-10 {
			t.Fatalf("NormMinMax not monotonic: NormMinMax(%f)=%f > NormMinMax(%f)=%f",
				x, resX, y, resY)
		}
	})
}

func TestRapidNormMinMax_Linearity(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		mm := genNonSingleMinMax().Draw(t, "minMax")
		value := rapid.Float64Range(mm.Min, mm.Max).Draw(t, "value")

		result := NormMinMax(value, mm)
		expected := (value - mm.Min) / mm.Range()

		if math.Abs(result-expected) > 1e-9 {
			t.Fatalf("NormMinMax(%f, {%f,%f}) = %f, expected %f (linearity)",
				value, mm.Min, mm.Max, result, expected)
		}
	})
}

// --- RecencyDecay ---

func TestRapidRecencyDecay_OutputBounds(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		days := rapid.Float64Range(0, 730).Draw(t, "days")
		halfLife := rapid.IntRange(1, 365).Draw(t, "halfLife")

		result := RecencyDecay(days, halfLife)

		if result < 0.0 || result > 1.0 {
			t.Fatalf("RecencyDecay(%f, %d) = %f, expected in [0,1]", days, halfLife, result)
		}
	})
}

func TestRapidRecencyDecay_MonotonicDecrease(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		halfLife := rapid.IntRange(1, 365).Draw(t, "halfLife")
		d1 := rapid.Float64Range(0, 365).Draw(t, "d1")
		d2 := rapid.Float64Range(d1, 730).Draw(t, "d2")

		res1 := RecencyDecay(d1, halfLife)
		res2 := RecencyDecay(d2, halfLife)

		if res2 > res1+1e-10 {
			t.Fatalf("RecencyDecay not monotonic decreasing: days=%f->%f, results=%f->%f",
				d1, d2, res1, res2)
		}
	})
}

func TestRapidRecencyDecay_HalfLifeProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		halfLife := rapid.IntRange(1, 365).Draw(t, "halfLife")

		result := RecencyDecay(float64(halfLife), halfLife)

		if math.Abs(result-0.5) > 0.01 {
			t.Fatalf("RecencyDecay(%d, %d) = %f, expected ≈ 0.5", halfLife, halfLife, result)
		}
	})
}

// --- Clamp ---

func TestRapidClamp_Idempotent(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		value := rapid.Float64Range(-100, 100).Draw(t, "value")

		once := Clamp(value)
		twice := Clamp(once)

		if once != twice {
			t.Fatalf("Clamp not idempotent: Clamp(%f)=%f, Clamp(Clamp(%f))=%f",
				value, once, value, twice)
		}
	})
}

func TestRapidClamp_IdentityOnUnitInterval(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		value := rapid.Float64Range(0, 1).Draw(t, "value")

		result := Clamp(value)

		if result != value {
			t.Fatalf("Clamp(%f) = %f, expected identity for value in [0,1]", value, result)
		}
	})
}
