package output

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/masmgr/bugspots-go/internal/aggregation"
	"github.com/masmgr/bugspots-go/internal/scoring"
)

func TestCIFileWriter_Write(t *testing.T) {
	now := time.Now()

	report := &FileAnalysisReport{
		RepoPath:    "/test/repo",
		Until:       now,
		GeneratedAt: now,
		Items: []scoring.FileRiskItem{
			{
				Path:      "hot.go",
				RiskScore: 0.85,
				Metrics:   &aggregation.FileMetrics{CommitCount: 10},
			},
			{
				Path:      "warm.go",
				RiskScore: 0.50,
				Metrics:   &aggregation.FileMetrics{CommitCount: 5},
			},
			{
				Path:      "cold.go",
				RiskScore: 0.20,
				Metrics:   &aggregation.FileMetrics{CommitCount: 2},
			},
		},
	}

	// Write to a temp file
	tmpFile := t.TempDir() + "/ci_output.ndjson"
	options := OutputOptions{
		Format:     FormatCI,
		OutputPath: tmpFile,
	}

	writer := &CIFileWriter{}
	if err := writer.Write(report, options); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Read the output
	data, err := readTestFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to read output: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 4 { // 1 summary + 3 files
		t.Fatalf("expected 4 lines, got %d: %s", len(lines), string(data))
	}

	// Verify summary line
	var summary CISummary
	if err := json.Unmarshal([]byte(lines[0]), &summary); err != nil {
		t.Fatalf("Failed to parse summary: %v", err)
	}
	if summary.Type != "summary" {
		t.Errorf("summary.Type = %q, want %q", summary.Type, "summary")
	}
	if summary.TotalFiles != 3 {
		t.Errorf("summary.TotalFiles = %d, want 3", summary.TotalFiles)
	}
	if summary.HighRiskCount != 1 {
		t.Errorf("summary.HighRiskCount = %d, want 1", summary.HighRiskCount)
	}
	if summary.MediumRiskCount != 1 {
		t.Errorf("summary.MediumRiskCount = %d, want 1", summary.MediumRiskCount)
	}
	if summary.MaxRiskScore != 0.85 {
		t.Errorf("summary.MaxRiskScore = %f, want 0.85", summary.MaxRiskScore)
	}

	// Verify first file entry
	var entry CIFileEntry
	if err := json.Unmarshal([]byte(lines[1]), &entry); err != nil {
		t.Fatalf("Failed to parse entry: %v", err)
	}
	if entry.Type != "file" {
		t.Errorf("entry.Type = %q, want %q", entry.Type, "file")
	}
	if entry.Path != "hot.go" {
		t.Errorf("entry.Path = %q, want %q", entry.Path, "hot.go")
	}
	if entry.RiskLevel != "high" {
		t.Errorf("entry.RiskLevel = %q, want %q", entry.RiskLevel, "high")
	}

}

func TestCIFileWriter_RiskLevelClassification(t *testing.T) {
	now := time.Now()

	report := &FileAnalysisReport{
		RepoPath:    "/test/repo",
		Until:       now,
		GeneratedAt: now,
		Items: []scoring.FileRiskItem{
			{Path: "high.go", RiskScore: 0.75, Metrics: &aggregation.FileMetrics{}},
			{Path: "medium.go", RiskScore: 0.50, Metrics: &aggregation.FileMetrics{}},
			{Path: "low.go", RiskScore: 0.30, Metrics: &aggregation.FileMetrics{}},
		},
	}

	tmpFile := t.TempDir() + "/ci_classify.ndjson"
	options := OutputOptions{Format: FormatCI, OutputPath: tmpFile}

	writer := &CIFileWriter{}
	if err := writer.Write(report, options); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	data, err := readTestFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to read output: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")

	expected := []struct {
		path  string
		level string
	}{
		{"high.go", "high"},
		{"medium.go", "medium"},
		{"low.go", "low"},
	}

	for i, exp := range expected {
		var entry CIFileEntry
		if err := json.Unmarshal([]byte(lines[i+1]), &entry); err != nil {
			t.Fatalf("Failed to parse entry %d: %v", i, err)
		}
		if entry.Path != exp.path {
			t.Errorf("entry[%d].Path = %q, want %q", i, entry.Path, exp.path)
		}
		if entry.RiskLevel != exp.level {
			t.Errorf("entry[%d].RiskLevel = %q, want %q", i, entry.RiskLevel, exp.level)
		}
	}
}

func TestCIFileWriter_TopOption(t *testing.T) {
	now := time.Now()

	report := &FileAnalysisReport{
		RepoPath:    "/test/repo",
		Until:       now,
		GeneratedAt: now,
		Items: []scoring.FileRiskItem{
			{Path: "a.go", RiskScore: 0.9, Metrics: &aggregation.FileMetrics{}},
			{Path: "b.go", RiskScore: 0.5, Metrics: &aggregation.FileMetrics{}},
			{Path: "c.go", RiskScore: 0.1, Metrics: &aggregation.FileMetrics{}},
		},
	}

	tmpFile := t.TempDir() + "/ci_top.ndjson"
	options := OutputOptions{Format: FormatCI, Top: 1, OutputPath: tmpFile}

	writer := &CIFileWriter{}
	if err := writer.Write(report, options); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	data, err := readTestFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to read output: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 { // 1 summary + 1 file
		t.Fatalf("expected 2 lines with Top=1, got %d", len(lines))
	}
}

func readTestFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}
