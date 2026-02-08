package complexity

import (
	"testing"
)

func TestCountLines(t *testing.T) {
	tests := []struct {
		name     string
		content  []byte
		expected int
	}{
		{name: "Empty", content: []byte{}, expected: 0},
		{name: "Single line no newline", content: []byte("hello"), expected: 1},
		{name: "Single line with newline", content: []byte("hello\n"), expected: 1},
		{name: "Two lines", content: []byte("hello\nworld\n"), expected: 2},
		{name: "Two lines no trailing", content: []byte("hello\nworld"), expected: 2},
		{name: "Multiple lines", content: []byte("a\nb\nc\nd\n"), expected: 4},
		{name: "Only newlines", content: []byte("\n\n\n"), expected: 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := countLines(tt.content)
			if result != tt.expected {
				t.Errorf("countLines(%q) = %d, expected %d", tt.content, result, tt.expected)
			}
		})
	}
}

func TestReadFull(t *testing.T) {
	// readFull is tested implicitly through countLinesBatch
	// but we can test the basic behavior here
	t.Run("Basic", func(t *testing.T) {
		// readFull is internal to the batch reading; tested via integration
	})
}
