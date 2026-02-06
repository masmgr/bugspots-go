package aggregation

import (
	"strings"
	"time"

	"github.com/masmgr/bugspots-go/internal/git"
)

// FileMetrics holds aggregated metrics for a single file.
type FileMetrics struct {
	Path                    string
	CommitCount             int
	AddedLines              int
	DeletedLines            int
	LastModifiedAt          time.Time
	Contributors            map[string]struct{}
	ContributorCommitCounts map[string]int
	CommitTimes             []time.Time
	BurstScore              float64
}

// NewFileMetrics creates a new FileMetrics instance.
func NewFileMetrics(path string) *FileMetrics {
	return &FileMetrics{
		Path:                    path,
		Contributors:            make(map[string]struct{}),
		ContributorCommitCounts: make(map[string]int),
		CommitTimes:             make([]time.Time, 0),
	}
}

// ChurnTotal returns total lines changed (added + deleted).
func (f *FileMetrics) ChurnTotal() int {
	return f.AddedLines + f.DeletedLines
}

// ContributorCount returns number of unique contributors.
func (f *FileMetrics) ContributorCount() int {
	return len(f.Contributors)
}

// OwnershipRatio returns proportion of commits by top contributor.
// A high ratio means concentrated ownership (one person owns the file).
// A low ratio means dispersed ownership (many people contribute).
func (f *FileMetrics) OwnershipRatio() float64 {
	if f.CommitCount == 0 || len(f.ContributorCommitCounts) == 0 {
		return 1.0
	}

	maxCommits := 0
	for _, count := range f.ContributorCommitCounts {
		if count > maxCommits {
			maxCommits = count
		}
	}

	return float64(maxCommits) / float64(f.CommitCount)
}

// AddCommit adds a commit's contribution to this file's metrics.
func (f *FileMetrics) AddCommit(commit git.CommitInfo, change git.FileChange) {
	f.CommitCount++
	f.AddedLines += change.LinesAdded
	f.DeletedLines += change.LinesDeleted

	if f.LastModifiedAt.IsZero() || commit.When.After(f.LastModifiedAt) {
		f.LastModifiedAt = commit.When
	}

	contributorKey := strings.ToLower(commit.Author.Email)
	f.Contributors[contributorKey] = struct{}{}
	f.ContributorCommitCounts[contributorKey]++

	f.CommitTimes = append(f.CommitTimes, commit.When)
}

// FileMetricsAggregator aggregates file changes from commits.
type FileMetricsAggregator struct {
	metrics map[string]*FileMetrics
}

// NewFileMetricsAggregator creates a new aggregator.
func NewFileMetricsAggregator() *FileMetricsAggregator {
	return &FileMetricsAggregator{
		metrics: make(map[string]*FileMetrics),
	}
}

// Process processes all commit change sets and aggregates metrics.
func (a *FileMetricsAggregator) Process(changeSets []git.CommitChangeSet) map[string]*FileMetrics {
	for _, cs := range changeSets {
		a.processChangeSet(cs)
	}
	return a.metrics
}

// processChangeSet processes a single commit change set.
func (a *FileMetricsAggregator) processChangeSet(cs git.CommitChangeSet) {
	for _, change := range cs.Changes {
		// Skip deleted files (they don't exist anymore)
		if change.Kind == git.ChangeKindDeleted {
			continue
		}

		path := change.Path

		// Handle renames: if there was an old path, merge its metrics
		if change.Kind == git.ChangeKindRenamed && change.OldPath != "" {
			if oldMetrics, exists := a.metrics[change.OldPath]; exists {
				// Create or get the new path metrics
				if _, newExists := a.metrics[path]; !newExists {
					a.metrics[path] = NewFileMetrics(path)
				}
				// Merge old metrics into new
				a.mergeMetrics(a.metrics[path], oldMetrics)
				delete(a.metrics, change.OldPath)
			}
		}

		// Get or create metrics for this path
		if _, exists := a.metrics[path]; !exists {
			a.metrics[path] = NewFileMetrics(path)
		}

		a.metrics[path].AddCommit(cs.Commit, change)
	}
}

// mergeMetrics merges source metrics into target.
func (a *FileMetricsAggregator) mergeMetrics(target, source *FileMetrics) {
	target.CommitCount += source.CommitCount
	target.AddedLines += source.AddedLines
	target.DeletedLines += source.DeletedLines

	if source.LastModifiedAt.After(target.LastModifiedAt) {
		target.LastModifiedAt = source.LastModifiedAt
	}

	for k := range source.Contributors {
		target.Contributors[k] = struct{}{}
	}

	for k, v := range source.ContributorCommitCounts {
		target.ContributorCommitCounts[k] += v
	}

	target.CommitTimes = append(target.CommitTimes, source.CommitTimes...)
}

// GetMetrics returns the aggregated metrics.
func (a *FileMetricsAggregator) GetMetrics() map[string]*FileMetrics {
	return a.metrics
}
