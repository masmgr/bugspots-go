package git

import (
	"errors"
	"testing"
	"time"
)

func TestMockHistoryReader_ReadChanges(t *testing.T) {
	// Create test data
	expectedChangeSets := []CommitChangeSet{
		{
			Commit: CommitInfo{
				SHA:     "abc123",
				When:    time.Now(),
				Author:  AuthorInfo{Name: "Test", Email: "test@example.com"},
				Message: "Test commit",
			},
			Changes: []FileChange{
				{Path: "file1.go", Kind: ChangeKindModified, LinesAdded: 10, LinesDeleted: 5},
			},
		},
	}

	t.Run("returns change sets", func(t *testing.T) {
		reader := NewMockHistoryReader(expectedChangeSets, nil)

		changeSets, err := reader.ReadChanges()

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if len(changeSets) != len(expectedChangeSets) {
			t.Errorf("expected %d change sets, got %d", len(expectedChangeSets), len(changeSets))
		}
	})

	t.Run("returns error", func(t *testing.T) {
		expectedErr := errors.New("test error")
		reader := NewMockHistoryReader(nil, expectedErr)

		_, err := reader.ReadChanges()

		if err != expectedErr {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
	})
}

func TestMockHistoryReader_ImplementsInterface(t *testing.T) {
	// This test verifies that MockHistoryReader implements RepositoryReader
	var _ RepositoryReader = (*MockHistoryReader)(nil)
}
