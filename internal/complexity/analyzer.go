package complexity

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// lsTreeEntry holds the blob hash and path from git ls-tree output.
type lsTreeEntry struct {
	BlobHash string
	Path     string
}

// FileLineCounts returns the line count for each file in the repository at the given ref.
// Only files whose paths are in the provided set are counted.
// Binary files (containing NUL bytes) are skipped and will not appear in the result.
func FileLineCounts(ctx context.Context, repoPath, ref string, paths map[string]struct{}) (map[string]int, error) {
	if len(paths) == 0 {
		return map[string]int{}, nil
	}

	if ref == "" {
		ref = "HEAD"
	}

	// Step 1: Run git ls-tree to get blob hashes for all files
	entries, err := listTreeEntries(ctx, repoPath, ref, paths)
	if err != nil {
		return nil, err
	}

	if len(entries) == 0 {
		return map[string]int{}, nil
	}

	// Step 2: Use git cat-file --batch to read blob contents and count lines
	return countLinesBatch(ctx, repoPath, entries)
}

// listTreeEntries runs git ls-tree and returns entries for paths in the wanted set.
func listTreeEntries(ctx context.Context, repoPath, ref string, wanted map[string]struct{}) ([]lsTreeEntry, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "ls-tree", "-r", ref)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-tree: %w", err)
	}

	var entries []lsTreeEntry
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		// Format: <mode> <type> <hash>\t<path>
		tabIdx := strings.IndexByte(line, '\t')
		if tabIdx < 0 {
			continue
		}
		path := line[tabIdx+1:]
		if _, ok := wanted[path]; !ok {
			continue
		}

		fields := strings.Fields(line[:tabIdx])
		if len(fields) < 3 {
			continue
		}
		objType := fields[1]
		if objType != "blob" {
			continue
		}

		entries = append(entries, lsTreeEntry{
			BlobHash: fields[2],
			Path:     path,
		})
	}

	return entries, scanner.Err()
}

// countLinesBatch uses git cat-file --batch to read blobs and count lines.
func countLinesBatch(ctx context.Context, repoPath string, entries []lsTreeEntry) (map[string]int, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "cat-file", "--batch")

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("cat-file stdin: %w", err)
	}

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("cat-file start: %w", err)
	}

	// Write all blob hashes to stdin
	for _, e := range entries {
		if _, err := fmt.Fprintf(stdin, "%s\n", e.BlobHash); err != nil {
			stdin.Close()
			_ = cmd.Wait()
			return nil, fmt.Errorf("cat-file write: %w", err)
		}
	}
	stdin.Close()

	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("cat-file: %w", err)
	}

	// Parse output: each blob response is:
	//   <hash> blob <size>\n
	//   <content>\n
	result := make(map[string]int, len(entries))
	reader := bufio.NewReader(&stdout)
	for _, entry := range entries {
		// Read header line: "<hash> blob <size>"
		headerLine, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		headerLine = strings.TrimRight(headerLine, "\n\r")
		parts := strings.Fields(headerLine)
		if len(parts) < 3 || parts[1] == "missing" {
			continue
		}

		size, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			continue
		}

		// Read exactly <size> bytes of content
		content := make([]byte, size)
		n, err := readFull(reader, content)
		if err != nil && n < int(size) {
			break
		}
		content = content[:n]

		// Read trailing newline after content
		_, _ = reader.ReadByte()

		// Skip binary files (contain NUL byte)
		if bytes.ContainsRune(content, 0) {
			continue
		}

		// Count lines
		lineCount := countLines(content)
		result[entry.Path] = lineCount
	}

	return result, nil
}

// readFull reads exactly len(buf) bytes from reader.
func readFull(reader *bufio.Reader, buf []byte) (int, error) {
	total := 0
	for total < len(buf) {
		n, err := reader.Read(buf[total:])
		total += n
		if err != nil {
			return total, err
		}
	}
	return total, nil
}

// countLines counts the number of lines in content.
// An empty file has 0 lines. A file with no trailing newline still counts its last line.
func countLines(content []byte) int {
	if len(content) == 0 {
		return 0
	}
	count := bytes.Count(content, []byte{'\n'})
	// If the last byte is not a newline, there's one more line
	if content[len(content)-1] != '\n' {
		count++
	}
	return count
}
