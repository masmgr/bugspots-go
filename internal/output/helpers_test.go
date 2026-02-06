package output

import "testing"

func TestTruncateMessage_Output(t *testing.T) {
	tests := []struct {
		name     string
		msg      string
		maxLen   int
		expected string
	}{
		{name: "Short message", msg: "hello", maxLen: 40, expected: "hello"},
		{name: "Exact length", msg: "1234567890", maxLen: 10, expected: "1234567890"},
		{name: "Over max length", msg: "a very long message here", maxLen: 10, expected: "a very ..."},
		{name: "Empty message", msg: "", maxLen: 40, expected: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateMessage(tt.msg, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncateMessage(%q, %d) = %q, expected %q", tt.msg, tt.maxLen, result, tt.expected)
			}
		})
	}
}

func TestGetRiskLevelEmoji(t *testing.T) {
	tests := []struct {
		name     string
		level    string
		expected string
	}{
		{name: "High", level: "high", expected: "\U0001F534"},
		{name: "Medium", level: "medium", expected: "\U0001F7E1"},
		{name: "Low", level: "low", expected: "\U0001F7E2"},
		{name: "Unknown", level: "critical", expected: "\U0001F7E2"},
		{name: "Empty", level: "", expected: "\U0001F7E2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getRiskLevelEmoji(tt.level)
			if result != tt.expected {
				t.Errorf("getRiskLevelEmoji(%q) = %q, expected %q", tt.level, result, tt.expected)
			}
		})
	}
}

func TestEscapeMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "Pipe", input: "a|b", expected: "a\\|b"},
		{name: "Asterisk", input: "a*b", expected: "a\\*b"},
		{name: "Underscore", input: "a_b", expected: "a\\_b"},
		{name: "Backtick", input: "a`b", expected: "a\\`b"},
		{name: "Multiple specials", input: "a|b*c_d", expected: "a\\|b\\*c\\_d"},
		{name: "No specials", input: "plain text", expected: "plain text"},
		{name: "Empty", input: "", expected: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeMarkdown(tt.input)
			if result != tt.expected {
				t.Errorf("escapeMarkdown(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}
