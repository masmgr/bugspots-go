package output

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/masmgr/bugspots-go/config"
)

// CIFileWriter writes file analysis reports as NDJSON (one JSON object per line) for CI pipelines.
type CIFileWriter struct{}

// CISummary is the first line of CI output, containing aggregate statistics.
type CISummary struct {
	Type            string  `json:"type"`
	TotalFiles      int     `json:"totalFiles"`
	HighRiskCount   int     `json:"highRiskCount"`
	MediumRiskCount int     `json:"mediumRiskCount"`
	MaxRiskScore    float64 `json:"maxRiskScore"`
}

// CIFileEntry represents a single file entry in CI output.
type CIFileEntry struct {
	Type      string  `json:"type"`
	Path      string  `json:"path"`
	RiskScore float64 `json:"riskScore"`
	RiskLevel string  `json:"riskLevel"`
}

// Write outputs the file analysis report as NDJSON.
func (w *CIFileWriter) Write(report *FileAnalysisReport, options OutputOptions) error {
	items := limitTop(report.Items, options.Top)

	out, file, err := openOutputWriter(options.OutputPath)
	if err != nil {
		return err
	}
	if file != nil {
		defer file.Close()
	}

	thresholds := config.DefaultRiskThresholds()

	// Classify and count risk levels
	var highCount, mediumCount int
	var maxScore float64
	for _, item := range items {
		level := thresholds.Classify(item.RiskScore)
		switch level {
		case config.RiskLevelHigh:
			highCount++
		case config.RiskLevelMedium:
			mediumCount++
		}
		if item.RiskScore > maxScore {
			maxScore = item.RiskScore
		}
	}

	// Write summary line
	summary := CISummary{
		Type:            "summary",
		TotalFiles:      len(items),
		HighRiskCount:   highCount,
		MediumRiskCount: mediumCount,
		MaxRiskScore:    maxScore,
	}
	if err := writeNDJSONLine(out, summary); err != nil {
		return err
	}

	// Write file entries
	for _, item := range items {
		level := thresholds.Classify(item.RiskScore)
		entry := CIFileEntry{
			Type:      "file",
			Path:      item.Path,
			RiskScore: item.RiskScore,
			RiskLevel: string(level),
		}
		if err := writeNDJSONLine(out, entry); err != nil {
			return err
		}
	}

	return nil
}

func writeNDJSONLine(w io.Writer, v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("failed to marshal NDJSON: %w", err)
	}
	_, err = fmt.Fprintf(w, "%s\n", data)
	return err
}
