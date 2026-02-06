package git

import (
	"strings"
	"time"
)

// CommitInfo represents minimal information about a Git commit.
type CommitInfo struct {
	SHA     string
	When    time.Time
	Author  AuthorInfo
	Message string
}

// AuthorInfo represents commit author information.
type AuthorInfo struct {
	Name  string
	Email string
}

// ContributorKey returns a normalized identifier for grouping contributors.
func (a AuthorInfo) ContributorKey() string {
	return strings.ToLower(a.Email)
}

// FileChange represents a file change within a commit.
type FileChange struct {
	Path         string
	OldPath      string // For renames
	LinesAdded   int
	LinesDeleted int
	Kind         ChangeKind
}

// Churn returns total lines changed (added + deleted).
func (f FileChange) Churn() int {
	return f.LinesAdded + f.LinesDeleted
}

// ChangeKind represents the type of change.
type ChangeKind int

const (
	ChangeKindAdded ChangeKind = iota
	ChangeKindModified
	ChangeKindDeleted
	ChangeKindRenamed
)

// String returns a string representation of the change kind.
func (k ChangeKind) String() string {
	switch k {
	case ChangeKindAdded:
		return "added"
	case ChangeKindModified:
		return "modified"
	case ChangeKindDeleted:
		return "deleted"
	case ChangeKindRenamed:
		return "renamed"
	default:
		return "unknown"
	}
}

// CommitChangeSet bundles a commit with its file changes.
type CommitChangeSet struct {
	Commit  CommitInfo
	Changes []FileChange
}

// RenameDetectMode controls how file renames are detected.
type RenameDetectMode int

const (
	RenameDetectOff RenameDetectMode = iota
	RenameDetectSimple
	RenameDetectAggressive
)

// ReadOptions configures the history reader.
type ReadOptions struct {
	RepoPath     string
	Branch       string
	Since        *time.Time
	Until        *time.Time
	Include      []string // Glob patterns to include
	Exclude      []string // Glob patterns to exclude
	RenameDetect RenameDetectMode
}
