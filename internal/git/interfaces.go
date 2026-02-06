package git

import "context"

// RepositoryReader defines the interface for reading Git repository history.
// This abstraction allows for easier testing and potential alternative implementations.
type RepositoryReader interface {
	// ReadChanges reads the commit history and returns a slice of CommitChangeSet.
	// The provided context controls cancellation; pass context.Background() if no cancellation is needed.
	ReadChanges(ctx context.Context) ([]CommitChangeSet, error)
}

// Compile-time interface conformance check.
var _ RepositoryReader = (*HistoryReader)(nil)
