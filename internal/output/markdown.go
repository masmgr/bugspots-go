package output

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// MarkdownFileWriter writes file analysis reports as Markdown.
type MarkdownFileWriter struct{}

// Write outputs the file analysis report as Markdown.
func (w *MarkdownFileWriter) Write(report *FileAnalysisReport, options OutputOptions) error {
	items := report.Items
	if options.Top > 0 && options.Top < len(items) {
		items = items[:options.Top]
	}

	out, file, err := createWriter(options.OutputPath)
	if err != nil {
		return err
	}
	if file != nil {
		defer file.Close()
	}

	// Header
	fmt.Fprintln(out, "# File Hotspot Analysis Results")
	fmt.Fprintln(out)
	fmt.Fprintf(out, "**Repository:** %s\n\n", report.RepoPath)
	if report.Since != nil {
		fmt.Fprintf(out, "**Period:** %s to %s\n\n", report.Since.Format("2006-01-02"), report.Until.Format("2006-01-02"))
	} else {
		fmt.Fprintf(out, "**Until:** %s\n\n", report.Until.Format("2006-01-02"))
	}
	fmt.Fprintf(out, "**Total Files Analyzed:** %d\n\n", len(report.Items))

	// Table header
	fmt.Fprintln(out, "## Top Hotspots")
	fmt.Fprintln(out)
	if options.Explain {
		fmt.Fprintln(out, "| # | Path | Score | Commits | Churn | Contributors | Burst | Bugfixes | Lines | C | Ch | R | B | O | Bf | Cx |")
		fmt.Fprintln(out, "|---|------|-------|---------|-------|--------------|-------|----------|-------|---|----|----|---|---|----|-----|")
	} else {
		fmt.Fprintln(out, "| # | Path | Score | Commits | Churn | Contributors | Burst | Bugfixes | Lines |")
		fmt.Fprintln(out, "|---|------|-------|---------|-------|--------------|-------|----------|-------|")
	}

	// Table rows
	for i, item := range items {
		if options.Explain && item.Breakdown != nil {
			fmt.Fprintf(out, "| %d | `%s` | %.4f | %d | %d | %d | %.2f | %d | %d | %.3f | %.3f | %.3f | %.3f | %.3f | %.3f | %.3f |\n",
				i+1, item.Path, item.RiskScore, item.Metrics.CommitCount, item.Metrics.ChurnTotal(),
				item.Metrics.ContributorCount(), item.Metrics.BurstScore, item.Metrics.BugfixCount,
				item.Metrics.FileSize,
				item.Breakdown.CommitComponent, item.Breakdown.ChurnComponent,
				item.Breakdown.RecencyComponent, item.Breakdown.BurstComponent,
				item.Breakdown.OwnershipComponent, item.Breakdown.BugfixComponent,
				item.Breakdown.ComplexityComponent)
		} else {
			fmt.Fprintf(out, "| %d | `%s` | %.4f | %d | %d | %d | %.2f | %d | %d |\n",
				i+1, item.Path, item.RiskScore, item.Metrics.CommitCount, item.Metrics.ChurnTotal(),
				item.Metrics.ContributorCount(), item.Metrics.BurstScore, item.Metrics.BugfixCount,
				item.Metrics.FileSize)
		}
	}

	if options.Explain {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "**Score Breakdown:** C=Commit, Ch=Churn, R=Recency, B=Burst, O=Ownership, Bf=Bugfix, Cx=Complexity")
	}

	return nil
}

// MarkdownCommitWriter writes commit analysis reports as Markdown.
type MarkdownCommitWriter struct{}

// Write outputs the commit analysis report as Markdown.
func (w *MarkdownCommitWriter) Write(report *CommitAnalysisReport, options OutputOptions) error {
	items := report.Items
	if options.Top > 0 && options.Top < len(items) {
		items = items[:options.Top]
	}

	out, file, err := createWriter(options.OutputPath)
	if err != nil {
		return err
	}
	if file != nil {
		defer file.Close()
	}

	// Header
	fmt.Fprintln(out, "# Commit Risk Analysis Results")
	fmt.Fprintln(out)
	fmt.Fprintf(out, "**Repository:** %s\n\n", report.RepoPath)
	if report.Since != nil {
		fmt.Fprintf(out, "**Period:** %s to %s\n\n", report.Since.Format("2006-01-02"), report.Until.Format("2006-01-02"))
	} else {
		fmt.Fprintf(out, "**Until:** %s\n\n", report.Until.Format("2006-01-02"))
	}
	fmt.Fprintf(out, "**Total Commits Analyzed:** %d\n\n", len(report.Items))

	// Table header
	fmt.Fprintln(out, "## High-Risk Commits")
	fmt.Fprintln(out)
	if options.Explain {
		fmt.Fprintln(out, "| # | SHA | Score | Level | Files | Churn | Entropy | Message | D | S | E |")
		fmt.Fprintln(out, "|---|-----|-------|-------|-------|-------|---------|---------|---|---|---|")
	} else {
		fmt.Fprintln(out, "| # | SHA | Score | Level | Files | Churn | Entropy | Message |")
		fmt.Fprintln(out, "|---|-----|-------|-------|-------|-------|---------|---------|")
	}

	// Table rows
	for i, item := range items {
		levelEmoji := getRiskLevelEmoji(string(item.RiskLevel))
		escapedMsg := escapeMarkdown(item.Metrics.Message)
		if len(escapedMsg) > 40 {
			escapedMsg = escapedMsg[:37] + "..."
		}

		if options.Explain && item.Breakdown != nil {
			fmt.Fprintf(out, "| %d | `%s` | %.4f | %s %s | %d | %d | %.2f | %s | %.3f | %.3f | %.3f |\n",
				i+1, item.Metrics.SHA[:8], item.RiskScore, levelEmoji, item.RiskLevel,
				item.Metrics.FileCount, item.Metrics.TotalChurn(), item.Metrics.ChangeEntropy,
				escapedMsg, item.Breakdown.DiffusionComponent, item.Breakdown.SizeComponent,
				item.Breakdown.EntropyComponent)
		} else {
			fmt.Fprintf(out, "| %d | `%s` | %.4f | %s %s | %d | %d | %.2f | %s |\n",
				i+1, item.Metrics.SHA[:8], item.RiskScore, levelEmoji, item.RiskLevel,
				item.Metrics.FileCount, item.Metrics.TotalChurn(), item.Metrics.ChangeEntropy,
				escapedMsg)
		}
	}

	if options.Explain {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "**Score Breakdown:** D=Diffusion, S=Size, E=Entropy")
	}

	return nil
}

// MarkdownCouplingWriter writes coupling analysis reports as Markdown.
type MarkdownCouplingWriter struct{}

// Write outputs the coupling analysis report as Markdown.
func (w *MarkdownCouplingWriter) Write(report *CouplingAnalysisReport, options OutputOptions) error {
	result := report.Result

	out, file, err := createWriter(options.OutputPath)
	if err != nil {
		return err
	}
	if file != nil {
		defer file.Close()
	}

	// Header
	fmt.Fprintln(out, "# Change Coupling Analysis Results")
	fmt.Fprintln(out)
	fmt.Fprintf(out, "**Repository:** %s\n\n", report.RepoPath)
	if report.Since != nil {
		fmt.Fprintf(out, "**Period:** %s to %s\n\n", report.Since.Format("2006-01-02"), report.Until.Format("2006-01-02"))
	} else {
		fmt.Fprintf(out, "**Until:** %s\n\n", report.Until.Format("2006-01-02"))
	}
	fmt.Fprintf(out, "**Statistics:** %d commits, %d files, %d pairs analyzed\n\n",
		result.TotalCommits, result.TotalFiles, result.TotalPairs)

	if len(result.Couplings) == 0 {
		fmt.Fprintln(out, "No significant file couplings found.")
		return nil
	}

	couplings := result.Couplings
	if options.Top > 0 && options.Top < len(couplings) {
		couplings = couplings[:options.Top]
	}

	// Table header
	fmt.Fprintln(out, "## Coupled File Pairs")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "| # | File A | File B | Co-Commits | Jaccard | Confidence | Lift |")
	fmt.Fprintln(out, "|---|--------|--------|------------|---------|------------|------|")

	// Table rows
	for i, c := range couplings {
		fmt.Fprintf(out, "| %d | `%s` | `%s` | %d | %.3f | %.3f | %.2f |\n",
			i+1, c.FileA, c.FileB, c.CoCommitCount,
			c.JaccardCoefficient, c.Confidence, c.Lift)
	}

	return nil
}

func createWriter(outputPath string) (io.Writer, *os.File, error) {
	if outputPath != "" {
		file, err := os.Create(outputPath)
		if err != nil {
			return nil, nil, err
		}
		return file, file, nil
	}
	return os.Stdout, nil, nil
}

func getRiskLevelEmoji(level string) string {
	switch level {
	case "high":
		return "🔴"
	case "medium":
		return "🟡"
	default:
		return "🟢"
	}
}

func escapeMarkdown(s string) string {
	replacer := strings.NewReplacer(
		"|", "\\|",
		"*", "\\*",
		"_", "\\_",
		"`", "\\`",
	)
	return replacer.Replace(s)
}
