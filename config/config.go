package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config is the root configuration structure.
type Config struct {
	Scoring       ScoringConfig       `json:"scoring"`
	Burst         BurstConfig         `json:"burst"`
	Bugfix        BugfixConfig        `json:"bugfix"`
	CommitScoring CommitScoringConfig `json:"commitScoring"`
	Coupling      CouplingConfig      `json:"coupling"`
	Filters       FilterConfig        `json:"filters"`
	Legacy        LegacyConfig        `json:"legacy"`
}

// BugfixConfig holds bugfix detection configuration.
type BugfixConfig struct {
	Patterns []string `json:"patterns"` // Regex patterns for bugfix commit detection
}

// LegacyConfig holds configuration for legacy bugspots scan mode.
type LegacyConfig struct {
	AnalysisWindowYears int    `json:"analysisWindowYears"` // Default: 3
	MaxHotspots         int    `json:"maxHotspots"`         // Default: 100
	DefaultBranch       string `json:"defaultBranch"`       // Default: "HEAD"
	DefaultBugfixRegex  string `json:"defaultBugfixRegex"`  // Default: \b(fix(es|ed)?|close(s|d)?)\b
}

// ScoringConfig holds file hotspot scoring configuration.
type ScoringConfig struct {
	HalfLifeDays int          `json:"halfLifeDays"`
	Weights      WeightConfig `json:"weights"`
}

// WeightConfig holds weights for multi-factor scoring.
type WeightConfig struct {
	Commit     float64 `json:"commit"`
	Churn      float64 `json:"churn"`
	Recency    float64 `json:"recency"`
	Burst      float64 `json:"burst"`
	Ownership  float64 `json:"ownership"`
	Bugfix     float64 `json:"bugfix"`
	Complexity float64 `json:"complexity"`
}

// BurstConfig holds burst calculation options.
type BurstConfig struct {
	WindowDays int `json:"windowDays"`
}

// CommitScoringConfig holds JIT commit risk scoring configuration.
type CommitScoringConfig struct {
	Weights    CommitWeightConfig `json:"weights"`
	Thresholds RiskThresholds     `json:"thresholds"`
}

// CommitWeightConfig holds weights for commit risk scoring.
type CommitWeightConfig struct {
	Diffusion float64 `json:"diffusion"`
	Size      float64 `json:"size"`
	Entropy   float64 `json:"entropy"`
}

// RiskThresholds for risk level classification.
type RiskThresholds struct {
	High   float64 `json:"high"`
	Medium float64 `json:"medium"`
}

func DefaultRiskThresholds() RiskThresholds {
	return RiskThresholds{
		High:   0.7,
		Medium: 0.4,
	}
}

// Classify returns the risk level for a given score.
func (t RiskThresholds) Classify(score float64) RiskLevel {
	if score >= t.High {
		return RiskLevelHigh
	}
	if score >= t.Medium {
		return RiskLevelMedium
	}
	return RiskLevelLow
}

// RiskLevel represents the risk classification.
type RiskLevel string

const (
	RiskLevelHigh   RiskLevel = "high"
	RiskLevelMedium RiskLevel = "medium"
	RiskLevelLow    RiskLevel = "low"
)

// CouplingConfig holds coupling analysis options.
type CouplingConfig struct {
	MinCoCommits        int     `json:"minCoCommits"`
	MinJaccardThreshold float64 `json:"minJaccardThreshold"`
	MaxFilesPerCommit   int     `json:"maxFilesPerCommit"`
	TopPairs            int     `json:"topPairs"`
}

// FilterConfig holds file path filtering options.
type FilterConfig struct {
	Include []string `json:"include"`
	Exclude []string `json:"exclude"`
}

// DefaultConfig returns a configuration with default values.
func DefaultConfig() *Config {
	return &Config{
		Scoring: ScoringConfig{
			HalfLifeDays: 30,
			Weights: WeightConfig{
				Commit:     0.20,
				Churn:      0.20,
				Recency:    0.15,
				Burst:      0.10,
				Ownership:  0.10,
				Bugfix:     0.15,
				Complexity: 0.10,
			},
		},
		Burst: BurstConfig{
			WindowDays: 7,
		},
		Bugfix: BugfixConfig{
			Patterns: []string{
				`\bfix(ed|es)?\b`,
				`\bbug\b`,
				`\bhotfix\b`,
				`\bpatch\b`,
			},
		},
		CommitScoring: CommitScoringConfig{
			Weights: CommitWeightConfig{
				Diffusion: 0.35,
				Size:      0.35,
				Entropy:   0.30,
			},
			Thresholds: DefaultRiskThresholds(),
		},
		Coupling: CouplingConfig{
			MinCoCommits:        3,
			MinJaccardThreshold: 0.1,
			MaxFilesPerCommit:   50,
			TopPairs:            50,
		},
		Filters: FilterConfig{
			Include: []string{},
			Exclude: []string{},
		},
		Legacy: LegacyConfig{
			AnalysisWindowYears: 3,
			MaxHotspots:         100,
			DefaultBranch:       "HEAD",
			DefaultBugfixRegex:  `\b(fix(es|ed)?|close(s|d)?)\b`,
		},
	}
}

// LoadConfig loads configuration from a file, merging with defaults.
func LoadConfig(path string) (*Config, error) {
	cfg := DefaultConfig()

	if path == "" {
		// Try default locations
		candidates := []string{".bugspots.json"}
		if home, err := os.UserHomeDir(); err == nil && home != "" {
			candidates = append(candidates, filepath.Join(home, ".bugspots.json"))
		} else if envHome := os.Getenv("HOME"); envHome != "" {
			candidates = append(candidates, filepath.Join(envHome, ".bugspots.json"))
		}
		for _, p := range candidates {
			if _, err := os.Stat(p); err == nil {
				path = p
				break
			}
		}
	}

	if path == "" {
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// SaveConfig saves configuration to a file.
func SaveConfig(cfg *Config, path string) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
