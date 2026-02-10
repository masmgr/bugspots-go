package complexity

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCountLines(t *testing.T) {
	tests := []struct {
		name     string
		content  []byte
		expected int
	}{
		{name: "Empty", content: []byte{}, expected: 0},
		{name: "Single line no newline", content: []byte("hello"), expected: 1},
		{name: "Single line with newline", content: []byte("hello\n"), expected: 1},
		{name: "Two lines", content: []byte("hello\nworld\n"), expected: 2},
		{name: "Two lines no trailing", content: []byte("hello\nworld"), expected: 2},
		{name: "Multiple lines", content: []byte("a\nb\nc\nd\n"), expected: 4},
		{name: "Only newlines", content: []byte("\n\n\n"), expected: 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := countLines(tt.content)
			if result != tt.expected {
				t.Errorf("countLines(%q) = %d, expected %d", tt.content, result, tt.expected)
			}
		})
	}
}

// --- Integration tests using temporary git repositories ---

func testRunGit(t testing.TB, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}

func initTestRepo(t testing.TB) string {
	t.Helper()
	dir := t.TempDir()
	testRunGit(t, dir, "init")
	testRunGit(t, dir, "config", "user.name", "Test")
	testRunGit(t, dir, "config", "user.email", "test@test.com")
	return dir
}

func writeTestFile(t testing.TB, dir, name, content string) {
	t.Helper()
	full := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	testRunGit(t, dir, "add", name)
}

func TestFileLineCounts_Basic(t *testing.T) {
	dir := initTestRepo(t)
	writeTestFile(t, dir, "a.go", "line1\nline2\nline3\n")
	writeTestFile(t, dir, "b.go", "one\ntwo\n")
	testRunGit(t, dir, "commit", "-m", "init")

	paths := map[string]struct{}{
		"a.go": {},
		"b.go": {},
	}
	result, err := FileLineCounts(context.Background(), dir, "HEAD", paths)
	if err != nil {
		t.Fatalf("FileLineCounts error: %v", err)
	}

	if got := result["a.go"]; got != 3 {
		t.Errorf("a.go: got %d lines, want 3", got)
	}
	if got := result["b.go"]; got != 2 {
		t.Errorf("b.go: got %d lines, want 2", got)
	}
}

func TestFileLineCounts_BinaryFileSkipped(t *testing.T) {
	dir := initTestRepo(t)
	writeTestFile(t, dir, "text.go", "hello\nworld\n")

	// Write a binary file containing NUL bytes
	binPath := filepath.Join(dir, "binary.dat")
	if err := os.WriteFile(binPath, []byte{0x89, 0x50, 0x00, 0x47, 0x0A}, 0o644); err != nil {
		t.Fatal(err)
	}
	testRunGit(t, dir, "add", "binary.dat")
	testRunGit(t, dir, "commit", "-m", "init")

	paths := map[string]struct{}{
		"text.go":    {},
		"binary.dat": {},
	}
	result, err := FileLineCounts(context.Background(), dir, "HEAD", paths)
	if err != nil {
		t.Fatalf("FileLineCounts error: %v", err)
	}

	if got := result["text.go"]; got != 2 {
		t.Errorf("text.go: got %d lines, want 2", got)
	}
	if _, ok := result["binary.dat"]; ok {
		t.Error("binary.dat should not be in results")
	}
}

func TestFileLineCounts_EmptyPathSet(t *testing.T) {
	dir := initTestRepo(t)
	writeTestFile(t, dir, "a.go", "hello\n")
	testRunGit(t, dir, "commit", "-m", "init")

	result, err := FileLineCounts(context.Background(), dir, "HEAD", map[string]struct{}{})
	if err != nil {
		t.Fatalf("FileLineCounts error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty result, got %v", result)
	}
}

func TestFileLineCounts_SubsetOfFiles(t *testing.T) {
	dir := initTestRepo(t)
	writeTestFile(t, dir, "a.go", "aaa\n")
	writeTestFile(t, dir, "b.go", "bbb\n")
	writeTestFile(t, dir, "c.go", "ccc\n")
	testRunGit(t, dir, "commit", "-m", "init")

	paths := map[string]struct{}{
		"a.go": {},
		"c.go": {},
	}
	result, err := FileLineCounts(context.Background(), dir, "HEAD", paths)
	if err != nil {
		t.Fatalf("FileLineCounts error: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 results, got %d: %v", len(result), result)
	}
	if _, ok := result["b.go"]; ok {
		t.Error("b.go should not be in results")
	}
}

func TestFileLineCounts_EmptyFile(t *testing.T) {
	dir := initTestRepo(t)
	writeTestFile(t, dir, "empty.go", "")
	testRunGit(t, dir, "commit", "-m", "init", "--allow-empty")

	paths := map[string]struct{}{
		"empty.go": {},
	}
	result, err := FileLineCounts(context.Background(), dir, "HEAD", paths)
	if err != nil {
		t.Fatalf("FileLineCounts error: %v", err)
	}

	if got := result["empty.go"]; got != 0 {
		t.Errorf("empty.go: got %d lines, want 0", got)
	}
}

func TestFileLineCounts_NoTrailingNewline(t *testing.T) {
	dir := initTestRepo(t)
	writeTestFile(t, dir, "notail.go", "line1\nline2\nline3")
	testRunGit(t, dir, "commit", "-m", "init")

	paths := map[string]struct{}{
		"notail.go": {},
	}
	result, err := FileLineCounts(context.Background(), dir, "HEAD", paths)
	if err != nil {
		t.Fatalf("FileLineCounts error: %v", err)
	}

	if got := result["notail.go"]; got != 3 {
		t.Errorf("notail.go: got %d lines, want 3", got)
	}
}

func TestFileLineCounts_DefaultRef(t *testing.T) {
	dir := initTestRepo(t)
	writeTestFile(t, dir, "a.go", "hello\nworld\n")
	testRunGit(t, dir, "commit", "-m", "init")

	paths := map[string]struct{}{
		"a.go": {},
	}
	// Pass empty ref, should default to HEAD
	result, err := FileLineCounts(context.Background(), dir, "", paths)
	if err != nil {
		t.Fatalf("FileLineCounts error: %v", err)
	}

	if got := result["a.go"]; got != 2 {
		t.Errorf("a.go: got %d lines, want 2", got)
	}
}
