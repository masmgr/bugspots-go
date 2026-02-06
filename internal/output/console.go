package output

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/fatih/color"
)

// ConsoleFileWriter writes file analysis reports to the console.
type ConsoleFileWriter struct{}

// Write outputs the file analysis report to the console.
func (w *ConsoleFileWriter) Write(report *FileAnalysisReport, options OutputOptions) error {
	items := report.Items
	if options.Top > 0 && options.Top < len(items) {
		items = items[:options.Top]
	}

	color.Green("File Hotspot Analysis Results")
	fmt.Printf("Repository: %s\n", report.RepoPath)
	if report.Since != nil {
		fmt.Printf("Period: %s to %s\n", report.Since.Format("2006-01-02"), report.Until.Format("2006-01-02"))
	} else {
		fmt.Printf("Until: %s\n", report.Until.Format("2006-01-02"))
	}
	fmt.Printf("Total files analyzed: %d\n\n", len(report.Items))

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	// Write header
	if options.Explain {
		fmt.Fprintln(tw, "#\tPath\tScore\tCommits\tChurn\tContributors\tBurst\tC\tCh\tR\tB\tO")
	} else {
		fmt.Fprintln(tw, "#\tPath\tScore\tCommits\tChurn\tContributors\tBurst")
	}

	// Write rows
	for i, item := range items {
		if options.Explain && item.Breakdown != nil {
			fmt.Fprintf(tw, "%d\t%s\t%.4f\t%d\t%d\t%d\t%.2f\t%.3f\t%.3f\t%.3f\t%.3f\t%.3f\n",
				i+1,
				item.Path,
				item.RiskScore,
				item.Metrics.CommitCount,
				item.Metrics.ChurnTotal(),
				item.Metrics.ContributorCount(),
				item.Metrics.BurstScore,
				item.Breakdown.CommitComponent,
				item.Breakdown.ChurnComponent,
				item.Breakdown.RecencyComponent,
				item.Breakdown.BurstComponent,
				item.Breakdown.OwnershipComponent,
			)
		} else {
			fmt.Fprintf(tw, "%d\t%s\t%.4f\t%d\t%d\t%d\t%.2f\n",
				i+1,
				item.Path,
				item.RiskScore,
				item.Metrics.CommitCount,
				item.Metrics.ChurnTotal(),
				item.Metrics.ContributorCount(),
				item.Metrics.BurstScore,
			)
		}
	}

	tw.Flush()

	if options.Explain {
		fmt.Println("\nScore breakdown: C=Commit, Ch=Churn, R=Recency, B=Burst, O=Ownership")
	}

	return nil
}

// ConsoleCommitWriter writes commit analysis reports to the console.
type ConsoleCommitWriter struct{}

// Write outputs the commit analysis report to the console.
func (w *ConsoleCommitWriter) Write(report *CommitAnalysisReport, options OutputOptions) error {
	items := report.Items
	if options.Top > 0 && options.Top < len(items) {
		items = items[:options.Top]
	}

	color.Green("Commit Risk Analysis Results")
	fmt.Printf("Repository: %s\n", report.RepoPath)
	if report.Since != nil {
		fmt.Printf("Period: %s to %s\n", report.Since.Format("2006-01-02"), report.Until.Format("2006-01-02"))
	} else {
		fmt.Printf("Until: %s\n", report.Until.Format("2006-01-02"))
	}
	fmt.Printf("Total commits analyzed: %d\n\n", len(report.Items))

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	// Write header
	if options.Explain {
		fmt.Fprintln(tw, "#\tSHA\tScore\tLevel\tFiles\tChurn\tEntropy\tMessage\tD\tS\tE")
	} else {
		fmt.Fprintln(tw, "#\tSHA\tScore\tLevel\tFiles\tChurn\tEntropy\tMessage")
	}

	// Write rows
	for i, item := range items {
		levelColor := getLevelColor(string(item.RiskLevel))
		if options.Explain && item.Breakdown != nil {
			fmt.Fprintf(tw, "%d\t%s\t%.4f\t%s\t%d\t%d\t%.2f\t%s\t%.3f\t%.3f\t%.3f\n",
				i+1,
				item.Metrics.SHA[:8],
				item.RiskScore,
				levelColor(string(item.RiskLevel)),
				item.Metrics.FileCount,
				item.Metrics.TotalChurn(),
				item.Metrics.ChangeEntropy,
				truncateMessage(item.Metrics.Message, 40),
				item.Breakdown.DiffusionComponent,
				item.Breakdown.SizeComponent,
				item.Breakdown.EntropyComponent,
			)
		} else {
			fmt.Fprintf(tw, "%d\t%s\t%.4f\t%s\t%d\t%d\t%.2f\t%s\n",
				i+1,
				item.Metrics.SHA[:8],
				item.RiskScore,
				levelColor(string(item.RiskLevel)),
				item.Metrics.FileCount,
				item.Metrics.TotalChurn(),
				item.Metrics.ChangeEntropy,
				truncateMessage(item.Metrics.Message, 40),
			)
		}
	}

	tw.Flush()

	if options.Explain {
		fmt.Println("\nScore breakdown: D=Diffusion, S=Size, E=Entropy")
	}

	return nil
}

// ConsoleCouplingWriter writes coupling analysis reports to the console.
type ConsoleCouplingWriter struct{}

// Write outputs the coupling analysis report to the console.
func (w *ConsoleCouplingWriter) Write(report *CouplingAnalysisReport, options OutputOptions) error {
	result := report.Result

	color.Green("Change Coupling Analysis Results")
	fmt.Printf("Repository: %s\n", report.RepoPath)
	if report.Since != nil {
		fmt.Printf("Period: %s to %s\n", report.Since.Format("2006-01-02"), report.Until.Format("2006-01-02"))
	} else {
		fmt.Printf("Until: %s\n", report.Until.Format("2006-01-02"))
	}
	fmt.Printf("Total commits: %d, Total files: %d, Total pairs: %d\n\n",
		result.TotalCommits, result.TotalFiles, result.TotalPairs)

	if len(result.Couplings) == 0 {
		fmt.Println("No significant file couplings found.")
		return nil
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	// Write header
	fmt.Fprintln(tw, "#\tFile A\tFile B\tCo-Commits\tJaccard\tConfidence\tLift")

	couplings := result.Couplings
	if options.Top > 0 && options.Top < len(couplings) {
		couplings = couplings[:options.Top]
	}

	// Write rows
	for i, c := range couplings {
		fmt.Fprintf(tw, "%d\t%s\t%s\t%d\t%.3f\t%.3f\t%.2f\n",
			i+1,
			c.FileA,
			c.FileB,
			c.CoCommitCount,
			c.JaccardCoefficient,
			c.Confidence,
			c.Lift,
		)
	}

	tw.Flush()

	return nil
}

// Helper functions

func truncateMessage(msg string, maxLen int) string {
	if len(msg) <= maxLen {
		return msg
	}
	return msg[:maxLen-3] + "..."
}

func getLevelColor(level string) func(string, ...interface{}) string {
	switch level {
	case "high":
		return color.RedString
	case "medium":
		return color.YellowString
	default:
		return color.GreenString
	}
}
