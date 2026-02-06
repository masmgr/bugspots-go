package aggregation

import (
	"strings"
	"testing"
)

func TestCommitMetrics_TotalChurn(t *testing.T) {
	tests := []struct {
		name     string
		added    int
		deleted  int
		expected int
	}{
		{name: "Both positive", added: 10, deleted: 5, expected: 15},
		{name: "Only added", added: 10, deleted: 0, expected: 10},
		{name: "Both zero", added: 0, deleted: 0, expected: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := &CommitMetrics{LinesAdded: tt.added, LinesDeleted: tt.deleted}
			result := cm.TotalChurn()
			if result != tt.expected {
				t.Errorf("TotalChurn() = %d, expected %d", result, tt.expected)
			}
		})
	}
}

func TestExtractPathComponents(t *testing.T) {
	tests := []struct {
		name              string
		path              string
		expectedDir       string
		expectedSubsystem string
	}{
		{name: "Normal path", path: "src/pkg/main.go", expectedDir: "src/pkg", expectedSubsystem: "src"},
		{name: "Root file", path: "main.go", expectedDir: "", expectedSubsystem: ""},
		{name: "Single directory", path: "cmd/app.go", expectedDir: "cmd", expectedSubsystem: "cmd"},
		{name: "Deep nesting", path: "a/b/c/d/e.go", expectedDir: "a/b/c/d", expectedSubsystem: "a"},
		{name: "Windows path", path: "src\\pkg\\main.go", expectedDir: "src/pkg", expectedSubsystem: "src"},
		{name: "Empty path", path: "", expectedDir: "", expectedSubsystem: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir, subsystem := extractPathComponents(tt.path)
			if dir != tt.expectedDir {
				t.Errorf("extractPathComponents(%q) dir = %q, expected %q", tt.path, dir, tt.expectedDir)
			}
			if subsystem != tt.expectedSubsystem {
				t.Errorf("extractPathComponents(%q) subsystem = %q, expected %q", tt.path, subsystem, tt.expectedSubsystem)
			}
		})
	}
}

func TestTruncateMessage(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		expected string
	}{
		{name: "Short message", message: "fix bug", expected: "fix bug"},
		{name: "Empty message", message: "", expected: ""},
		{name: "Exactly 100 chars", message: strings.Repeat("a", 100), expected: strings.Repeat("a", 100)},
		{name: "Over 100 chars", message: strings.Repeat("a", 110), expected: strings.Repeat("a", 97) + "..."},
		{name: "Multi-line with LF", message: "first line\nsecond line", expected: "first line"},
		{name: "Multi-line with CRLF", message: "first line\r\nsecond line", expected: "first line"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateMessage(tt.message)
			if result != tt.expected {
				t.Errorf("truncateMessage(%q) = %q, expected %q", tt.message, result, tt.expected)
			}
		})
	}
}
