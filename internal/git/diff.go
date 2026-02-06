package git

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// DiffFileEntry represents a file changed between two refs.
type DiffFileEntry struct {
	Path       string
	ChangeKind ChangeKind
	OldPath    string // non-empty for renames
}

// DiffOptions configures a diff operation.
type DiffOptions struct {
	RepoPath string
	DiffSpec string // e.g., "origin/main...HEAD" or "abc123..def456"
}

// DiffResult holds the result of a diff between two refs.
type DiffResult struct {
	Base         string
	Head         string
	ChangedFiles []DiffFileEntry
}

// ParseDiffSpec splits a diff spec into base and head refs.
// Supports both "..." (three-dot) and ".." (two-dot) syntax.
func ParseDiffSpec(spec string) (base, head string, err error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return "", "", fmt.Errorf("empty diff spec")
	}

	// Try three-dot first (merge-base comparison)
	if idx := strings.Index(spec, "..."); idx != -1 {
		base = spec[:idx]
		head = spec[idx+3:]
	} else if idx := strings.Index(spec, ".."); idx != -1 {
		base = spec[:idx]
		head = spec[idx+2:]
	} else {
		return "", "", fmt.Errorf("invalid diff spec %q: expected 'base..head' or 'base...head'", spec)
	}

	if base == "" {
		return "", "", fmt.Errorf("invalid diff spec %q: missing base ref", spec)
	}
	if head == "" {
		head = "HEAD"
	}

	return base, head, nil
}

// ReadDiff reads the list of changed files between two refs.
func ReadDiff(ctx context.Context, opts DiffOptions) (*DiffResult, error) {
	base, head, err := ParseDiffSpec(opts.DiffSpec)
	if err != nil {
		return nil, err
	}

	args := []string{
		"-C", opts.RepoPath,
		"diff",
		"--name-status",
		"-z",
		opts.DiffSpec,
	}

	out, err := exec.CommandContext(ctx, "git", args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git diff failed: %w: %s", err, strings.TrimSpace(string(out)))
	}

	entries, err := parseDiffNameStatus(out)
	if err != nil {
		return nil, err
	}

	return &DiffResult{
		Base:         base,
		Head:         head,
		ChangedFiles: entries,
	}, nil
}

// parseDiffNameStatus parses NUL-delimited `git diff --name-status -z` output.
// Format: STATUS\0PATH\0 (or STATUS\0OLDPATH\0NEWPATH\0 for renames/copies)
func parseDiffNameStatus(data []byte) ([]DiffFileEntry, error) {
	// Split by NUL
	parts := bytes.Split(data, []byte{0x00})

	entries := make([]DiffFileEntry, 0, len(parts)/2)
	i := 0

	for i < len(parts) {
		status := strings.TrimSpace(string(parts[i]))
		if status == "" {
			i++
			continue
		}

		if i+1 >= len(parts) {
			break
		}

		kind, isRename := diffStatusToChangeKind(status)

		if isRename {
			// Rename/Copy: STATUS\0OLDPATH\0NEWPATH
			if i+2 >= len(parts) {
				return nil, fmt.Errorf("unexpected diff output: rename entry missing new path")
			}
			oldPath := string(parts[i+1])
			newPath := string(parts[i+2])
			entries = append(entries, DiffFileEntry{
				Path:       newPath,
				ChangeKind: kind,
				OldPath:    oldPath,
			})
			i += 3
		} else {
			path := string(parts[i+1])
			entries = append(entries, DiffFileEntry{
				Path:       path,
				ChangeKind: kind,
			})
			i += 2
		}
	}

	return entries, nil
}

// diffStatusToChangeKind converts a git diff status letter to ChangeKind.
// Returns the kind and whether it's a rename/copy (which has two paths).
func diffStatusToChangeKind(status string) (ChangeKind, bool) {
	if len(status) == 0 {
		return ChangeKindModified, false
	}
	switch status[0] {
	case 'A':
		return ChangeKindAdded, false
	case 'D':
		return ChangeKindDeleted, false
	case 'R':
		return ChangeKindRenamed, true
	case 'C':
		// Copy is treated like Added for our purposes, but has two paths
		return ChangeKindAdded, true
	default:
		return ChangeKindModified, false
	}
}
