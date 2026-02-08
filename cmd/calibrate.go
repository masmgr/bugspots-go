package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/masmgr/bugspots-go/config"
	"github.com/masmgr/bugspots-go/internal/aggregation"
	"github.com/masmgr/bugspots-go/internal/burst"
	"github.com/masmgr/bugspots-go/internal/calibration"
	"github.com/masmgr/bugspots-go/internal/git"
	"github.com/urfave/cli/v2"
)

// CalibrateCmd returns the calibrate command.
func CalibrateCmd() *cli.Command {
	flags := append(commonFlags(),
		&cli.IntFlag{
			Name:  "half-life",
			Usage: "Half-life in days for recency decay",
			Value: 30,
		},
		&cli.IntFlag{
			Name:  "window-days",
			Usage: "Window size in days for burst detection",
			Value: 7,
		},
		&cli.StringSliceFlag{
			Name:  "bug-patterns",
			Usage: "Regex patterns for bugfix commit detection (can be specified multiple times)",
		},
		&cli.IntFlag{
			Name:  "top-percent",
			Usage: "Top N% threshold for recall calculation",
			Value: 20,
		},
	)

	return &cli.Command{
		Name:   "calibrate",
		Usage:  "Calibrate scoring weights based on historical bugfix data",
		Flags:  flags,
		Action: calibrateAction,
	}
}

func calibrateAction(c *cli.Context) error {
	return executeWithContext(c, git.ChangeDetailFull, func(ctx *CommandContext, c *cli.Context) error {
		// Aggregate file metrics
		aggregator := aggregation.NewFileMetricsAggregator()
		metrics := aggregator.Process(ctx.ChangeSets)

		// Detect bugfix commits
		bugPatterns := resolveBugPatterns(c, ctx.Config)
		if len(bugPatterns) == 0 {
			return fmt.Errorf("no bugfix patterns configured; use --bug-patterns or configure in .bugspots.json")
		}
		result, err := detectAndApplyBugfixes(ctx.ChangeSets, metrics, aggregator, bugPatterns)
		if err != nil {
			return err
		}

		if result.TotalBugfixes == 0 {
			fmt.Println("No bugfix commits found. Cannot calibrate weights.")
			fmt.Println("Consider adjusting --bug-patterns or --since to include more history.")
			return nil
		}

		// Calculate burst scores
		burstCalc := burst.NewCalculator(ctx.Config.Burst.WindowDays)
		burstCalc.Compute(metrics)

		// Build bugfix file set
		bugfixFiles := make(map[string]struct{})
		for path := range result.FileBugfixCounts {
			canonical := aggregator.CanonicalPath(path)
			if _, ok := metrics[canonical]; ok {
				bugfixFiles[canonical] = struct{}{}
			}
		}

		// Run calibration
		calResult := calibration.Calibrate(calibration.CalibrateInput{
			Metrics:        metrics,
			BugfixFiles:    bugfixFiles,
			CurrentWeights: ctx.Config.Scoring.Weights,
			HalfLifeDays:   ctx.Config.Scoring.HalfLifeDays,
			Until:          ctx.Until,
			TopPercent:     c.Int("top-percent"),
		})

		// Display results
		printCalibrationResult(calResult, c.Int("top-percent"))

		return nil
	})
}

func printCalibrationResult(result calibration.CalibrateResult, topPercent int) {
	color.Green("Calibration Results (based on %d bugfix files out of %d total files):",
		result.BugfixFileCount, result.TotalFileCount)
	fmt.Println()

	fmt.Printf("Current weights detection rate (top %d%%): %.1f%%\n\n",
		topPercent, result.CurrentDetectionRate*100)

	names := calibration.WeightNames()
	cur := weightsToSlice(result.CurrentWeights)
	rec := weightsToSlice(result.RecommendedWeights)

	fmt.Println("Recommended weights:")
	for i, name := range names {
		marker := ""
		if rec[i] > cur[i]+0.01 {
			marker = color.CyanString("  ^")
		} else if rec[i] < cur[i]-0.01 {
			marker = color.YellowString("  v")
		}
		fmt.Printf("  %-12s %.2f (current: %.2f)%s\n", name+":", rec[i], cur[i], marker)
	}

	fmt.Printf("\nExpected detection rate with recommended weights (top %d%%): %.1f%%\n",
		topPercent, result.RecommendedRate*100)

	if result.RecommendedRate > result.CurrentDetectionRate {
		improvement := (result.RecommendedRate - result.CurrentDetectionRate) * 100
		color.Green("\nImprovement: +%.1f percentage points", improvement)
	} else {
		color.Green("\nCurrent weights are already optimal for this dataset.")
	}
}

func weightsToSlice(w config.WeightConfig) [7]float64 {
	return [7]float64{w.Commit, w.Churn, w.Recency, w.Burst, w.Ownership, w.Bugfix, w.Complexity}
}
