package output

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
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

	table := tablewriter.NewWriter(os.Stdout)

	headers := []string{"#", "Path", "Score", "Commits", "Churn", "Contributors", "Burst"}
	if options.Explain {
		headers = append(headers, "C", "Ch", "R", "B", "O")
	}
	table.SetHeader(headers)

	table.SetBorder(false)
	table.SetColumnSeparator(" ")
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	for i, item := range items {
		row := []string{
			fmt.Sprintf("%d", i+1),
			truncatePath(item.Path, 50),
			fmt.Sprintf("%.4f", item.RiskScore),
			fmt.Sprintf("%d", item.Metrics.CommitCount),
			fmt.Sprintf("%d", item.Metrics.ChurnTotal()),
			fmt.Sprintf("%d", item.Metrics.ContributorCount()),
			fmt.Sprintf("%.2f", item.Metrics.BurstScore),
		}
		if options.Explain && item.Breakdown != nil {
			row = append(row,
				fmt.Sprintf("%.3f", item.Breakdown.CommitComponent),
				fmt.Sprintf("%.3f", item.Breakdown.ChurnComponent),
				fmt.Sprintf("%.3f", item.Breakdown.RecencyComponent),
				fmt.Sprintf("%.3f", item.Breakdown.BurstComponent),
				fmt.Sprintf("%.3f", item.Breakdown.OwnershipComponent),
			)
		}
		table.Append(row)
	}

	table.Render()

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

	table := tablewriter.NewWriter(os.Stdout)

	headers := []string{"#", "SHA", "Score", "Level", "Files", "Churn", "Entropy", "Message"}
	if options.Explain {
		headers = append(headers, "D", "S", "E")
	}
	table.SetHeader(headers)

	table.SetBorder(false)
	table.SetColumnSeparator(" ")
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	for i, item := range items {
		levelColor := getLevelColor(string(item.RiskLevel))
		row := []string{
			fmt.Sprintf("%d", i+1),
			item.Metrics.SHA[:8],
			fmt.Sprintf("%.4f", item.RiskScore),
			levelColor(string(item.RiskLevel)),
			fmt.Sprintf("%d", item.Metrics.FileCount),
			fmt.Sprintf("%d", item.Metrics.TotalChurn()),
			fmt.Sprintf("%.2f", item.Metrics.ChangeEntropy),
			truncateMessage(item.Metrics.Message, 40),
		}
		if options.Explain && item.Breakdown != nil {
			row = append(row,
				fmt.Sprintf("%.3f", item.Breakdown.DiffusionComponent),
				fmt.Sprintf("%.3f", item.Breakdown.SizeComponent),
				fmt.Sprintf("%.3f", item.Breakdown.EntropyComponent),
			)
		}
		table.Append(row)
	}

	table.Render()

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

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"#", "File A", "File B", "Co-Commits", "Jaccard", "Confidence", "Lift"})

	table.SetBorder(false)
	table.SetColumnSeparator(" ")
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	couplings := result.Couplings
	if options.Top > 0 && options.Top < len(couplings) {
		couplings = couplings[:options.Top]
	}

	for i, c := range couplings {
		row := []string{
			fmt.Sprintf("%d", i+1),
			truncatePath(c.FileA, 35),
			truncatePath(c.FileB, 35),
			fmt.Sprintf("%d", c.CoCommitCount),
			fmt.Sprintf("%.3f", c.JaccardCoefficient),
			fmt.Sprintf("%.3f", c.Confidence),
			fmt.Sprintf("%.2f", c.Lift),
		}
		table.Append(row)
	}

	table.Render()

	return nil
}

// Helper functions

func truncatePath(path string, maxLen int) string {
	if len(path) <= maxLen {
		return path
	}
	// Keep the last part of the path
	parts := strings.Split(path, "/")
	result := ""
	for i := len(parts) - 1; i >= 0; i-- {
		if result == "" {
			result = parts[i]
		} else {
			newResult := parts[i] + "/" + result
			if len(newResult) > maxLen-3 {
				return "..." + result
			}
			result = newResult
		}
	}
	return result
}

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
