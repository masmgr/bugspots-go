package output

import (
	"encoding/csv"
	"fmt"
	"os"
)

// CSVFileWriter writes file analysis reports as CSV.
type CSVFileWriter struct{}

// Write outputs the file analysis report as CSV.
func (w *CSVFileWriter) Write(report *FileAnalysisReport, options OutputOptions) error {
	items := report.Items
	if options.Top > 0 && options.Top < len(items) {
		items = items[:options.Top]
	}

	writer, file, err := createCSVWriter(options.OutputPath)
	if err != nil {
		return err
	}
	if file != nil {
		defer file.Close()
	}

	// Write header
	headers := []string{"Path", "RiskScore", "CommitCount", "ChurnAdded", "ChurnDeleted", "ChurnTotal",
		"LastModified", "Contributors", "BurstScore", "OwnershipRatio", "BugfixCount", "FileSize"}
	if options.Explain {
		headers = append(headers, "CommitComponent", "ChurnComponent", "RecencyComponent",
			"BurstComponent", "OwnershipComponent", "BugfixComponent", "ComplexityComponent")
	}
	if err := writer.Write(headers); err != nil {
		return err
	}

	// Write data
	for _, item := range items {
		row := []string{
			item.Path,
			fmt.Sprintf("%.6f", item.RiskScore),
			fmt.Sprintf("%d", item.Metrics.CommitCount),
			fmt.Sprintf("%d", item.Metrics.AddedLines),
			fmt.Sprintf("%d", item.Metrics.DeletedLines),
			fmt.Sprintf("%d", item.Metrics.ChurnTotal()),
			item.Metrics.LastModifiedAt.Format("2006-01-02T15:04:05"),
			fmt.Sprintf("%d", item.Metrics.ContributorCount()),
			fmt.Sprintf("%.6f", item.Metrics.BurstScore),
			fmt.Sprintf("%.6f", item.Metrics.OwnershipRatio()),
			fmt.Sprintf("%d", item.Metrics.BugfixCount),
			fmt.Sprintf("%d", item.Metrics.FileSize),
		}
		if options.Explain && item.Breakdown != nil {
			row = append(row,
				fmt.Sprintf("%.6f", item.Breakdown.CommitComponent),
				fmt.Sprintf("%.6f", item.Breakdown.ChurnComponent),
				fmt.Sprintf("%.6f", item.Breakdown.RecencyComponent),
				fmt.Sprintf("%.6f", item.Breakdown.BurstComponent),
				fmt.Sprintf("%.6f", item.Breakdown.OwnershipComponent),
				fmt.Sprintf("%.6f", item.Breakdown.BugfixComponent),
				fmt.Sprintf("%.6f", item.Breakdown.ComplexityComponent),
			)
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	writer.Flush()
	return writer.Error()
}

// CSVCommitWriter writes commit analysis reports as CSV.
type CSVCommitWriter struct{}

// Write outputs the commit analysis report as CSV.
func (w *CSVCommitWriter) Write(report *CommitAnalysisReport, options OutputOptions) error {
	items := report.Items
	if options.Top > 0 && options.Top < len(items) {
		items = items[:options.Top]
	}

	writer, file, err := createCSVWriter(options.OutputPath)
	if err != nil {
		return err
	}
	if file != nil {
		defer file.Close()
	}

	// Write header
	headers := []string{"SHA", "When", "Author", "Message", "RiskScore", "RiskLevel",
		"FileCount", "DirectoryCount", "SubsystemCount", "LinesAdded", "LinesDeleted",
		"TotalChurn", "ChangeEntropy"}
	if options.Explain {
		headers = append(headers, "DiffusionComponent", "SizeComponent", "EntropyComponent")
	}
	if err := writer.Write(headers); err != nil {
		return err
	}

	// Write data
	for _, item := range items {
		row := []string{
			item.Metrics.SHA,
			item.Metrics.When.Format("2006-01-02T15:04:05"),
			item.Metrics.Author.Name,
			item.Metrics.Message,
			fmt.Sprintf("%.6f", item.RiskScore),
			string(item.RiskLevel),
			fmt.Sprintf("%d", item.Metrics.FileCount),
			fmt.Sprintf("%d", item.Metrics.DirectoryCount),
			fmt.Sprintf("%d", item.Metrics.SubsystemCount),
			fmt.Sprintf("%d", item.Metrics.LinesAdded),
			fmt.Sprintf("%d", item.Metrics.LinesDeleted),
			fmt.Sprintf("%d", item.Metrics.TotalChurn()),
			fmt.Sprintf("%.6f", item.Metrics.ChangeEntropy),
		}
		if options.Explain && item.Breakdown != nil {
			row = append(row,
				fmt.Sprintf("%.6f", item.Breakdown.DiffusionComponent),
				fmt.Sprintf("%.6f", item.Breakdown.SizeComponent),
				fmt.Sprintf("%.6f", item.Breakdown.EntropyComponent),
			)
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	writer.Flush()
	return writer.Error()
}

// CSVCouplingWriter writes coupling analysis reports as CSV.
type CSVCouplingWriter struct{}

// Write outputs the coupling analysis report as CSV.
func (w *CSVCouplingWriter) Write(report *CouplingAnalysisReport, options OutputOptions) error {
	couplings := report.Result.Couplings
	if options.Top > 0 && options.Top < len(couplings) {
		couplings = couplings[:options.Top]
	}

	writer, file, err := createCSVWriter(options.OutputPath)
	if err != nil {
		return err
	}
	if file != nil {
		defer file.Close()
	}

	// Write header
	headers := []string{"FileA", "FileB", "CoCommitCount", "FileACommitCount",
		"FileBCommitCount", "JaccardCoefficient", "Confidence", "Lift"}
	if err := writer.Write(headers); err != nil {
		return err
	}

	// Write data
	for _, c := range couplings {
		row := []string{
			c.FileA,
			c.FileB,
			fmt.Sprintf("%d", c.CoCommitCount),
			fmt.Sprintf("%d", c.FileACommitCount),
			fmt.Sprintf("%d", c.FileBCommitCount),
			fmt.Sprintf("%.6f", c.JaccardCoefficient),
			fmt.Sprintf("%.6f", c.Confidence),
			fmt.Sprintf("%.6f", c.Lift),
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	writer.Flush()
	return writer.Error()
}

func createCSVWriter(outputPath string) (*csv.Writer, *os.File, error) {
	if outputPath != "" {
		file, err := os.Create(outputPath)
		if err != nil {
			return nil, nil, err
		}
		return csv.NewWriter(file), file, nil
	}
	return csv.NewWriter(os.Stdout), nil, nil
}
