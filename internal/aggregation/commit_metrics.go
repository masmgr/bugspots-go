package aggregation

import (
	"strings"
	"time"

	"github.com/masmgr/bugspots-go/internal/entropy"
	"github.com/masmgr/bugspots-go/internal/git"
)

// CommitMetrics holds diffusion, size, and entropy metrics for a single commit.
type CommitMetrics struct {
	SHA            string
	When           time.Time
	Author         git.AuthorInfo
	Message        string
	FileCount      int     // NF: Number of files
	DirectoryCount int     // ND: Number of directories
	SubsystemCount int     // NS: Number of subsystems (top-level directories)
	LinesAdded     int     // LA
	LinesDeleted   int     // LD
	ChangeEntropy  float64 // Normalized Shannon entropy
}

// TotalChurn returns the total lines changed (added + deleted).
func (c *CommitMetrics) TotalChurn() int {
	return c.LinesAdded + c.LinesDeleted
}

// CommitMetricsCalculator calculates diffusion, size, and entropy metrics for commits.
type CommitMetricsCalculator struct {
	entropyCalculator *entropy.Calculator
}

// NewCommitMetricsCalculator creates a new commit metrics calculator.
func NewCommitMetricsCalculator() *CommitMetricsCalculator {
	return &CommitMetricsCalculator{
		entropyCalculator: entropy.NewCalculator(),
	}
}

// Calculate computes metrics for a single commit change set.
func (c *CommitMetricsCalculator) Calculate(changeSet git.CommitChangeSet) CommitMetrics {
	commit := changeSet.Commit
	changes := changeSet.Changes

	// Diffusion metrics
	fileCount := len(changes)
	directories := make(map[string]struct{})
	subsystems := make(map[string]struct{})

	// Size metrics
	linesAdded := 0
	linesDeleted := 0

	for _, change := range changes {
		// Accumulate size metrics
		linesAdded += change.LinesAdded
		linesDeleted += change.LinesDeleted

		// Extract directory and subsystem from path
		dir, subsystem := extractPathComponents(change.Path)
		if dir != "" {
			directories[strings.ToLower(dir)] = struct{}{}
		}
		if subsystem != "" {
			subsystems[strings.ToLower(subsystem)] = struct{}{}
		}
	}

	// Calculate entropy
	entropyValue := c.entropyCalculator.CalculateCommitEntropy(changes)

	subsystemCount := len(subsystems)
	if subsystemCount == 0 {
		subsystemCount = 1
	}

	return CommitMetrics{
		SHA:            commit.SHA,
		When:           commit.When,
		Author:         commit.Author,
		Message:        truncateMessage(commit.Message),
		FileCount:      fileCount,
		DirectoryCount: len(directories),
		SubsystemCount: subsystemCount,
		LinesAdded:     linesAdded,
		LinesDeleted:   linesDeleted,
		ChangeEntropy:  entropyValue,
	}
}

// CalculateAll computes metrics for all commit change sets.
func (c *CommitMetricsCalculator) CalculateAll(changeSets []git.CommitChangeSet) []CommitMetrics {
	results := make([]CommitMetrics, 0, len(changeSets))
	for _, cs := range changeSets {
		results = append(results, c.Calculate(cs))
	}
	return results
}

// extractPathComponents extracts directory path and subsystem from a file path.
// Subsystem is the first directory component (e.g., "src", "tests", "docs").
func extractPathComponents(path string) (directory, subsystem string) {
	if path == "" {
		return "", ""
	}

	// Normalize path separators
	normalizedPath := path
	if strings.Contains(path, "\\") {
		normalizedPath = strings.ReplaceAll(path, "\\", "/")
	}

	lastSlash := strings.LastIndex(normalizedPath, "/")
	if lastSlash <= 0 {
		// File is in root directory
		return "", ""
	}

	directory = normalizedPath[:lastSlash]

	// Subsystem is the first directory component
	firstSlash := strings.Index(normalizedPath, "/")
	if firstSlash > 0 {
		subsystem = normalizedPath[:firstSlash]
	} else {
		subsystem = directory
	}

	return directory, subsystem
}

// truncateMessage truncates commit message to first line, max 100 chars.
func truncateMessage(message string) string {
	if message == "" {
		return ""
	}

	// Get first line
	firstNewline := strings.IndexAny(message, "\r\n")
	firstLine := message
	if firstNewline > 0 {
		firstLine = message[:firstNewline]
	}

	// Truncate if too long
	if len(firstLine) > 100 {
		return firstLine[:97] + "..."
	}

	return firstLine
}
