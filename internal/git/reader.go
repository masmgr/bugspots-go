package git

import (
	"fmt"
	"strings"
	"time"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// HistoryReader reads commit history from a Git repository.
type HistoryReader struct {
	repo        *git.Repository
	opts        ReadOptions
	filterCache map[string]bool // Cache for pattern matching results
}

// NewHistoryReader creates a new history reader for the given repository.
func NewHistoryReader(opts ReadOptions) (*HistoryReader, error) {
	repo, err := git.PlainOpen(opts.RepoPath)
	if err != nil {
		return nil, err
	}
	return &HistoryReader{
		repo:        repo,
		opts:        opts,
		filterCache: make(map[string]bool),
	}, nil
}

// ReadChanges reads commit changes from the repository.
// It returns a channel of CommitChangeSet for streaming processing.
func (r *HistoryReader) ReadChanges() ([]CommitChangeSet, error) {
	fromHash, err := r.resolveFromHash()
	if err != nil {
		return nil, err
	}

	logOpts := &git.LogOptions{From: fromHash}

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

	results := make([]CommitChangeSet, 0, 1000)

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

func (r *HistoryReader) resolveFromHash() (plumbing.Hash, error) {
	branch := strings.TrimSpace(r.opts.Branch)
	if branch == "" || strings.EqualFold(branch, "HEAD") {
		ref, err := r.repo.Head()
		if err != nil {
			return plumbing.ZeroHash, err
		}
		return ref.Hash(), nil
	}

	remoteRef := plumbing.ReferenceName("")
	if !strings.HasPrefix(branch, "refs/") && strings.Contains(branch, "/") {
		if parts := strings.SplitN(branch, "/", 2); len(parts) == 2 && parts[0] != "" && parts[1] != "" {
			remoteRef = plumbing.NewRemoteReferenceName(parts[0], parts[1])
		}
	}

	candidates := []plumbing.ReferenceName{
		// Full reference name (e.g., refs/heads/main)
		plumbing.ReferenceName(branch),
		// Local branch name (e.g., main)
		plumbing.NewBranchReferenceName(branch),
		// Remote branch name (e.g., origin/main)
		plumbing.NewRemoteReferenceName("origin", branch),
		// Remote branch name when passed as <remote>/<branch> (e.g., origin/main)
		remoteRef,
		// Tag name (e.g., v1.2.3)
		plumbing.NewTagReferenceName(branch),
	}

	for _, name := range candidates {
		if name == "" {
			continue
		}
		ref, err := r.repo.Reference(name, true)
		if err == nil {
			return ref.Hash(), nil
		}
	}

	// As a last resort, allow revisions (e.g., HEAD~10, a commit SHA).
	h, err := r.repo.ResolveRevision(plumbing.Revision(branch))
	if err == nil && h != nil {
		return *h, nil
	}

	return plumbing.ZeroHash, fmt.Errorf("branch/ref not found: %q", branch)
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

	filePatches := patch.FilePatches()
	changes := make([]FileChange, 0, len(filePatches))

	for _, filePatch := range filePatches {
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
		matches, err := r.matchesFilters(path)
		if err != nil {
			return nil, err
		}
		if !matches {
			continue
		}

		// Calculate line stats using strings.Count to avoid allocations
		var added, deleted int
		for _, chunk := range filePatch.Chunks() {
			content := chunk.Content()
			lineCount := strings.Count(content, "\n")
			if len(content) > 0 && content[len(content)-1] != '\n' {
				lineCount++
			}
			switch chunk.Type() {
			case 1: // Add
				added += lineCount
			case 2: // Delete
				deleted += lineCount
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
// Results are cached to avoid repeated pattern matching for the same path.
func (r *HistoryReader) matchesFilters(path string) (bool, error) {
	// Normalize path separators
	path = strings.ReplaceAll(path, "\\", "/")

	// Check cache first
	if result, ok := r.filterCache[path]; ok {
		return result, nil
	}

	// Check exclude patterns first
	for _, pattern := range r.opts.Exclude {
		matched, err := doublestar.Match(pattern, path)
		if err != nil {
			return false, fmt.Errorf("invalid exclude glob pattern %q: %w", pattern, err)
		}
		if matched {
			r.filterCache[path] = false
			return false, nil
		}
	}

	// If no include patterns, accept all
	if len(r.opts.Include) == 0 {
		r.filterCache[path] = true
		return true, nil
	}

	// Check include patterns
	for _, pattern := range r.opts.Include {
		matched, err := doublestar.Match(pattern, path)
		if err != nil {
			return false, fmt.Errorf("invalid include glob pattern %q: %w", pattern, err)
		}
		if matched {
			r.filterCache[path] = true
			return true, nil
		}
	}

	r.filterCache[path] = false
	return false, nil
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
