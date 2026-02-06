package git

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestHistoryReader_ReadChanges_RespectsBranch(t *testing.T) {
	repoDir := t.TempDir()

	testRunGit(t, repoDir, "init")
	testRunGit(t, repoDir, "config", "user.name", "Test")
	testRunGit(t, repoDir, "config", "user.email", "test@example.com")

	write := func(rel, content string) {
		t.Helper()
		full := filepath.Join(repoDir, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		testRunGit(t, repoDir, "add", rel)
	}

	now := time.Now()

	commit := func(msg string, when time.Time) {
		t.Helper()
		testRunGitWithEnv(t, repoDir, []string{
			fmt.Sprintf("GIT_AUTHOR_DATE=%s", when.Format(time.RFC3339)),
			fmt.Sprintf("GIT_COMMITTER_DATE=%s", when.Format(time.RFC3339)),
		}, "commit", "-m", msg)
	}

	// Initial commit on base branch (will be skipped by HistoryReader because it has no parents).
	write("file.txt", "initial\n")
	commit("initial", now.Add(-3*time.Hour))

	baseBranchShort := testGitOutput(t, repoDir, "rev-parse", "--abbrev-ref", "HEAD")

	// Create feature branch and commit on it.
	testRunGit(t, repoDir, "checkout", "-b", "feature")
	write("file.txt", "feature\n")
	commit("feature commit", now.Add(-2*time.Hour))

	// Back to base branch and commit there (should not show up when analyzing "feature").
	testRunGit(t, repoDir, "checkout", baseBranchShort)
	write("master.txt", "base\n")
	commit("base commit", now.Add(-1*time.Hour))

	featureReader, err := NewHistoryReader(ReadOptions{
		RepoPath: repoDir,
		Branch:   "feature",
	})
	if err != nil {
		t.Fatalf("NewHistoryReader(feature): %v", err)
	}

	featureChanges, err := featureReader.ReadChanges(context.Background())
	if err != nil {
		t.Fatalf("ReadChanges(feature): %v", err)
	}
	if len(featureChanges) != 1 {
		t.Fatalf("feature changesets = %d, expected 1", len(featureChanges))
	}
	if featureChanges[0].Commit.Message != "feature commit" {
		t.Fatalf("feature head message = %q, expected %q", featureChanges[0].Commit.Message, "feature commit")
	}

	baseReader, err := NewHistoryReader(ReadOptions{
		RepoPath: repoDir,
		Branch:   baseBranchShort,
	})
	if err != nil {
		t.Fatalf("NewHistoryReader(%s): %v", baseBranchShort, err)
	}

	baseChanges, err := baseReader.ReadChanges(context.Background())
	if err != nil {
		t.Fatalf("ReadChanges(%s): %v", baseBranchShort, err)
	}
	if len(baseChanges) != 1 {
		t.Fatalf("base changesets = %d, expected 1", len(baseChanges))
	}
	if baseChanges[0].Commit.Message != "base commit" {
		t.Fatalf("base head message = %q, expected %q", baseChanges[0].Commit.Message, "base commit")
	}
}

// testRunGit runs a git command in the given directory (test helper).
func testRunGit(t testing.TB, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}

// testRunGitWithEnv runs a git command with extra environment variables.
func testRunGitWithEnv(t testing.TB, dir string, env []string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}

// testGitOutput runs a git command and returns its trimmed stdout.
func testGitOutput(t testing.TB, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %v failed: %v", args, err)
	}
	return strings.TrimSpace(string(out))
}
