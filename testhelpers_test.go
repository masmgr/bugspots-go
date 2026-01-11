package main

import (
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// createTestRepo creates a temporary git repository with test commits
func createTestRepo(t *testing.T) (string, *git.Repository) {
	// Create temporary directory
	tmpDir := t.TempDir()

	// Initialize bare repository
	repo, err := git.PlainInit(tmpDir, false)
	if err != nil {
		t.Fatalf("Failed to initialize git repo: %v", err)
	}

	return tmpDir, repo
}

// addCommitToRepo adds a commit to a test repository
func addCommitToRepo(t *testing.T, repo *git.Repository, message string, filenames []string, commitTime time.Time) {
	w, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Failed to get worktree: %v", err)
	}

	// Create/modify files
	for _, filename := range filenames {
		filePath := fmt.Sprintf("%s/%s", w.Filesystem.Root(), filename)

		// Create directory if needed
		dir := filePath[:len(filePath)-len(filename)]
		os.MkdirAll(dir, 0755)

		// Write file content with timestamp to ensure different content
		content := fmt.Sprintf("Content for %s at %s\n", filename, commitTime.String())
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}
	}

	// Add files to staging area
	for _, filename := range filenames {
		_, err := w.Add(filename)
		if err != nil {
			t.Fatalf("Failed to add file: %v", err)
		}
	}

	// Create commit with custom time
	hash, err := w.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test Author",
			Email: "test@example.com",
			When:  commitTime,
		},
	})
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	_ = hash
}

// suppressOutput temporarily suppresses stdout for testing
func suppressOutput(t *testing.T, fn func()) {
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = oldStdout
}

// discardOutput is used with os.Pipe to discard output
func discardOutput(t *testing.T, fn func()) {
	// Redirect stdout to discard buffer
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}

	oldStdout := os.Stdout
	os.Stdout = w

	go io.ReadAll(r) // Discard the output

	fn()

	w.Close()
	os.Stdout = oldStdout
}

// configureGitUser sets up git user config for test repo
func configureGitUser(t *testing.T, repo *git.Repository) {
	cfg, err := repo.Config()
	if err != nil {
		t.Fatalf("Failed to get config: %v", err)
	}

	cfg.User.Name = "Test User"
	cfg.User.Email = "test@example.com"
}
