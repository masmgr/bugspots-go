package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func TestHistoryReader_ReadChanges_RespectsBranch(t *testing.T) {
	repoDir := t.TempDir()

	repo, err := gogit.PlainInit(repoDir, false)
	if err != nil {
		t.Fatalf("PlainInit: %v", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Worktree: %v", err)
	}

	write := func(rel, content string) {
		t.Helper()
		full := filepath.Join(repoDir, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		if _, err := wt.Add(rel); err != nil {
			t.Fatalf("Add: %v", err)
		}
	}

	commit := func(msg string, when time.Time) {
		t.Helper()
		_, err := wt.Commit(msg, &gogit.CommitOptions{
			Author: &object.Signature{
				Name:  "Test",
				Email: "test@example.com",
				When:  when,
			},
			Committer: &object.Signature{
				Name:  "Test",
				Email: "test@example.com",
				When:  when,
			},
		})
		if err != nil {
			t.Fatalf("Commit: %v", err)
		}
	}

	now := time.Now()

	// Initial commit on base branch (will be skipped by HistoryReader because it has no parents).
	write("file.txt", "initial\n")
	commit("initial", now.Add(-3*time.Hour))

	head, err := repo.Head()
	if err != nil {
		t.Fatalf("Head: %v", err)
	}
	baseBranch := head.Name()
	baseBranchShort := baseBranch.Short()

	// Create feature branch and commit on it.
	if err := wt.Checkout(&gogit.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName("feature"),
		Create: true,
	}); err != nil {
		t.Fatalf("Checkout(feature): %v", err)
	}
	write("file.txt", "feature\n")
	commit("feature commit", now.Add(-2*time.Hour))

	// Back to base branch and commit there (should not show up when analyzing "feature").
	if err := wt.Checkout(&gogit.CheckoutOptions{Branch: baseBranch}); err != nil {
		t.Fatalf("Checkout(%s): %v", baseBranch, err)
	}
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
