package scoring

import (
	"sort"

	"github.com/masmgr/bugspots-go/config"
	"github.com/masmgr/bugspots-go/internal/aggregation"
)

// CommitRiskItem represents a commit with its calculated risk score.
type CommitRiskItem struct {
	Metrics   aggregation.CommitMetrics
	RiskScore float64
	RiskLevel config.RiskLevel
	Breakdown *CommitRiskBreakdown
}

// CommitRiskBreakdown shows the contribution of each component to the total score.
type CommitRiskBreakdown struct {
	DiffusionComponent float64
	SizeComponent      float64
	EntropyComponent   float64
}

// CommitNormalizationContext holds the min/max values needed for normalization.
type CommitNormalizationContext struct {
	FileCount      MinMax
	DirectoryCount MinMax
	SubsystemCount MinMax
	TotalChurn     MinMax
}

// CommitContextFromMetrics computes the normalization context from commit metrics.
func CommitContextFromMetrics(metrics []aggregation.CommitMetrics) CommitNormalizationContext {
	ctx := CommitNormalizationContext{
		FileCount:      MinMax{Min: 0, Max: 0},
		DirectoryCount: MinMax{Min: 0, Max: 0},
		SubsystemCount: MinMax{Min: 0, Max: 0},
		TotalChurn:     MinMax{Min: 0, Max: 0},
	}

	if len(metrics) == 0 {
		return ctx
	}

	first := true
	for _, cm := range metrics {
		fileCount := float64(cm.FileCount)
		dirCount := float64(cm.DirectoryCount)
		subCount := float64(cm.SubsystemCount)
		churn := float64(cm.TotalChurn())

		if first {
			ctx.FileCount = MinMax{Min: fileCount, Max: fileCount}
			ctx.DirectoryCount = MinMax{Min: dirCount, Max: dirCount}
			ctx.SubsystemCount = MinMax{Min: subCount, Max: subCount}
			ctx.TotalChurn = MinMax{Min: churn, Max: churn}
			first = false
			continue
		}

		if fileCount < ctx.FileCount.Min {
			ctx.FileCount.Min = fileCount
		}
		if fileCount > ctx.FileCount.Max {
			ctx.FileCount.Max = fileCount
		}
		if dirCount < ctx.DirectoryCount.Min {
			ctx.DirectoryCount.Min = dirCount
		}
		if dirCount > ctx.DirectoryCount.Max {
			ctx.DirectoryCount.Max = dirCount
		}
		if subCount < ctx.SubsystemCount.Min {
			ctx.SubsystemCount.Min = subCount
		}
		if subCount > ctx.SubsystemCount.Max {
			ctx.SubsystemCount.Max = subCount
		}
		if churn < ctx.TotalChurn.Min {
			ctx.TotalChurn.Min = churn
		}
		if churn > ctx.TotalChurn.Max {
			ctx.TotalChurn.Max = churn
		}
	}

	return ctx
}

// CommitScorer calculates risk scores for commits based on their metrics.
type CommitScorer struct {
	options config.CommitScoringConfig
}

// NewCommitScorer creates a new commit scorer with the given options.
func NewCommitScorer(options config.CommitScoringConfig) *CommitScorer {
	return &CommitScorer{options: options}
}

// ScoreAndRank scores all commits and returns them sorted by risk score (descending).
func (s *CommitScorer) ScoreAndRank(
	metrics []aggregation.CommitMetrics,
	explain bool,
) []CommitRiskItem {
	if len(metrics) == 0 {
		return nil
	}

	ctx := CommitContextFromMetrics(metrics)
	weights := s.options.Weights
	thresholds := s.options.Thresholds

	items := make([]CommitRiskItem, 0, len(metrics))

	for _, cm := range metrics {
		// Calculate diffusion component (average of NF, ND, NS normalized)
		nfNorm := NormLog(float64(cm.FileCount), ctx.FileCount)
		ndNorm := NormLog(float64(cm.DirectoryCount), ctx.DirectoryCount)
		nsNorm := NormLog(float64(cm.SubsystemCount), ctx.SubsystemCount)
		diffusionComponent := weights.Diffusion * ((nfNorm + ndNorm + nsNorm) / 3.0)

		// Calculate size component (log-normalized churn)
		sizeComponent := weights.Size * NormLog(float64(cm.TotalChurn()), ctx.TotalChurn)

		// Entropy is already normalized (0-1), just apply weight
		entropyComponent := weights.Entropy * cm.ChangeEntropy

		// Calculate total score
		totalScore := diffusionComponent + sizeComponent + entropyComponent

		// Clamp to [0, 1]
		totalScore = Clamp(totalScore)

		riskLevel := thresholds.Classify(totalScore)

		var breakdown *CommitRiskBreakdown
		if explain {
			breakdown = &CommitRiskBreakdown{
				DiffusionComponent: diffusionComponent,
				SizeComponent:      sizeComponent,
				EntropyComponent:   entropyComponent,
			}
		}

		items = append(items, CommitRiskItem{
			Metrics:   cm,
			RiskScore: totalScore,
			RiskLevel: riskLevel,
			Breakdown: breakdown,
		})
	}

	// Sort by risk score descending
	sort.Slice(items, func(i, j int) bool {
		return items[i].RiskScore > items[j].RiskScore
	})

	return items
}

// FilterByRiskLevel filters commit risk items by minimum risk level.
func FilterByRiskLevel(items []CommitRiskItem, level config.RiskLevel) []CommitRiskItem {
	if level == "" {
		return items
	}

	result := make([]CommitRiskItem, 0)
	for _, item := range items {
		switch level {
		case config.RiskLevelHigh:
			if item.RiskLevel == config.RiskLevelHigh {
				result = append(result, item)
			}
		case config.RiskLevelMedium:
			if item.RiskLevel == config.RiskLevelHigh || item.RiskLevel == config.RiskLevelMedium {
				result = append(result, item)
			}
		default:
			result = append(result, item)
		}
	}
	return result
}
