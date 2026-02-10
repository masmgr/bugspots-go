package cmd

import (
	"testing"
	"time"

	"github.com/masmgr/bugspots-go/internal/git"
	"github.com/masmgr/bugspots-go/internal/output"
)

func TestParseRenameDetectFlag(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    git.RenameDetectMode
		wantErr bool
	}{
		{name: "DefaultAuto", input: "", want: git.RenameDetectSimple},
		{name: "OffAlias", input: "false", want: git.RenameDetectOff},
		{name: "SimpleAlias", input: "exact", want: git.RenameDetectSimple},
		{name: "AggressiveAlias", input: "similarity", want: git.RenameDetectAggressive},
		{name: "Invalid", input: "unknown", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseRenameDetectFlag(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("parseRenameDetectFlag(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseDateFlag(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		got, err := parseDateFlag("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != nil {
			t.Fatalf("expected nil, got %v", got)
		}
	})

	t.Run("ValidDate", func(t *testing.T) {
		got, err := parseDateFlag("2025-12-31")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
		if !got.Equal(want) {
			t.Fatalf("parseDateFlag(valid) = %v, want %v", got, want)
		}
	})

	t.Run("InvalidDate", func(t *testing.T) {
		if _, err := parseDateFlag("31-12-2025"); err == nil {
			t.Fatalf("expected error, got nil")
		}
	})
}

func TestGetOutputFormat(t *testing.T) {
	tests := []struct {
		input string
		want  output.OutputFormat
	}{
		{input: "json", want: output.FormatJSON},
		{input: "csv", want: output.FormatCSV},
		{input: "markdown", want: output.FormatMarkdown},
		{input: "md", want: output.FormatMarkdown},
		{input: "ci", want: output.FormatCI},
		{input: "ndjson", want: output.FormatCI},
		{input: "unknown", want: output.FormatConsole},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := getOutputFormat(tt.input); got != tt.want {
				t.Fatalf("getOutputFormat(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
