package main

import (
	"regexp"
	"testing"
)

// TestConvertToRegex tests the convertToRegex function
func TestConvertToRegex(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Single word",
			input:    "fix",
			expected: "fix",
		},
		{
			name:     "Two words",
			input:    "fix,close",
			expected: "fix|close",
		},
		{
			name:     "Multiple words",
			input:    "fix,close,resolve,closes",
			expected: "fix|close|resolve|closes",
		},
		{
			name:     "Words with spaces (trimmed or not)",
			input:    "fixed,resolved",
			expected: "fixed|resolved",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertToRegex(tt.input)
			if result != tt.expected {
				t.Errorf("convertToRegex(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestConvertToRegex_RegexValidity tests that the output can be compiled as a valid regex
func TestConvertToRegex_RegexValidity(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "Single word", input: "fix"},
		{name: "Two words", input: "fix,close"},
		{name: "Multiple words", input: "fixes,closes,resolved,resolves"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			regexStr := convertToRegex(tt.input)
			_, err := regexp.Compile(regexStr)
			if err != nil {
				t.Errorf("convertToRegex(%q) produced invalid regex: %v", tt.input, err)
			}
		})
	}
}

// TestConvertToRegex_RegexMatching tests that the converted regex matches expected patterns
func TestConvertToRegex_RegexMatching(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		testStrings []string
		shouldMatch []bool
	}{
		{
			name:        "Match single word",
			input:       "fix",
			testStrings: []string{"fix", "fixed", "fixing", "notafix"},
			shouldMatch: []bool{true, true, true, true},
		},
		{
			name:        "Match multiple words",
			input:       "fix,close",
			testStrings: []string{"fix", "close", "closes", "resolve"},
			shouldMatch: []bool{true, true, true, false},
		},
		{
			name:        "Case sensitive matching",
			input:       "fix,close",
			testStrings: []string{"Fix", "CLOSE", "fix"},
			shouldMatch: []bool{false, false, true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			regexStr := convertToRegex(tt.input)
			regex := regexp.MustCompile(regexStr)

			for i, testStr := range tt.testStrings {
				matches := regex.MatchString(testStr)
				if matches != tt.shouldMatch[i] {
					t.Errorf("Regex from %q should match %q: %v, expected %v",
						tt.input, testStr, matches, tt.shouldMatch[i])
				}
			}
		})
	}
}

// TestConvertToRegex_EdgeCases tests edge cases
func TestConvertToRegex_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Single word with no commas",
			input:    "bugfix",
			expected: "bugfix",
		},
		{
			name:     "Trailing comma",
			input:    "fix,close,",
			expected: "fix|close|",
		},
		{
			name:     "Leading comma",
			input:    ",fix,close",
			expected: "|fix|close",
		},
		{
			name:     "Multiple consecutive commas",
			input:    "fix,,close",
			expected: "fix||close",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertToRegex(tt.input)
			if result != tt.expected {
				t.Errorf("convertToRegex(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}
