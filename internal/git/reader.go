package git

import (
	"strings"
	"time"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// HistoryReader reads commit history from a Git repository.
type HistoryReader struct {
	repo *git.Repository
	opts ReadOptions
}

// NewHistoryReader creates a new history reader for the given repository.
func NewHistoryReader(opts ReadOptions) (*HistoryReader, error) {
	repo, err := git.PlainOpen(opts.RepoPath)
	if err != nil {
		return nil, err
	}
	return &HistoryReader{repo: repo, opts: opts}, nil
}

// ReadChanges reads commit changes from the repository.
// It returns a channel of CommitChangeSet for streaming processing.
func (r *HistoryReader) ReadChanges() ([]CommitChangeSet, error) {
	ref, err := r.repo.Head()
	if err != nil {
		return nil, err
	}

	logOpts := &git.LogOptions{From: ref.Hash()}

	if r.opts.Since != nil {
		logOpts.Since = r.opts.Since
	}
	if r.opts.Until != nil {
		logOpts.Until = r.opts.Until
	}

	cIter, err := r.repo.Log(logOpts)
	if err != nil {
		return nil, err
	}

	var results []CommitChangeSet

	err = cIter.ForEach(func(c *object.Commit) error {
		// Skip commits without parents (initial commit)
		if c.NumParents() == 0 {
			return nil
		}

		changes, err := r.getCommitChanges(c)
		if err != nil {
			return err
		}

		if len(changes) == 0 {
			return nil
		}

		// Extract first line of commit message
		message := c.Message
		if idx := strings.IndexByte(message, '\n'); idx != -1 {
			message = message[:idx]
		}

		commitInfo := CommitInfo{
			SHA:     c.Hash.String(),
			When:    c.Committer.When,
			Author:  AuthorInfo{Name: c.Author.Name, Email: c.Author.Email},
			Message: message,
		}

		results = append(results, CommitChangeSet{
			Commit:  commitInfo,
			Changes: changes,
		})

		return nil
	})

	if err != nil {
		return nil, err
	}

	return results, nil
}

// getCommitChanges extracts file changes from a commit.
func (r *HistoryReader) getCommitChanges(c *object.Commit) ([]FileChange, error) {
	parent, err := c.Parent(0)
	if err != nil {
		return nil, err
	}

	patch, err := parent.Patch(c)
	if err != nil {
		return nil, err
	}

	var changes []FileChange

	for _, filePatch := range patch.FilePatches() {
		from, to := filePatch.Files()

		var path, oldPath string
		var kind ChangeKind

		switch {
		case from == nil && to != nil:
			// Added
			path = to.Path()
			kind = ChangeKindAdded
		case from != nil && to == nil:
			// Deleted
			path = from.Path()
			kind = ChangeKindDeleted
		case from != nil && to != nil && from.Path() != to.Path():
			// Renamed
			path = to.Path()
			oldPath = from.Path()
			kind = ChangeKindRenamed
		default:
			// Modified
			if to != nil {
				path = to.Path()
			} else if from != nil {
				path = from.Path()
			}
			kind = ChangeKindModified
		}

		if path == "" {
			continue
		}

		// Apply filters
		if !r.matchesFilters(path) {
			continue
		}

		// Calculate line stats
		var added, deleted int
		for _, chunk := range filePatch.Chunks() {
			content := chunk.Content()
			lines := strings.Split(content, "\n")
			switch chunk.Type() {
			case 1: // Add
				added += len(lines)
			case 2: // Delete
				deleted += len(lines)
			}
		}

		changes = append(changes, FileChange{
			Path:         path,
			OldPath:      oldPath,
			LinesAdded:   added,
			LinesDeleted: deleted,
			Kind:         kind,
		})
	}

	return changes, nil
}

// matchesFilters checks if a path matches the include/exclude filters.
func (r *HistoryReader) matchesFilters(path string) bool {
	// Normalize path separators
	path = strings.ReplaceAll(path, "\\", "/")

	// Check exclude patterns first
	for _, pattern := range r.opts.Exclude {
		matched, _ := doublestar.Match(pattern, path)
		if matched {
			return false
		}
	}

	// If no include patterns, accept all
	if len(r.opts.Include) == 0 {
		return true
	}

	// Check include patterns
	for _, pattern := range r.opts.Include {
		matched, _ := doublestar.Match(pattern, path)
		if matched {
			return true
		}
	}

	return false
}

// ReadChangesWithDateRange is a convenience method to read changes within a date range.
func ReadChangesWithDateRange(repoPath string, since, until time.Time) ([]CommitChangeSet, error) {
	reader, err := NewHistoryReader(ReadOptions{
		RepoPath: repoPath,
		Since:    &since,
		Until:    &until,
	})
	if err != nil {
		return nil, err
	}
	return reader.ReadChanges()
}
