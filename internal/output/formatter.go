package output

import (
	"time"

	"github.com/masmgr/bugspots-go/internal/coupling"
	"github.com/masmgr/bugspots-go/internal/scoring"
)

// Compile-time interface conformance checks.
// These ensure that all writer types correctly implement their respective interfaces.
var (
	// FileReportWriter implementations
	_ FileReportWriter = (*ConsoleFileWriter)(nil)
	_ FileReportWriter = (*JSONFileWriter)(nil)
	_ FileReportWriter = (*CSVFileWriter)(nil)
	_ FileReportWriter = (*MarkdownFileWriter)(nil)
	_ FileReportWriter = (*CIFileWriter)(nil)

	// CommitReportWriter implementations
	_ CommitReportWriter = (*ConsoleCommitWriter)(nil)
	_ CommitReportWriter = (*JSONCommitWriter)(nil)
	_ CommitReportWriter = (*CSVCommitWriter)(nil)
	_ CommitReportWriter = (*MarkdownCommitWriter)(nil)

	// CouplingReportWriter implementations
	_ CouplingReportWriter = (*ConsoleCouplingWriter)(nil)
	_ CouplingReportWriter = (*JSONCouplingWriter)(nil)
	_ CouplingReportWriter = (*CSVCouplingWriter)(nil)
	_ CouplingReportWriter = (*MarkdownCouplingWriter)(nil)
)

// OutputFormat represents the output format type.
type OutputFormat string

const (
	FormatConsole  OutputFormat = "console"
	FormatJSON     OutputFormat = "json"
	FormatCSV      OutputFormat = "csv"
	FormatMarkdown OutputFormat = "markdown"
	FormatCI       OutputFormat = "ci"
)

// OutputOptions controls output behavior.
type OutputOptions struct {
	Format     OutputFormat
	Top        int
	OutputPath string
	Explain    bool
}

// FileAnalysisReport holds the results of file hotspot analysis.
type FileAnalysisReport struct {
	RepoPath    string
	Since       *time.Time
	Until       time.Time
	GeneratedAt time.Time
	Items       []scoring.FileRiskItem
}

// CommitAnalysisReport holds the results of commit risk analysis.
type CommitAnalysisReport struct {
	RepoPath    string
	Since       *time.Time
	Until       time.Time
	GeneratedAt time.Time
	Items       []scoring.CommitRiskItem
}

// CouplingAnalysisReport holds the results of coupling analysis.
type CouplingAnalysisReport struct {
	RepoPath    string
	Since       *time.Time
	Until       time.Time
	GeneratedAt time.Time
	Result      coupling.CouplingAnalysisResult
}

// FileReportWriter writes file analysis reports.
type FileReportWriter interface {
	Write(report *FileAnalysisReport, options OutputOptions) error
}

// CommitReportWriter writes commit analysis reports.
type CommitReportWriter interface {
	Write(report *CommitAnalysisReport, options OutputOptions) error
}

// CouplingReportWriter writes coupling analysis reports.
type CouplingReportWriter interface {
	Write(report *CouplingAnalysisReport, options OutputOptions) error
}

// NewFileReportWriter creates a report writer for the specified format.
func NewFileReportWriter(format OutputFormat) FileReportWriter {
	switch format {
	case FormatJSON:
		return &JSONFileWriter{}
	case FormatCSV:
		return &CSVFileWriter{}
	case FormatMarkdown:
		return &MarkdownFileWriter{}
	case FormatCI:
		return &CIFileWriter{}
	default:
		return &ConsoleFileWriter{}
	}
}

// NewCommitReportWriter creates a commit report writer for the specified format.
func NewCommitReportWriter(format OutputFormat) CommitReportWriter {
	switch format {
	case FormatJSON:
		return &JSONCommitWriter{}
	case FormatCSV:
		return &CSVCommitWriter{}
	case FormatMarkdown:
		return &MarkdownCommitWriter{}
	default:
		return &ConsoleCommitWriter{}
	}
}

// NewCouplingReportWriter creates a coupling report writer for the specified format.
func NewCouplingReportWriter(format OutputFormat) CouplingReportWriter {
	switch format {
	case FormatJSON:
		return &JSONCouplingWriter{}
	case FormatCSV:
		return &CSVCouplingWriter{}
	case FormatMarkdown:
		return &MarkdownCouplingWriter{}
	default:
		return &ConsoleCouplingWriter{}
	}
}
