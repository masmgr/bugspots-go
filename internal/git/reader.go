package git

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/bmatcuk/doublestar/v4"
)

// HistoryReader reads commit history from a Git repository.
type HistoryReader struct {
	opts        ReadOptions
	filterCache map[string]bool // Cache for pattern matching results
}

// NewHistoryReader creates a new history reader for the given repository.
func NewHistoryReader(opts ReadOptions) (*HistoryReader, error) {
	out, err := exec.Command("git", "-C", opts.RepoPath, "rev-parse", "--git-dir").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("invalid Git repository %q: %w: %s", opts.RepoPath, err, strings.TrimSpace(string(out)))
	}
	return &HistoryReader{
		opts:        opts,
		filterCache: make(map[string]bool),
	}, nil
}

// ReadChanges reads commit changes from the repository.
// The provided context controls cancellation of the operation.
func (r *HistoryReader) ReadChanges(ctx context.Context) ([]CommitChangeSet, error) {
	return r.readChangesGitCLI(ctx)
}

// matchesFilters checks if a path matches the include/exclude filters.
// Results are cached to avoid repeated pattern matching for the same path.
func (r *HistoryReader) matchesFilters(path string) (bool, error) {
	// Normalize path separators
	path = strings.ReplaceAll(path, "\\", "/")

	// Check cache first
	if result, ok := r.filterCache[path]; ok {
		return result, nil
	}

	result, err := MatchesGlobFilters(path, r.opts.Include, r.opts.Exclude)
	if err != nil {
		return false, err
	}
	r.filterCache[path] = result
	return result, nil
}

// MatchesGlobFilters checks if a path matches the given include/exclude glob patterns.
// If include is empty, all non-excluded paths match.
// The path should already be normalized (forward slashes).
func MatchesGlobFilters(path string, include, exclude []string) (bool, error) {
	// Check exclude patterns first
	for _, pattern := range exclude {
		matched, err := doublestar.Match(pattern, path)
		if err != nil {
			return false, fmt.Errorf("invalid exclude glob pattern %q: %w", pattern, err)
		}
		if matched {
			return false, nil
		}
	}

	// If no include patterns, accept all
	if len(include) == 0 {
		return true, nil
	}

	// Check include patterns
	for _, pattern := range include {
		matched, err := doublestar.Match(pattern, path)
		if err != nil {
			return false, fmt.Errorf("invalid include glob pattern %q: %w", pattern, err)
		}
		if matched {
			return true, nil
		}
	}

	return false, nil
}

// ReadChangesWithDateRange is a convenience method to read changes within a date range.
func ReadChangesWithDateRange(ctx context.Context, repoPath string, since, until time.Time) ([]CommitChangeSet, error) {
	reader, err := NewHistoryReader(ReadOptions{
		RepoPath: repoPath,
		Since:    &since,
		Until:    &until,
	})
	if err != nil {
		return nil, err
	}
	return reader.ReadChanges(ctx)
}
