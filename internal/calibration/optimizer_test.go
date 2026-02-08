package calibration

import (
	"math"
	"testing"
	"time"

	"github.com/masmgr/bugspots-go/config"
	"github.com/masmgr/bugspots-go/internal/aggregation"
)

func TestCalibrate_NoBugfixFiles(t *testing.T) {
	metrics := map[string]*aggregation.FileMetrics{
		"file1.go": {
			CommitCount:             5,
			AddedLines:              50,
			DeletedLines:            20,
			LastModifiedAt:          time.Now().Add(-7 * 24 * time.Hour),
			Contributors:            map[string]struct{}{"a": {}},
			ContributorCommitCounts: map[string]int{"a": 5},
			CommitTimes:             []time.Time{},
		},
	}

	result := Calibrate(CalibrateInput{
		Metrics:        metrics,
		BugfixFiles:    map[string]struct{}{},
		CurrentWeights: config.DefaultConfig().Scoring.Weights,
		HalfLifeDays:   30,
		Until:          time.Now(),
		TopPercent:     20,
	})

	if result.CurrentDetectionRate != 0 {
		t.Errorf("Expected 0 detection rate with no bugfix files, got %f", result.CurrentDetectionRate)
	}
	if result.BugfixFileCount != 0 {
		t.Errorf("Expected 0 bugfix file count, got %d", result.BugfixFileCount)
	}
}

func TestCalibrate_EmptyMetrics(t *testing.T) {
	result := Calibrate(CalibrateInput{
		Metrics:        map[string]*aggregation.FileMetrics{},
		BugfixFiles:    map[string]struct{}{"file1.go": {}},
		CurrentWeights: config.DefaultConfig().Scoring.Weights,
		HalfLifeDays:   30,
		Until:          time.Now(),
		TopPercent:     20,
	})

	if result.TotalFileCount != 0 {
		t.Errorf("Expected 0 total files, got %d", result.TotalFileCount)
	}
}

func TestCalibrate_AllBugfixFiles(t *testing.T) {
	now := time.Now()
	metrics := map[string]*aggregation.FileMetrics{
		"file1.go": {
			CommitCount:             5,
			AddedLines:              50,
			DeletedLines:            20,
			LastModifiedAt:          now.Add(-7 * 24 * time.Hour),
			Contributors:            map[string]struct{}{"a": {}},
			ContributorCommitCounts: map[string]int{"a": 5},
			BugfixCount:             3,
			CommitTimes:             []time.Time{},
		},
		"file2.go": {
			CommitCount:             3,
			AddedLines:              30,
			DeletedLines:            10,
			LastModifiedAt:          now.Add(-14 * 24 * time.Hour),
			Contributors:            map[string]struct{}{"b": {}},
			ContributorCommitCounts: map[string]int{"b": 3},
			BugfixCount:             2,
			CommitTimes:             []time.Time{},
		},
	}

	bugfixFiles := map[string]struct{}{
		"file1.go": {},
		"file2.go": {},
	}

	result := Calibrate(CalibrateInput{
		Metrics:        metrics,
		BugfixFiles:    bugfixFiles,
		CurrentWeights: config.DefaultConfig().Scoring.Weights,
		HalfLifeDays:   30,
		Until:          now,
		TopPercent:     100,
	})

	// With top 100%, all bugfix files should be detected
	if result.CurrentDetectionRate != 1.0 {
		t.Errorf("Expected 100%% detection rate with all files as bugfix and top 100%%, got %f", result.CurrentDetectionRate)
	}
}

func TestCalibrate_RecommendedImproves(t *testing.T) {
	now := time.Now()

	// Create a scenario where bugfix files have high bugfix count but low other metrics
	metrics := map[string]*aggregation.FileMetrics{}
	bugfixFiles := map[string]struct{}{}

	// 5 bugfix files with high bugfix count, low other metrics
	for i := 0; i < 5; i++ {
		path := "bugfix" + string(rune('a'+i)) + ".go"
		metrics[path] = &aggregation.FileMetrics{
			CommitCount:             2,
			AddedLines:              10,
			DeletedLines:            5,
			LastModifiedAt:          now.Add(-60 * 24 * time.Hour),
			Contributors:            map[string]struct{}{"a": {}},
			ContributorCommitCounts: map[string]int{"a": 2},
			BugfixCount:             10,
			CommitTimes:             []time.Time{},
		}
		bugfixFiles[path] = struct{}{}
	}

	// 20 non-bugfix files with high commit count, recent, but no bugfix
	for i := 0; i < 20; i++ {
		path := "clean" + string(rune('a'+i)) + ".go"
		metrics[path] = &aggregation.FileMetrics{
			CommitCount:             20,
			AddedLines:              200,
			DeletedLines:            100,
			LastModifiedAt:          now.Add(-2 * 24 * time.Hour),
			Contributors:            map[string]struct{}{"a": {}, "b": {}, "c": {}},
			ContributorCommitCounts: map[string]int{"a": 10, "b": 5, "c": 5},
			BurstScore:              0.8,
			BugfixCount:             0,
			CommitTimes:             []time.Time{},
		}
	}

	result := Calibrate(CalibrateInput{
		Metrics:        metrics,
		BugfixFiles:    bugfixFiles,
		CurrentWeights: config.DefaultConfig().Scoring.Weights,
		HalfLifeDays:   30,
		Until:          now,
		TopPercent:     20,
	})

	if result.TotalFileCount != 25 {
		t.Errorf("Expected 25 total files, got %d", result.TotalFileCount)
	}
	if result.BugfixFileCount != 5 {
		t.Errorf("Expected 5 bugfix files, got %d", result.BugfixFileCount)
	}

	// The optimizer should recommend higher bugfix weight since bugfix files have high bugfix count
	if result.RecommendedRate < result.CurrentDetectionRate {
		t.Errorf("Recommended rate %f should be >= current rate %f", result.RecommendedRate, result.CurrentDetectionRate)
	}
}

func TestDetectionRate(t *testing.T) {
	files := []fileFeatures{
		{path: "buggy1.go", features: [7]float64{0.9, 0, 0, 0, 0, 0, 0}, isBugfix: true},
		{path: "buggy2.go", features: [7]float64{0.8, 0, 0, 0, 0, 0, 0}, isBugfix: true},
		{path: "clean1.go", features: [7]float64{0.5, 0, 0, 0, 0, 0, 0}, isBugfix: false},
		{path: "clean2.go", features: [7]float64{0.3, 0, 0, 0, 0, 0, 0}, isBugfix: false},
		{path: "clean3.go", features: [7]float64{0.1, 0, 0, 0, 0, 0, 0}, isBugfix: false},
	}

	// All weight on commit dimension
	weights := [7]float64{1, 0, 0, 0, 0, 0, 0}

	// Top 40% = 2 files → both bugfix files are in top 2
	rate := detectionRate(files, weights, 40)
	if math.Abs(rate-1.0) > 0.001 {
		t.Errorf("Expected 100%% detection with top 40%%, got %f", rate)
	}

	// Top 20% = 1 file → only 1 of 2 bugfix files detected
	rate = detectionRate(files, weights, 20)
	if math.Abs(rate-0.5) > 0.001 {
		t.Errorf("Expected 50%% detection with top 20%%, got %f", rate)
	}
}

func TestWeightsToVec_VecToWeights_Roundtrip(t *testing.T) {
	original := config.WeightConfig{
		Commit:     0.20,
		Churn:      0.20,
		Recency:    0.15,
		Burst:      0.10,
		Ownership:  0.10,
		Bugfix:     0.15,
		Complexity: 0.10,
	}

	vec := weightsToVec(original)
	result := vecToWeights(vec)

	if math.Abs(result.Commit-original.Commit) > 0.001 {
		t.Errorf("Commit mismatch: %f != %f", result.Commit, original.Commit)
	}
	if math.Abs(result.Complexity-original.Complexity) > 0.001 {
		t.Errorf("Complexity mismatch: %f != %f", result.Complexity, original.Complexity)
	}
}

func TestRoundTo(t *testing.T) {
	tests := []struct {
		val      float64
		decimals int
		expected float64
	}{
		{0.1234, 2, 0.12},
		{0.1267, 2, 0.13},
		{0.5, 2, 0.50},
		{0.0, 2, 0.0},
	}

	for _, tt := range tests {
		result := roundTo(tt.val, tt.decimals)
		if math.Abs(result-tt.expected) > 0.001 {
			t.Errorf("roundTo(%f, %d) = %f, expected %f", tt.val, tt.decimals, result, tt.expected)
		}
	}
}
