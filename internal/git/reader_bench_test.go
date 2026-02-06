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

func createBenchRepo(tb testing.TB, commits, files, vendorLines int) string {
	tb.Helper()

	repoDir := tb.TempDir()

	testRunGit(tb, repoDir, "init")
	testRunGit(tb, repoDir, "config", "user.name", "Bench")
	testRunGit(tb, repoDir, "config", "user.email", "bench@example.com")

	writeAndAdd := func(rel, content string) {
		tb.Helper()
		full := filepath.Join(repoDir, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			tb.Fatalf("MkdirAll: %v", err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			tb.Fatalf("WriteFile: %v", err)
		}
		testRunGit(tb, repoDir, "add", rel)
	}

	commitWithTime := func(msg string, when time.Time) {
		tb.Helper()
		cmd := exec.Command("git", "commit", "-m", msg)
		cmd.Dir = repoDir
		cmd.Env = append(os.Environ(),
			fmt.Sprintf("GIT_AUTHOR_DATE=%s", when.Format(time.RFC3339)),
			fmt.Sprintf("GIT_COMMITTER_DATE=%s", when.Format(time.RFC3339)),
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			tb.Fatalf("git commit failed: %v\n%s", err, out)
		}
	}

	base := time.Now().Add(-time.Duration(commits+10) * time.Hour)

	// Initial commit (HistoryReader skips it because it has no parents).
	writeAndAdd("src/file000.txt", "initial\n")
	if vendorLines > 0 {
		writeAndAdd("vendor/big.txt", "initial\n")
	}
	commitWithTime("initial", base)

	for i := 0; i < commits; i++ {
		when := base.Add(time.Duration(i+1) * time.Hour)

		for f := 0; f < files; f++ {
			rel := fmt.Sprintf("src/file%03d.txt", f)
			// Keep diffs small-ish but non-empty.
			content := fmt.Sprintf("commit=%d file=%d\nline\n", i, f)
			writeAndAdd(rel, content)
		}

		if vendorLines > 0 {
			var sb strings.Builder
			sb.Grow(vendorLines * 16)
			for l := 0; l < vendorLines; l++ {
				sb.WriteString("x")
				sb.WriteString(fmt.Sprintf("%d", i))
				sb.WriteByte('\n')
			}
			writeAndAdd("vendor/big.txt", sb.String())
		}

		commitWithTime(fmt.Sprintf("commit %d", i), when)
	}

	return repoDir
}

func BenchmarkHistoryReader_ReadChanges_Full(b *testing.B) {
	repoDir := createBenchRepo(b, 80, 25, 0)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		reader, err := NewHistoryReader(ReadOptions{
			RepoPath:     repoDir,
			DetailLevel:  ChangeDetailFull,
			RenameDetect: RenameDetectAggressive,
		})
		if err != nil {
			b.Fatalf("NewHistoryReader: %v", err)
		}
		changeSets, err := reader.ReadChanges(context.Background())
		if err != nil {
			b.Fatalf("ReadChanges: %v", err)
		}
		if len(changeSets) == 0 {
			b.Fatalf("unexpected empty changesets")
		}
	}
}

func BenchmarkHistoryReader_ReadChanges_PathsOnly(b *testing.B) {
	repoDir := createBenchRepo(b, 80, 25, 0)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		reader, err := NewHistoryReader(ReadOptions{
			RepoPath:     repoDir,
			DetailLevel:  ChangeDetailPathsOnly,
			RenameDetect: RenameDetectSimple,
		})
		if err != nil {
			b.Fatalf("NewHistoryReader: %v", err)
		}
		changeSets, err := reader.ReadChanges(context.Background())
		if err != nil {
			b.Fatalf("ReadChanges: %v", err)
		}
		if len(changeSets) == 0 {
			b.Fatalf("unexpected empty changesets")
		}
	}
}

func BenchmarkHistoryReader_ReadChanges_PathsOnly_SimpleRename(b *testing.B) {
	repoDir := createBenchRepo(b, 80, 25, 0)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		reader, err := NewHistoryReader(ReadOptions{
			RepoPath:     repoDir,
			DetailLevel:  ChangeDetailPathsOnly,
			RenameDetect: RenameDetectSimple,
		})
		if err != nil {
			b.Fatalf("NewHistoryReader: %v", err)
		}
		changeSets, err := reader.ReadChanges(context.Background())
		if err != nil {
			b.Fatalf("ReadChanges: %v", err)
		}
		if len(changeSets) == 0 {
			b.Fatalf("unexpected empty changesets")
		}
	}
}

func BenchmarkHistoryReader_ReadChanges_Full_ExcludeLargePath(b *testing.B) {
	repoDir := createBenchRepo(b, 80, 5, 4000)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		reader, err := NewHistoryReader(ReadOptions{
			RepoPath:     repoDir,
			DetailLevel:  ChangeDetailFull,
			RenameDetect: RenameDetectAggressive,
			Exclude:      []string{"vendor/**"},
		})
		if err != nil {
			b.Fatalf("NewHistoryReader: %v", err)
		}
		changeSets, err := reader.ReadChanges(context.Background())
		if err != nil {
			b.Fatalf("ReadChanges: %v", err)
		}
		if len(changeSets) == 0 {
			b.Fatalf("unexpected empty changesets")
		}
	}
}

func BenchmarkHistoryReader_ReadChanges_Full_IncludeLargePath(b *testing.B) {
	repoDir := createBenchRepo(b, 80, 5, 4000)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		reader, err := NewHistoryReader(ReadOptions{
			RepoPath:     repoDir,
			DetailLevel:  ChangeDetailFull,
			RenameDetect: RenameDetectAggressive,
		})
		if err != nil {
			b.Fatalf("NewHistoryReader: %v", err)
		}
		changeSets, err := reader.ReadChanges(context.Background())
		if err != nil {
			b.Fatalf("ReadChanges: %v", err)
		}
		if len(changeSets) == 0 {
			b.Fatalf("unexpected empty changesets")
		}
	}
}

// BenchmarkHistoryReader_ReadChanges_TimeWindow measures the early termination
// optimization: 200 total commits but only the last 40 are within the Since window.
func BenchmarkHistoryReader_ReadChanges_TimeWindow(b *testing.B) {
	const totalCommits = 200
	repoDir := createBenchRepo(b, totalCommits, 5, 0)

	// Only include commits from the last 40 hours (out of totalCommits+10 hours).
	since := time.Now().Add(-40 * time.Hour)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		reader, err := NewHistoryReader(ReadOptions{
			RepoPath:     repoDir,
			DetailLevel:  ChangeDetailPathsOnly,
			RenameDetect: RenameDetectSimple,
			Since:        &since,
		})
		if err != nil {
			b.Fatalf("NewHistoryReader: %v", err)
		}
		changeSets, err := reader.ReadChanges(context.Background())
		if err != nil {
			b.Fatalf("ReadChanges: %v", err)
		}
		if len(changeSets) == 0 {
			b.Fatalf("unexpected empty changesets")
		}
	}
}
