package calibration

import (
	"math"
	"sort"
	"time"

	"github.com/masmgr/bugspots-go/config"
	"github.com/masmgr/bugspots-go/internal/aggregation"
	"github.com/masmgr/bugspots-go/internal/scoring"
)

// CalibrateInput holds the input data for calibration.
type CalibrateInput struct {
	Metrics        map[string]*aggregation.FileMetrics
	BugfixFiles    map[string]struct{} // Files touched by bugfix commits
	CurrentWeights config.WeightConfig
	HalfLifeDays   int
	Until          time.Time
	TopPercent     int // Top N% threshold for recall calculation (default: 20)
}

// CalibrateResult holds the output of calibration.
type CalibrateResult struct {
	CurrentWeights       config.WeightConfig
	CurrentDetectionRate float64
	RecommendedWeights   config.WeightConfig
	RecommendedRate      float64
	BugfixFileCount      int
	TotalFileCount       int
}

// fileFeatures holds normalized feature values for a single file.
type fileFeatures struct {
	path     string
	features [7]float64 // commit, churn, recency, burst, ownership, bugfix, complexity
	isBugfix bool
}

// weightNames defines the order of weight components for the optimizer.
var weightNames = [7]string{"commit", "churn", "recency", "burst", "ownership", "bugfix", "complexity"}

// Calibrate optimizes scoring weights to maximize detection of bugfix files.
func Calibrate(input CalibrateInput) CalibrateResult {
	if len(input.Metrics) == 0 || len(input.BugfixFiles) == 0 {
		return CalibrateResult{
			CurrentWeights:       input.CurrentWeights,
			CurrentDetectionRate: 0,
			RecommendedWeights:   input.CurrentWeights,
			RecommendedRate:      0,
			BugfixFileCount:      len(input.BugfixFiles),
			TotalFileCount:       len(input.Metrics),
		}
	}

	topPercent := input.TopPercent
	if topPercent <= 0 {
		topPercent = 20
	}

	// Compute normalization context
	ctx := scoring.FromMetrics(input.Metrics)

	// Build feature matrix: one row per file with normalized metric values
	halfLife := input.HalfLifeDays
	if halfLife <= 0 {
		halfLife = 30
	}

	files := make([]fileFeatures, 0, len(input.Metrics))
	for path, fm := range input.Metrics {
		daysSince := input.Until.Sub(fm.LastModifiedAt).Hours() / 24

		ff := fileFeatures{
			path: path,
			features: [7]float64{
				scoring.NormLog(float64(fm.CommitCount), ctx.CommitCount),
				scoring.NormLog(float64(fm.ChurnTotal()), ctx.ChurnTotal),
				scoring.RecencyDecay(daysSince, halfLife),
				fm.BurstScore,
				1.0 - fm.OwnershipRatio(),
				scoring.NormLog(float64(fm.BugfixCount), ctx.BugfixCount),
				scoring.NormLog(float64(fm.FileSize), ctx.FileSize),
			},
		}
		_, ff.isBugfix = input.BugfixFiles[path]
		files = append(files, ff)
	}

	// Compute current detection rate
	currentWeightVec := weightsToVec(input.CurrentWeights)
	currentRate := detectionRate(files, currentWeightVec, topPercent)

	// Optimize weights using coordinate descent
	bestWeights := optimizeWeights(files, topPercent)
	bestRate := detectionRate(files, bestWeights, topPercent)

	// If optimization didn't improve, keep current
	if bestRate <= currentRate {
		return CalibrateResult{
			CurrentWeights:       input.CurrentWeights,
			CurrentDetectionRate: currentRate,
			RecommendedWeights:   input.CurrentWeights,
			RecommendedRate:      currentRate,
			BugfixFileCount:      len(input.BugfixFiles),
			TotalFileCount:       len(input.Metrics),
		}
	}

	return CalibrateResult{
		CurrentWeights:       input.CurrentWeights,
		CurrentDetectionRate: currentRate,
		RecommendedWeights:   vecToWeights(bestWeights),
		RecommendedRate:      bestRate,
		BugfixFileCount:      len(input.BugfixFiles),
		TotalFileCount:       len(input.Metrics),
	}
}

// detectionRate calculates the recall of bugfix files in the top N% of ranked files.
func detectionRate(files []fileFeatures, weights [7]float64, topPercent int) float64 {
	type scored struct {
		score    float64
		isBugfix bool
	}

	scoredFiles := make([]scored, len(files))
	for i, f := range files {
		var s float64
		for j := range weights {
			s += weights[j] * f.features[j]
		}
		scoredFiles[i] = scored{score: s, isBugfix: f.isBugfix}
	}

	// Sort by score descending
	sort.Slice(scoredFiles, func(i, j int) bool {
		return scoredFiles[i].score > scoredFiles[j].score
	})

	// Count bugfix files in top N%
	topN := int(math.Ceil(float64(len(scoredFiles)) * float64(topPercent) / 100.0))
	if topN > len(scoredFiles) {
		topN = len(scoredFiles)
	}

	bugfixInTop := 0
	totalBugfix := 0
	for i, sf := range scoredFiles {
		if sf.isBugfix {
			totalBugfix++
			if i < topN {
				bugfixInTop++
			}
		}
	}

	if totalBugfix == 0 {
		return 0
	}

	return float64(bugfixInTop) / float64(totalBugfix)
}

// optimizeWeights uses coordinate descent to find weights that maximize detection rate.
func optimizeWeights(files []fileFeatures, topPercent int) [7]float64 {
	// Start with equal weights
	weights := [7]float64{1.0 / 7, 1.0 / 7, 1.0 / 7, 1.0 / 7, 1.0 / 7, 1.0 / 7, 1.0 / 7}
	bestRate := detectionRate(files, weights, topPercent)

	step := 0.05
	maxIterations := 100

	for iter := 0; iter < maxIterations; iter++ {
		improved := false

		for i := 0; i < 7; i++ {
			for j := 0; j < 7; j++ {
				if i == j {
					continue
				}

				// Try transferring weight from j to i
				if weights[j] < step {
					continue
				}

				trial := weights
				trial[i] += step
				trial[j] -= step

				// Ensure non-negative
				if trial[j] < 0 {
					continue
				}

				rate := detectionRate(files, trial, topPercent)
				if rate > bestRate+1e-10 {
					weights = trial
					bestRate = rate
					improved = true
				}
			}
		}

		if !improved {
			if step > 0.01 {
				step /= 2
			} else {
				break
			}
		}
	}

	return weights
}

// weightsToVec converts WeightConfig to an array.
func weightsToVec(w config.WeightConfig) [7]float64 {
	return [7]float64{w.Commit, w.Churn, w.Recency, w.Burst, w.Ownership, w.Bugfix, w.Complexity}
}

// vecToWeights converts an array to WeightConfig.
func vecToWeights(v [7]float64) config.WeightConfig {
	return config.WeightConfig{
		Commit:     roundTo(v[0], 2),
		Churn:      roundTo(v[1], 2),
		Recency:    roundTo(v[2], 2),
		Burst:      roundTo(v[3], 2),
		Ownership:  roundTo(v[4], 2),
		Bugfix:     roundTo(v[5], 2),
		Complexity: roundTo(v[6], 2),
	}
}

func roundTo(val float64, decimals int) float64 {
	pow := math.Pow(10, float64(decimals))
	return math.Round(val*pow) / pow
}

// WeightNames returns the ordered weight component names (for display).
func WeightNames() [7]string {
	return weightNames
}
