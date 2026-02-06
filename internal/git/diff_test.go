package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestParseDiffSpec_ThreeDot(t *testing.T) {
	base, head, err := ParseDiffSpec("origin/main...HEAD")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if base != "origin/main" {
		t.Errorf("base = %q, want %q", base, "origin/main")
	}
	if head != "HEAD" {
		t.Errorf("head = %q, want %q", head, "HEAD")
	}
}

func TestParseDiffSpec_TwoDot(t *testing.T) {
	base, head, err := ParseDiffSpec("abc123..def456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if base != "abc123" {
		t.Errorf("base = %q, want %q", base, "abc123")
	}
	if head != "def456" {
		t.Errorf("head = %q, want %q", head, "def456")
	}
}

func TestParseDiffSpec_EmptyHead(t *testing.T) {
	base, head, err := ParseDiffSpec("origin/main...")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if base != "origin/main" {
		t.Errorf("base = %q, want %q", base, "origin/main")
	}
	if head != "HEAD" {
		t.Errorf("head = %q, want %q", head, "HEAD")
	}
}

func TestParseDiffSpec_EmptyBase(t *testing.T) {
	_, _, err := ParseDiffSpec("...HEAD")
	if err == nil {
		t.Fatal("expected error for empty base")
	}
}

func TestParseDiffSpec_NoDots(t *testing.T) {
	_, _, err := ParseDiffSpec("origin/main")
	if err == nil {
		t.Fatal("expected error for missing '..' or '...'")
	}
}

func TestParseDiffSpec_Empty(t *testing.T) {
	_, _, err := ParseDiffSpec("")
	if err == nil {
		t.Fatal("expected error for empty spec")
	}
}

func TestParseDiffNameStatus(t *testing.T) {
	// Simulate: M\0file1.go\0A\0file2.go\0D\0file3.go\0
	data := []byte("M\x00file1.go\x00A\x00file2.go\x00D\x00file3.go\x00")

	entries, err := parseDiffNameStatus(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	tests := []struct {
		path string
		kind ChangeKind
	}{
		{"file1.go", ChangeKindModified},
		{"file2.go", ChangeKindAdded},
		{"file3.go", ChangeKindDeleted},
	}

	for i, tt := range tests {
		if entries[i].Path != tt.path {
			t.Errorf("entry[%d].Path = %q, want %q", i, entries[i].Path, tt.path)
		}
		if entries[i].ChangeKind != tt.kind {
			t.Errorf("entry[%d].ChangeKind = %v, want %v", i, entries[i].ChangeKind, tt.kind)
		}
	}
}

func TestParseDiffNameStatus_Rename(t *testing.T) {
	// Simulate: R100\0old.go\0new.go\0
	data := []byte("R100\x00old.go\x00new.go\x00")

	entries, err := parseDiffNameStatus(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	if entries[0].Path != "new.go" {
		t.Errorf("Path = %q, want %q", entries[0].Path, "new.go")
	}
	if entries[0].OldPath != "old.go" {
		t.Errorf("OldPath = %q, want %q", entries[0].OldPath, "old.go")
	}
	if entries[0].ChangeKind != ChangeKindRenamed {
		t.Errorf("ChangeKind = %v, want %v", entries[0].ChangeKind, ChangeKindRenamed)
	}
}

func TestParseDiffNameStatus_Empty(t *testing.T) {
	entries, err := parseDiffNameStatus([]byte{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestReadDiff_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Create a temporary git repo
	dir := t.TempDir()

	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Test",
			"GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=Test",
			"GIT_COMMITTER_EMAIL=test@test.com",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v failed: %v: %s", args, err, string(out))
		}
	}

	writeFile := func(name, content string) {
		t.Helper()
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// Initialize repo with initial commit
	runGit("init", "-b", "main")
	writeFile("base.go", "package main\n")
	runGit("add", ".")
	runGit("commit", "-m", "initial commit")

	// Create a branch and make changes
	runGit("checkout", "-b", "feature")
	writeFile("added.go", "package added\n")
	writeFile("base.go", "package main\n// modified\n")
	runGit("add", ".")
	runGit("commit", "-m", "feature changes")

	// Test diff between main and feature
	result, err := ReadDiff(context.Background(), DiffOptions{
		RepoPath: dir,
		DiffSpec: "main...feature",
	})
	if err != nil {
		t.Fatalf("ReadDiff failed: %v", err)
	}

	if result.Base != "main" {
		t.Errorf("Base = %q, want %q", result.Base, "main")
	}
	if result.Head != "feature" {
		t.Errorf("Head = %q, want %q", result.Head, "feature")
	}

	if len(result.ChangedFiles) != 2 {
		t.Fatalf("expected 2 changed files, got %d: %+v", len(result.ChangedFiles), result.ChangedFiles)
	}

	pathSet := make(map[string]ChangeKind)
	for _, f := range result.ChangedFiles {
		pathSet[f.Path] = f.ChangeKind
	}

	if kind, ok := pathSet["added.go"]; !ok {
		t.Error("expected added.go in changed files")
	} else if kind != ChangeKindAdded {
		t.Errorf("added.go kind = %v, want %v", kind, ChangeKindAdded)
	}

	if kind, ok := pathSet["base.go"]; !ok {
		t.Error("expected base.go in changed files")
	} else if kind != ChangeKindModified {
		t.Errorf("base.go kind = %v, want %v", kind, ChangeKindModified)
	}
}
