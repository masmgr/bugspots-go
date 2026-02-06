package git

// RepositoryReader defines the interface for reading Git repository history.
// This abstraction allows for easier testing and potential alternative implementations.
type RepositoryReader interface {
	// ReadChanges reads the commit history and returns a slice of CommitChangeSet.
	ReadChanges() ([]CommitChangeSet, error)
}

// Compile-time interface conformance check.
var _ RepositoryReader = (*HistoryReader)(nil)
