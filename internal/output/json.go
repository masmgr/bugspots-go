package output

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// JSONFileWriter writes file analysis reports as JSON.
type JSONFileWriter struct{}

// JSONFileReport is the JSON output structure for file analysis.
type JSONFileReport struct {
	RepoPath    string         `json:"repo"`
	Since       *string        `json:"since,omitempty"`
	Until       string         `json:"until"`
	GeneratedAt string         `json:"generatedAt"`
	TotalFiles  int            `json:"totalFiles"`
	Items       []JSONFileItem `json:"items"`
}

// JSONFileItem is the JSON output structure for a single file.
type JSONFileItem struct {
	Path      string             `json:"path"`
	RiskScore float64            `json:"riskScore"`
	Metrics   JSONFileMetrics    `json:"metrics"`
	Breakdown *JSONFileBreakdown `json:"breakdown,omitempty"`
}

// JSONFileMetrics holds the metrics for a file in JSON format.
type JSONFileMetrics struct {
	CommitCount    int     `json:"commitCount"`
	ChurnAdded     int     `json:"churnAdded"`
	ChurnDeleted   int     `json:"churnDeleted"`
	ChurnTotal     int     `json:"churnTotal"`
	LastModified   string  `json:"lastModified"`
	Contributors   int     `json:"contributors"`
	BurstScore     float64 `json:"burstScore"`
	OwnershipRatio float64 `json:"ownershipRatio"`
	BugfixCount    int     `json:"bugfixCount"`
	FileSize       int     `json:"fileSize"`
}

// JSONFileBreakdown holds the score breakdown for a file in JSON format.
type JSONFileBreakdown struct {
	Commit     float64 `json:"commit"`
	Churn      float64 `json:"churn"`
	Recency    float64 `json:"recency"`
	Burst      float64 `json:"burst"`
	Ownership  float64 `json:"ownership"`
	Bugfix     float64 `json:"bugfix"`
	Complexity float64 `json:"complexity"`
}

// Write outputs the file analysis report as JSON.
func (w *JSONFileWriter) Write(report *FileAnalysisReport, options OutputOptions) error {
	items := report.Items
	if options.Top > 0 && options.Top < len(items) {
		items = items[:options.Top]
	}

	jsonItems := make([]JSONFileItem, len(items))
	for i, item := range items {
		jsonItem := JSONFileItem{
			Path:      item.Path,
			RiskScore: item.RiskScore,
			Metrics: JSONFileMetrics{
				CommitCount:    item.Metrics.CommitCount,
				ChurnAdded:     item.Metrics.AddedLines,
				ChurnDeleted:   item.Metrics.DeletedLines,
				ChurnTotal:     item.Metrics.ChurnTotal(),
				LastModified:   item.Metrics.LastModifiedAt.Format(time.RFC3339),
				Contributors:   item.Metrics.ContributorCount(),
				BurstScore:     item.Metrics.BurstScore,
				OwnershipRatio: item.Metrics.OwnershipRatio(),
				BugfixCount:    item.Metrics.BugfixCount,
				FileSize:       item.Metrics.FileSize,
			},
		}
		if options.Explain && item.Breakdown != nil {
			jsonItem.Breakdown = &JSONFileBreakdown{
				Commit:     item.Breakdown.CommitComponent,
				Churn:      item.Breakdown.ChurnComponent,
				Recency:    item.Breakdown.RecencyComponent,
				Burst:      item.Breakdown.BurstComponent,
				Ownership:  item.Breakdown.OwnershipComponent,
				Bugfix:     item.Breakdown.BugfixComponent,
				Complexity: item.Breakdown.ComplexityComponent,
			}
		}
		jsonItems[i] = jsonItem
	}

	var since *string
	if report.Since != nil {
		s := report.Since.Format("2006-01-02")
		since = &s
	}

	jsonReport := JSONFileReport{
		RepoPath:    report.RepoPath,
		Since:       since,
		Until:       report.Until.Format("2006-01-02"),
		GeneratedAt: report.GeneratedAt.Format(time.RFC3339),
		TotalFiles:  len(report.Items),
		Items:       jsonItems,
	}

	return writeJSON(jsonReport, options.OutputPath)
}

// JSONCommitWriter writes commit analysis reports as JSON.
type JSONCommitWriter struct{}

// JSONCommitReport is the JSON output structure for commit analysis.
type JSONCommitReport struct {
	RepoPath     string           `json:"repo"`
	Since        *string          `json:"since,omitempty"`
	Until        string           `json:"until"`
	GeneratedAt  string           `json:"generatedAt"`
	TotalCommits int              `json:"totalCommits"`
	Items        []JSONCommitItem `json:"items"`
}

// JSONCommitItem is the JSON output structure for a single commit.
type JSONCommitItem struct {
	SHA       string               `json:"sha"`
	When      string               `json:"when"`
	Author    string               `json:"author"`
	Message   string               `json:"message"`
	RiskScore float64              `json:"riskScore"`
	RiskLevel string               `json:"riskLevel"`
	Metrics   JSONCommitMetrics    `json:"metrics"`
	Breakdown *JSONCommitBreakdown `json:"breakdown,omitempty"`
}

// JSONCommitMetrics holds the metrics for a commit in JSON format.
type JSONCommitMetrics struct {
	FileCount      int     `json:"fileCount"`
	DirectoryCount int     `json:"directoryCount"`
	SubsystemCount int     `json:"subsystemCount"`
	LinesAdded     int     `json:"linesAdded"`
	LinesDeleted   int     `json:"linesDeleted"`
	TotalChurn     int     `json:"totalChurn"`
	ChangeEntropy  float64 `json:"changeEntropy"`
}

// JSONCommitBreakdown holds the score breakdown for a commit in JSON format.
type JSONCommitBreakdown struct {
	Diffusion float64 `json:"diffusion"`
	Size      float64 `json:"size"`
	Entropy   float64 `json:"entropy"`
}

// Write outputs the commit analysis report as JSON.
func (w *JSONCommitWriter) Write(report *CommitAnalysisReport, options OutputOptions) error {
	items := report.Items
	if options.Top > 0 && options.Top < len(items) {
		items = items[:options.Top]
	}

	jsonItems := make([]JSONCommitItem, len(items))
	for i, item := range items {
		jsonItem := JSONCommitItem{
			SHA:       item.Metrics.SHA,
			When:      item.Metrics.When.Format(time.RFC3339),
			Author:    item.Metrics.Author.Name,
			Message:   item.Metrics.Message,
			RiskScore: item.RiskScore,
			RiskLevel: string(item.RiskLevel),
			Metrics: JSONCommitMetrics{
				FileCount:      item.Metrics.FileCount,
				DirectoryCount: item.Metrics.DirectoryCount,
				SubsystemCount: item.Metrics.SubsystemCount,
				LinesAdded:     item.Metrics.LinesAdded,
				LinesDeleted:   item.Metrics.LinesDeleted,
				TotalChurn:     item.Metrics.TotalChurn(),
				ChangeEntropy:  item.Metrics.ChangeEntropy,
			},
		}
		if options.Explain && item.Breakdown != nil {
			jsonItem.Breakdown = &JSONCommitBreakdown{
				Diffusion: item.Breakdown.DiffusionComponent,
				Size:      item.Breakdown.SizeComponent,
				Entropy:   item.Breakdown.EntropyComponent,
			}
		}
		jsonItems[i] = jsonItem
	}

	var since *string
	if report.Since != nil {
		s := report.Since.Format("2006-01-02")
		since = &s
	}

	jsonReport := JSONCommitReport{
		RepoPath:     report.RepoPath,
		Since:        since,
		Until:        report.Until.Format("2006-01-02"),
		GeneratedAt:  report.GeneratedAt.Format(time.RFC3339),
		TotalCommits: len(report.Items),
		Items:        jsonItems,
	}

	return writeJSON(jsonReport, options.OutputPath)
}

// JSONCouplingWriter writes coupling analysis reports as JSON.
type JSONCouplingWriter struct{}

// JSONCouplingReport is the JSON output structure for coupling analysis.
type JSONCouplingReport struct {
	RepoPath     string             `json:"repo"`
	Since        *string            `json:"since,omitempty"`
	Until        string             `json:"until"`
	GeneratedAt  string             `json:"generatedAt"`
	TotalCommits int                `json:"totalCommits"`
	TotalFiles   int                `json:"totalFiles"`
	TotalPairs   int                `json:"totalPairs"`
	Items        []JSONCouplingItem `json:"items"`
}

// JSONCouplingItem is the JSON output structure for a single coupling pair.
type JSONCouplingItem struct {
	FileA              string  `json:"fileA"`
	FileB              string  `json:"fileB"`
	CoCommitCount      int     `json:"coCommitCount"`
	FileACommitCount   int     `json:"fileACommitCount"`
	FileBCommitCount   int     `json:"fileBCommitCount"`
	JaccardCoefficient float64 `json:"jaccardCoefficient"`
	Confidence         float64 `json:"confidence"`
	Lift               float64 `json:"lift"`
}

// Write outputs the coupling analysis report as JSON.
func (w *JSONCouplingWriter) Write(report *CouplingAnalysisReport, options OutputOptions) error {
	couplings := report.Result.Couplings
	if options.Top > 0 && options.Top < len(couplings) {
		couplings = couplings[:options.Top]
	}

	jsonItems := make([]JSONCouplingItem, len(couplings))
	for i, c := range couplings {
		jsonItems[i] = JSONCouplingItem{
			FileA:              c.FileA,
			FileB:              c.FileB,
			CoCommitCount:      c.CoCommitCount,
			FileACommitCount:   c.FileACommitCount,
			FileBCommitCount:   c.FileBCommitCount,
			JaccardCoefficient: c.JaccardCoefficient,
			Confidence:         c.Confidence,
			Lift:               c.Lift,
		}
	}

	var since *string
	if report.Since != nil {
		s := report.Since.Format("2006-01-02")
		since = &s
	}

	jsonReport := JSONCouplingReport{
		RepoPath:     report.RepoPath,
		Since:        since,
		Until:        report.Until.Format("2006-01-02"),
		GeneratedAt:  report.GeneratedAt.Format(time.RFC3339),
		TotalCommits: report.Result.TotalCommits,
		TotalFiles:   report.Result.TotalFiles,
		TotalPairs:   report.Result.TotalPairs,
		Items:        jsonItems,
	}

	return writeJSON(jsonReport, options.OutputPath)
}

func writeJSON(data interface{}, outputPath string) error {
	encoder := json.NewEncoder(os.Stdout)
	if outputPath != "" {
		file, err := os.Create(outputPath)
		if err != nil {
			return err
		}
		defer file.Close()
		encoder = json.NewEncoder(file)
	}

	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}
	return nil
}
