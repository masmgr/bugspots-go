package git

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	fdiff "github.com/go-git/go-git/v5/plumbing/format/diff"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
)

// HistoryReader reads commit history from a Git repository.
type HistoryReader struct {
	repo        *git.Repository
	opts        ReadOptions
	filterCache map[string]bool // Cache for pattern matching results

	// One-entry tree cache: in linear history the parent tree of commit N
	// is the current tree of commit N-1, so we can avoid re-decoding it.
	lastTreeHash plumbing.Hash
	lastTree     *object.Tree
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
// The provided context controls cancellation of the operation.
func (r *HistoryReader) ReadChanges(ctx context.Context) ([]CommitChangeSet, error) {
	fromHash, err := r.resolveFromHash()
	if err != nil {
		return nil, err
	}

	// Use LogOrderCommitterTime so commits arrive in chronological order
	// (newest first). This lets us stop early once we pass the Since boundary,
	// instead of walking the entire commit graph.
	logOpts := &git.LogOptions{
		From:  fromHash,
		Order: git.LogOrderCommitterTime,
	}

	cIter, err := r.repo.Log(logOpts)
	if err != nil {
		return nil, err
	}
	defer cIter.Close()

	results := make([]CommitChangeSet, 0, 1000)
	processed := 0

	err = cIter.ForEach(func(c *object.Commit) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Manual time filtering with early termination.
		// Because commits are ordered by committer time (newest first),
		// once we see a commit before Since, all remaining are older too.
		when := c.Committer.When
		if r.opts.Until != nil && when.After(*r.opts.Until) {
			return nil // skip: too new
		}
		if r.opts.Since != nil && when.Before(*r.opts.Since) {
			return storer.ErrStop // early termination: all remaining are older
		}

		// Skip commits without parents (initial commit)
		if c.NumParents() == 0 {
			return nil
		}

		// Skip merge commits
		if c.NumParents() > 1 {
			return nil
		}

		changes, err := r.getCommitChanges(ctx, c)
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

		processed++
		if r.opts.OnProgress != nil {
			r.opts.OnProgress(processed)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return results, nil
}

func (r *HistoryReader) diffTreeOptions() *object.DiffTreeOptions {
	switch r.opts.RenameDetect {
	case RenameDetectOff:
		return &object.DiffTreeOptions{DetectRenames: false}
	case RenameDetectSimple:
		// Exact renames only; avoids content similarity work.
		return &object.DiffTreeOptions{
			DetectRenames:    true,
			RenameScore:      100,
			RenameLimit:      0,
			OnlyExactRenames: true,
		}
	case RenameDetectAggressive:
		fallthrough
	default:
		// Copy to avoid accidental shared mutation.
		opts := *object.DefaultDiffTreeOptions
		return &opts
	}
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
func (r *HistoryReader) getCommitChanges(ctx context.Context, c *object.Commit) ([]FileChange, error) {
	parent, err := c.Parent(0)
	if err != nil {
		return nil, err
	}

	// Fast path: identical tree hashes mean no file changes at all.
	if parent.TreeHash == c.TreeHash {
		return nil, nil
	}

	// Reuse cached tree when possible. In linear history the parent tree of
	// the current commit is the current tree of the previous commit.
	var parentTree *object.Tree
	if r.lastTreeHash == parent.TreeHash && r.lastTree != nil {
		parentTree = r.lastTree
	} else {
		parentTree, err = parent.Tree()
		if err != nil {
			return nil, err
		}
	}

	tree, err := c.Tree()
	if err != nil {
		return nil, err
	}

	// Cache this commit's tree for the next iteration.
	r.lastTreeHash = c.TreeHash
	r.lastTree = tree

	changes, err := object.DiffTreeWithOptions(ctx, parentTree, tree, r.diffTreeOptions())
	if err != nil {
		return nil, err
	}

	switch r.opts.DetailLevel {
	case ChangeDetailPathsOnly:
		return r.changesFromTreeDiff(changes)
	default:
		return r.changesWithLineStats(ctx, changes)
	}
}

// extractPathAndKind determines the file path, old path, and change kind
// from a single Change entry. It returns an empty path when the change
// should be skipped (e.g. both names are empty).
func extractPathAndKind(change *object.Change) (path, oldPath string, kind ChangeKind) {
	switch {
	case change.From.Name == "" && change.To.Name != "":
		return change.To.Name, "", ChangeKindAdded
	case change.From.Name != "" && change.To.Name == "":
		return change.From.Name, "", ChangeKindDeleted
	case change.From.Name != "" && change.To.Name != "" && change.From.Name != change.To.Name:
		return change.To.Name, change.From.Name, ChangeKindRenamed
	default:
		p := change.To.Name
		if p == "" {
			p = change.From.Name
		}
		return p, "", ChangeKindModified
	}
}

func (r *HistoryReader) changesFromTreeDiff(changes object.Changes) ([]FileChange, error) {
	results := make([]FileChange, 0, len(changes))

	for _, change := range changes {
		// Skip non-file entries (directories, submodules, etc.).
		if !change.From.TreeEntry.Mode.IsFile() && !change.To.TreeEntry.Mode.IsFile() {
			continue
		}

		path, oldPath, kind := extractPathAndKind(change)
		if path == "" {
			continue
		}

		matches, err := r.matchesFilters(path)
		if err != nil {
			return nil, err
		}
		if !matches {
			continue
		}

		results = append(results, FileChange{
			Path:         path,
			OldPath:      oldPath,
			LinesAdded:   0,
			LinesDeleted: 0,
			Kind:         kind,
		})
	}

	return results, nil
}

func (r *HistoryReader) changesWithLineStats(ctx context.Context, changes object.Changes) ([]FileChange, error) {
	filtered := make(object.Changes, 0, len(changes))

	for _, change := range changes {
		// Skip non-file entries (directories, submodules, etc.).
		if !change.From.TreeEntry.Mode.IsFile() && !change.To.TreeEntry.Mode.IsFile() {
			continue
		}

		path := change.To.Name
		if path == "" {
			path = change.From.Name
		}
		if path == "" {
			continue
		}

		matches, err := r.matchesFilters(path)
		if err != nil {
			return nil, err
		}
		if !matches {
			continue
		}

		filtered = append(filtered, change)
	}

	if len(filtered) == 0 {
		return nil, nil
	}

	// Process each file individually instead of generating a bulk patch.
	// This reduces peak memory: only one file's diff data is held at a time,
	// and supports cancellation between files via context.
	results := make([]FileChange, 0, len(filtered))

	for _, change := range filtered {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Fast path: when blob hashes are identical the content is unchanged
		// (e.g. mode-only change like chmod +x). Skip the expensive
		// PatchContext call which would read both blobs and run Myers diff.
		if change.From.TreeEntry.Hash == change.To.TreeEntry.Hash {
			path, oldPath, kind := extractPathAndKind(change)
			if path != "" {
				results = append(results, FileChange{
					Path:    path,
					OldPath: oldPath,
					Kind:    kind,
				})
			}
			continue
		}

		patch, err := change.PatchContext(ctx)
		if err != nil {
			return nil, err
		}

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

			// Calculate line stats using strings.Count to avoid allocations.
			var added, deleted int
			for _, chunk := range filePatch.Chunks() {
				content := chunk.Content()
				if len(content) == 0 {
					continue
				}

				lineCount := strings.Count(content, "\n")
				if content[len(content)-1] != '\n' {
					lineCount++
				}
				switch chunk.Type() {
				case fdiff.Add:
					added += lineCount
				case fdiff.Delete:
					deleted += lineCount
				}
			}

			results = append(results, FileChange{
				Path:         path,
				OldPath:      oldPath,
				LinesAdded:   added,
				LinesDeleted: deleted,
				Kind:         kind,
			})
		}
	}

	return results, nil
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
