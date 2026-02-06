package scoring

import (
	"sort"
	"time"

	"github.com/masmgr/bugspots-go/config"
	"github.com/masmgr/bugspots-go/internal/aggregation"
)

// FileRiskItem represents a file with its calculated risk score.
type FileRiskItem struct {
	Path      string
	RiskScore float64
	Metrics   *aggregation.FileMetrics
	Breakdown *ScoreBreakdown
}

// ScoreBreakdown shows the contribution of each component to the total score.
type ScoreBreakdown struct {
	CommitComponent    float64
	ChurnComponent     float64
	RecencyComponent   float64
	BurstComponent     float64
	OwnershipComponent float64
	BugfixComponent    float64
}

// NormalizationContext holds the min/max values needed for normalization.
type NormalizationContext struct {
	CommitCount MinMax
	ChurnTotal  MinMax
	BugfixCount MinMax
}

// FromMetrics computes the normalization context from file metrics.
func FromMetrics(metrics map[string]*aggregation.FileMetrics) NormalizationContext {
	ctx := NormalizationContext{
		CommitCount: MinMax{Min: 0, Max: 0},
		ChurnTotal:  MinMax{Min: 0, Max: 0},
		BugfixCount: MinMax{Min: 0, Max: 0},
	}

	first := true
	for _, fm := range metrics {
		commitCount := float64(fm.CommitCount)
		churnTotal := float64(fm.ChurnTotal())
		bugfixCount := float64(fm.BugfixCount)

		if first {
			ctx.CommitCount = MinMax{Min: commitCount, Max: commitCount}
			ctx.ChurnTotal = MinMax{Min: churnTotal, Max: churnTotal}
			ctx.BugfixCount = MinMax{Min: bugfixCount, Max: bugfixCount}
			first = false
			continue
		}

		if commitCount < ctx.CommitCount.Min {
			ctx.CommitCount.Min = commitCount
		}
		if commitCount > ctx.CommitCount.Max {
			ctx.CommitCount.Max = commitCount
		}
		if churnTotal < ctx.ChurnTotal.Min {
			ctx.ChurnTotal.Min = churnTotal
		}
		if churnTotal > ctx.ChurnTotal.Max {
			ctx.ChurnTotal.Max = churnTotal
		}
		if bugfixCount < ctx.BugfixCount.Min {
			ctx.BugfixCount.Min = bugfixCount
		}
		if bugfixCount > ctx.BugfixCount.Max {
			ctx.BugfixCount.Max = bugfixCount
		}
	}

	return ctx
}

// FileScorer calculates risk scores for files based on their metrics.
type FileScorer struct {
	options config.ScoringConfig
}

// NewFileScorer creates a new file scorer with the given options.
func NewFileScorer(options config.ScoringConfig) *FileScorer {
	return &FileScorer{options: options}
}

// ScoreAndRank scores all files and returns them sorted by risk score (descending).
func (s *FileScorer) ScoreAndRank(
	metrics map[string]*aggregation.FileMetrics,
	explain bool,
	until time.Time,
) []FileRiskItem {
	if len(metrics) == 0 {
		return nil
	}

	ctx := FromMetrics(metrics)
	weights := s.options.Weights
	halfLifeDays := s.options.HalfLifeDays

	items := make([]FileRiskItem, 0, len(metrics))

	for path, fm := range metrics {
		// Calculate commit frequency component
		commitComponent := weights.Commit * NormLog(float64(fm.CommitCount), ctx.CommitCount)

		// Calculate code churn component
		churnComponent := weights.Churn * NormLog(float64(fm.ChurnTotal()), ctx.ChurnTotal)

		// Calculate recency component
		daysSinceModified := until.Sub(fm.LastModifiedAt).Hours() / 24
		recencyComponent := weights.Recency * RecencyDecay(daysSinceModified, halfLifeDays)

		// Calculate burst component
		burstComponent := weights.Burst * fm.BurstScore

		// Calculate ownership component
		// Low ownership ratio = high risk (dispersed ownership)
		// Invert the ratio: 1.0 - ratio so that lower ownership ratio gives higher score
		ownershipComponent := weights.Ownership * (1.0 - fm.OwnershipRatio())

		// Calculate bugfix component
		bugfixComponent := weights.Bugfix * NormLog(float64(fm.BugfixCount), ctx.BugfixCount)

		// Calculate total score
		totalScore := commitComponent + churnComponent + recencyComponent +
			burstComponent + ownershipComponent + bugfixComponent

		var breakdown *ScoreBreakdown
		if explain {
			breakdown = &ScoreBreakdown{
				CommitComponent:    commitComponent,
				ChurnComponent:     churnComponent,
				RecencyComponent:   recencyComponent,
				BurstComponent:     burstComponent,
				OwnershipComponent: ownershipComponent,
				BugfixComponent:    bugfixComponent,
			}
		}

		items = append(items, FileRiskItem{
			Path:      path,
			RiskScore: totalScore,
			Metrics:   fm,
			Breakdown: breakdown,
		})
	}

	// Sort by risk score descending
	sort.Slice(items, func(i, j int) bool {
		return items[i].RiskScore > items[j].RiskScore
	})

	return items
}
