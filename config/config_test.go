package config

import (
	"math"
	"testing"
)

func TestRiskThresholds_Classify(t *testing.T) {
	thresholds := RiskThresholds{High: 0.7, Medium: 0.4}

	tests := []struct {
		name     string
		score    float64
		expected RiskLevel
	}{
		{name: "High risk", score: 0.8, expected: RiskLevelHigh},
		{name: "High boundary", score: 0.7, expected: RiskLevelHigh},
		{name: "Just below high", score: 0.69, expected: RiskLevelMedium},
		{name: "Medium risk", score: 0.5, expected: RiskLevelMedium},
		{name: "Medium boundary", score: 0.4, expected: RiskLevelMedium},
		{name: "Just below medium", score: 0.39, expected: RiskLevelLow},
		{name: "Low risk", score: 0.3, expected: RiskLevelLow},
		{name: "Zero score", score: 0.0, expected: RiskLevelLow},
		{name: "Perfect score", score: 1.0, expected: RiskLevelHigh},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := thresholds.Classify(tt.score)
			if result != tt.expected {
				t.Errorf("Classify(%f) = %q, expected %q", tt.score, result, tt.expected)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Scoring.HalfLifeDays != 30 {
		t.Errorf("HalfLifeDays = %d, expected 30", cfg.Scoring.HalfLifeDays)
	}
	if cfg.Scoring.Weights.Commit != 0.20 {
		t.Errorf("Weights.Commit = %f, expected 0.20", cfg.Scoring.Weights.Commit)
	}
	if cfg.Scoring.Weights.Churn != 0.20 {
		t.Errorf("Weights.Churn = %f, expected 0.20", cfg.Scoring.Weights.Churn)
	}
	if cfg.Scoring.Weights.Recency != 0.15 {
		t.Errorf("Weights.Recency = %f, expected 0.15", cfg.Scoring.Weights.Recency)
	}
	if cfg.Scoring.Weights.Burst != 0.10 {
		t.Errorf("Weights.Burst = %f, expected 0.10", cfg.Scoring.Weights.Burst)
	}
	if cfg.Scoring.Weights.Ownership != 0.10 {
		t.Errorf("Weights.Ownership = %f, expected 0.10", cfg.Scoring.Weights.Ownership)
	}
	if cfg.Scoring.Weights.Bugfix != 0.15 {
		t.Errorf("Weights.Bugfix = %f, expected 0.15", cfg.Scoring.Weights.Bugfix)
	}
	if cfg.Scoring.Weights.Complexity != 0.10 {
		t.Errorf("Weights.Complexity = %f, expected 0.10", cfg.Scoring.Weights.Complexity)
	}
	if len(cfg.Bugfix.Patterns) != 4 {
		t.Errorf("Bugfix.Patterns length = %d, expected 4", len(cfg.Bugfix.Patterns))
	}
	if cfg.Burst.WindowDays != 7 {
		t.Errorf("Burst.WindowDays = %d, expected 7", cfg.Burst.WindowDays)
	}
	if cfg.CommitScoring.Thresholds.High != 0.7 {
		t.Errorf("Thresholds.High = %f, expected 0.7", cfg.CommitScoring.Thresholds.High)
	}
	if cfg.CommitScoring.Thresholds.Medium != 0.4 {
		t.Errorf("Thresholds.Medium = %f, expected 0.4", cfg.CommitScoring.Thresholds.Medium)
	}
	if cfg.CommitScoring.Weights.Diffusion != 0.35 {
		t.Errorf("CommitScoring.Weights.Diffusion = %f, expected 0.35", cfg.CommitScoring.Weights.Diffusion)
	}
	if cfg.CommitScoring.Weights.Size != 0.35 {
		t.Errorf("CommitScoring.Weights.Size = %f, expected 0.35", cfg.CommitScoring.Weights.Size)
	}
	if cfg.CommitScoring.Weights.Entropy != 0.30 {
		t.Errorf("CommitScoring.Weights.Entropy = %f, expected 0.30", cfg.CommitScoring.Weights.Entropy)
	}
	if cfg.Coupling.MinCoCommits != 3 {
		t.Errorf("Coupling.MinCoCommits = %d, expected 3", cfg.Coupling.MinCoCommits)
	}
	if cfg.Coupling.TopPairs != 50 {
		t.Errorf("Coupling.TopPairs = %d, expected 50", cfg.Coupling.TopPairs)
	}
	if cfg.Legacy.AnalysisWindowYears != 3 {
		t.Errorf("Legacy.AnalysisWindowYears = %d, expected 3", cfg.Legacy.AnalysisWindowYears)
	}
	if cfg.Legacy.MaxHotspots != 100 {
		t.Errorf("Legacy.MaxHotspots = %d, expected 100", cfg.Legacy.MaxHotspots)
	}
	if cfg.Legacy.DefaultBranch != "HEAD" {
		t.Errorf("Legacy.DefaultBranch = %q, expected %q", cfg.Legacy.DefaultBranch, "HEAD")
	}
}

func TestDefaultConfig_WeightsSum(t *testing.T) {
	cfg := DefaultConfig()

	// File scoring weights should sum to 1.0
	fileWeightsSum := cfg.Scoring.Weights.Commit +
		cfg.Scoring.Weights.Churn +
		cfg.Scoring.Weights.Recency +
		cfg.Scoring.Weights.Burst +
		cfg.Scoring.Weights.Ownership +
		cfg.Scoring.Weights.Bugfix +
		cfg.Scoring.Weights.Complexity

	if math.Abs(fileWeightsSum-1.0) > 0.001 {
		t.Errorf("File scoring weights sum = %f, expected 1.0", fileWeightsSum)
	}

	// Commit scoring weights should sum to 1.0
	commitWeightsSum := cfg.CommitScoring.Weights.Diffusion +
		cfg.CommitScoring.Weights.Size +
		cfg.CommitScoring.Weights.Entropy

	if math.Abs(commitWeightsSum-1.0) > 0.001 {
		t.Errorf("Commit scoring weights sum = %f, expected 1.0", commitWeightsSum)
	}
}
