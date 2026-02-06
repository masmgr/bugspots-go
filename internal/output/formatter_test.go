package output

import "testing"

func TestNewFileReportWriter(t *testing.T) {
	tests := []struct {
		name         string
		format       OutputFormat
		expectedType string
	}{
		{name: "Console", format: FormatConsole, expectedType: "*output.ConsoleFileWriter"},
		{name: "JSON", format: FormatJSON, expectedType: "*output.JSONFileWriter"},
		{name: "CSV", format: FormatCSV, expectedType: "*output.CSVFileWriter"},
		{name: "Markdown", format: FormatMarkdown, expectedType: "*output.MarkdownFileWriter"},
		{name: "Unknown defaults to Console", format: "unknown", expectedType: "*output.ConsoleFileWriter"},
		{name: "Empty defaults to Console", format: "", expectedType: "*output.ConsoleFileWriter"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer := NewFileReportWriter(tt.format)
			if writer == nil {
				t.Fatal("NewFileReportWriter returned nil")
			}

			switch tt.format {
			case FormatJSON:
				if _, ok := writer.(*JSONFileWriter); !ok {
					t.Errorf("Expected *JSONFileWriter for format %q", tt.format)
				}
			case FormatCSV:
				if _, ok := writer.(*CSVFileWriter); !ok {
					t.Errorf("Expected *CSVFileWriter for format %q", tt.format)
				}
			case FormatMarkdown:
				if _, ok := writer.(*MarkdownFileWriter); !ok {
					t.Errorf("Expected *MarkdownFileWriter for format %q", tt.format)
				}
			default:
				if _, ok := writer.(*ConsoleFileWriter); !ok {
					t.Errorf("Expected *ConsoleFileWriter for format %q", tt.format)
				}
			}
		})
	}
}

func TestNewCommitReportWriter(t *testing.T) {
	tests := []struct {
		name   string
		format OutputFormat
	}{
		{name: "Console", format: FormatConsole},
		{name: "JSON", format: FormatJSON},
		{name: "CSV", format: FormatCSV},
		{name: "Markdown", format: FormatMarkdown},
		{name: "Unknown defaults to Console", format: "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer := NewCommitReportWriter(tt.format)
			if writer == nil {
				t.Fatal("NewCommitReportWriter returned nil")
			}

			switch tt.format {
			case FormatJSON:
				if _, ok := writer.(*JSONCommitWriter); !ok {
					t.Errorf("Expected *JSONCommitWriter for format %q", tt.format)
				}
			case FormatCSV:
				if _, ok := writer.(*CSVCommitWriter); !ok {
					t.Errorf("Expected *CSVCommitWriter for format %q", tt.format)
				}
			case FormatMarkdown:
				if _, ok := writer.(*MarkdownCommitWriter); !ok {
					t.Errorf("Expected *MarkdownCommitWriter for format %q", tt.format)
				}
			default:
				if _, ok := writer.(*ConsoleCommitWriter); !ok {
					t.Errorf("Expected *ConsoleCommitWriter for format %q", tt.format)
				}
			}
		})
	}
}

func TestNewCouplingReportWriter(t *testing.T) {
	tests := []struct {
		name   string
		format OutputFormat
	}{
		{name: "Console", format: FormatConsole},
		{name: "JSON", format: FormatJSON},
		{name: "CSV", format: FormatCSV},
		{name: "Markdown", format: FormatMarkdown},
		{name: "Unknown defaults to Console", format: "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer := NewCouplingReportWriter(tt.format)
			if writer == nil {
				t.Fatal("NewCouplingReportWriter returned nil")
			}

			switch tt.format {
			case FormatJSON:
				if _, ok := writer.(*JSONCouplingWriter); !ok {
					t.Errorf("Expected *JSONCouplingWriter for format %q", tt.format)
				}
			case FormatCSV:
				if _, ok := writer.(*CSVCouplingWriter); !ok {
					t.Errorf("Expected *CSVCouplingWriter for format %q", tt.format)
				}
			case FormatMarkdown:
				if _, ok := writer.(*MarkdownCouplingWriter); !ok {
					t.Errorf("Expected *MarkdownCouplingWriter for format %q", tt.format)
				}
			default:
				if _, ok := writer.(*ConsoleCouplingWriter); !ok {
					t.Errorf("Expected *ConsoleCouplingWriter for format %q", tt.format)
				}
			}
		})
	}
}
