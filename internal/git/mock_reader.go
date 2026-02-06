package git

import "context"

// MockHistoryReader is a test double for HistoryReader.
// It allows tests to provide predefined commit data without needing a real Git repository.
type MockHistoryReader struct {
	ChangeSets []CommitChangeSet
	Error      error
}

// NewMockHistoryReader creates a new MockHistoryReader with the given data.
func NewMockHistoryReader(changeSets []CommitChangeSet, err error) *MockHistoryReader {
	return &MockHistoryReader{
		ChangeSets: changeSets,
		Error:      err,
	}
}

// ReadChanges returns the predefined change sets or error.
func (m *MockHistoryReader) ReadChanges(_ context.Context) ([]CommitChangeSet, error) {
	return m.ChangeSets, m.Error
}

// Compile-time interface conformance check.
var _ RepositoryReader = (*MockHistoryReader)(nil)
