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
	cachedOwnershipRatio    *float64 // Cached ownership ratio to avoid repeated calculation
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
// Results are cached for performance.
func (f *FileMetrics) OwnershipRatio() float64 {
	if f.cachedOwnershipRatio != nil {
		return *f.cachedOwnershipRatio
	}

	var ratio float64
	if f.CommitCount == 0 || len(f.ContributorCommitCounts) == 0 {
		ratio = 1.0
	} else {
		maxCommits := 0
		for _, count := range f.ContributorCommitCounts {
			if count > maxCommits {
				maxCommits = count
			}
		}
		ratio = float64(maxCommits) / float64(f.CommitCount)
	}

	f.cachedOwnershipRatio = &ratio
	return ratio
}

// AddCommit adds a commit's contribution to this file's metrics.
func (f *FileMetrics) AddCommit(commit git.CommitInfo, change git.FileChange, collectCommitTimes bool) {
	f.CommitCount++
	f.AddedLines += change.LinesAdded
	f.DeletedLines += change.LinesDeleted

	if f.LastModifiedAt.IsZero() || commit.When.After(f.LastModifiedAt) {
		f.LastModifiedAt = commit.When
	}

	contributorKey := strings.ToLower(commit.Author.Email)
	f.Contributors[contributorKey] = struct{}{}
	f.ContributorCommitCounts[contributorKey]++

	// Only collect commit times if needed for burst calculation
	if collectCommitTimes {
		f.CommitTimes = append(f.CommitTimes, commit.When)
	}

	// Invalidate cached ownership ratio when metrics change
	f.cachedOwnershipRatio = nil
}

// FileMetricsAggregator aggregates file changes from commits.
type FileMetricsAggregator struct {
	metrics            map[string]*FileMetrics
	collectCommitTimes bool // Whether to collect commit times for burst calculation
}

// NewFileMetricsAggregator creates a new aggregator.
// By default, commit times are collected for burst calculation.
func NewFileMetricsAggregator() *FileMetricsAggregator {
	return &FileMetricsAggregator{
		metrics:            make(map[string]*FileMetrics),
		collectCommitTimes: true,
	}
}

// NewFileMetricsAggregatorWithOptions creates a new aggregator with options.
func NewFileMetricsAggregatorWithOptions(collectCommitTimes bool) *FileMetricsAggregator {
	return &FileMetricsAggregator{
		metrics:            make(map[string]*FileMetrics),
		collectCommitTimes: collectCommitTimes,
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

		a.metrics[path].AddCommit(cs.Commit, change, a.collectCommitTimes)
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
