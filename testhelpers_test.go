package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"testing"
	"time"
)

// createTestRepo creates a temporary git repository with test commits.
// Returns the directory path.
func createTestRepo(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()

	runGit(t, tmpDir, "init")
	runGit(t, tmpDir, "config", "user.name", "Test User")
	runGit(t, tmpDir, "config", "user.email", "test@example.com")

	return tmpDir
}

// addCommitToRepo adds a commit to a test repository.
func addCommitToRepo(t *testing.T, repoDir string, message string, filenames []string, commitTime time.Time) {
	t.Helper()

	// Create/modify files
	for _, filename := range filenames {
		filePath := fmt.Sprintf("%s/%s", repoDir, filename)

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
		runGit(t, repoDir, "add", filename)
	}

	// Create commit with custom time
	env := []string{
		fmt.Sprintf("GIT_AUTHOR_DATE=%s", commitTime.Format(time.RFC3339)),
		fmt.Sprintf("GIT_COMMITTER_DATE=%s", commitTime.Format(time.RFC3339)),
		fmt.Sprintf("GIT_AUTHOR_NAME=%s", "Test Author"),
		fmt.Sprintf("GIT_AUTHOR_EMAIL=%s", "test@example.com"),
		fmt.Sprintf("GIT_COMMITTER_NAME=%s", "Test Author"),
		fmt.Sprintf("GIT_COMMITTER_EMAIL=%s", "test@example.com"),
	}
	cmd := exec.Command("git", "commit", "-m", message)
	cmd.Dir = repoDir
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to commit: %v\n%s", err, out)
	}
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

// runGit runs a git command in the given directory.
func runGit(t testing.TB, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}
