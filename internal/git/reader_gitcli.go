package git

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/go-git/go-git/v5/plumbing/filemode"
)

type gitRawEntry struct {
	srcMode filemode.FileMode
	dstMode filemode.FileMode
	status  string // e.g. "M", "A", "D", "R100"
	path    string // destination path (or path for non-renames)
	oldPath string // source path for renames
}

type gitNumstat struct {
	added   int
	deleted int
}

func (r *HistoryReader) readChangesGitCLI(ctx context.Context) ([]CommitChangeSet, error) {
	// Each commit header line is prefixed by 0x1e (record separator), then NUL-separated fields,
	// and ends with a newline. This makes the combined --raw/-z and --numstat/-z output
	// reliably parseable as "records" split by 0x1e.
	const format = "%x1e%H%x00%P%x00%cI%x00%an%x00%ae%x00%s%n"

	args := []string{
		"-C", r.opts.RepoPath,
		"log",
		"--no-color",
		"--no-merges",
		"--pretty=format:" + format,
		"--raw", "-z",
	}

	if r.opts.DetailLevel == ChangeDetailFull {
		args = append(args, "--numstat", "-z")
	}

	switch r.opts.RenameDetect {
	case RenameDetectOff:
		args = append(args, "--no-renames")
	case RenameDetectSimple:
		args = append(args, "-M100%")
	case RenameDetectAggressive:
		// Match go-git's default threshold (60).
		args = append(args, "-M60%")
	}

	if r.opts.Since != nil {
		args = append(args, fmt.Sprintf("--since=@%d", r.opts.Since.Unix()))
	}
	if r.opts.Until != nil {
		args = append(args, fmt.Sprintf("--until=@%d", r.opts.Until.Unix()))
	}

	rev := strings.TrimSpace(r.opts.Branch)
	if rev != "" && !strings.EqualFold(rev, "HEAD") {
		args = append(args, rev)
	}

	out, err := exec.CommandContext(ctx, "git", args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git log failed: %w: %s", err, strings.TrimSpace(string(out)))
	}

	records := bytes.Split(out, []byte{0x1e})
	results := make([]CommitChangeSet, 0, 1000)
	processed := 0

	for _, rec := range records {
		if len(rec) == 0 {
			continue
		}

		header, body := splitHeaderBody(rec)
		if len(header) == 0 {
			continue
		}

		fields := bytes.SplitN(header, []byte{0x00}, 6)
		if len(fields) < 6 {
			return nil, fmt.Errorf("unexpected git log header format")
		}

		sha := string(fields[0])
		parents := strings.TrimSpace(string(fields[1]))
		// Skip commits without parents (initial commit), consistent with go-git reader.
		if parents == "" {
			continue
		}

		when, err := time.Parse(time.RFC3339, string(fields[2]))
		if err != nil {
			return nil, fmt.Errorf("parse committer date: %w", err)
		}

		authorName := string(fields[3])
		authorEmail := string(fields[4])
		subject := string(fields[5])

		rawEntries, pos, err := parseGitRawEntries(body)
		if err != nil {
			return nil, err
		}

		var stats []gitNumstat
		if r.opts.DetailLevel == ChangeDetailFull {
			stats, err = parseGitNumstat(body[pos:], rawEntries)
			if err != nil {
				return nil, err
			}
		} else {
			stats = make([]gitNumstat, len(rawEntries))
		}

		changes := make([]FileChange, 0, len(rawEntries))
		for i, e := range rawEntries {
			if !e.srcMode.IsFile() && !e.dstMode.IsFile() {
				continue
			}

			path := e.path
			if path == "" {
				continue
			}

			matches, err := r.matchesFilters(path)
			if err != nil {
				return nil, err
			}
			if !matches {
				continue
			}

			kind, oldPath := kindFromGitStatus(e.status, e.oldPath)
			st := stats[i]

			changes = append(changes, FileChange{
				Path:         path,
				OldPath:      oldPath,
				LinesAdded:   st.added,
				LinesDeleted: st.deleted,
				Kind:         kind,
			})
		}

		if len(changes) == 0 {
			continue
		}

		results = append(results, CommitChangeSet{
			Commit: CommitInfo{
				SHA:     sha,
				When:    when,
				Author:  AuthorInfo{Name: authorName, Email: authorEmail},
				Message: subject,
			},
			Changes: changes,
		})

		processed++
		if r.opts.OnProgress != nil {
			r.opts.OnProgress(processed)
		}
	}

	return results, nil
}

func splitHeaderBody(rec []byte) (header []byte, body []byte) {
	// The pretty line is followed by '\n', then diff output.
	if idx := bytes.IndexByte(rec, '\n'); idx != -1 {
		return rec[:idx], rec[idx+1:]
	}
	return rec, nil
}

func parseGitRawEntries(body []byte) ([]gitRawEntry, int, error) {
	i := 0
	for i < len(body) && (body[i] == '\n' || body[i] == '\r') {
		i++
	}

	entries := make([]gitRawEntry, 0, 128)

	for i < len(body) && body[i] == ':' {
		meta, ok := readUntilNUL(body, &i)
		if !ok {
			return nil, 0, fmt.Errorf("unexpected git --raw format (missing NUL)")
		}

		fields := strings.Fields(string(meta))
		if len(fields) < 5 {
			return nil, 0, fmt.Errorf("unexpected git --raw meta: %q", string(meta))
		}

		srcMode, err := parseGitFileMode(strings.TrimPrefix(fields[0], ":"))
		if err != nil {
			return nil, 0, err
		}
		dstMode, err := parseGitFileMode(fields[1])
		if err != nil {
			return nil, 0, err
		}

		status := fields[len(fields)-1]

		path1, ok := readStringUntilNUL(body, &i)
		if !ok {
			return nil, 0, fmt.Errorf("unexpected git --raw format (missing path)")
		}

		path := path1
		oldPath := ""
		if len(status) > 0 && (status[0] == 'R' || status[0] == 'C') {
			path2, ok := readStringUntilNUL(body, &i)
			if !ok {
				return nil, 0, fmt.Errorf("unexpected git --raw format (missing rename path)")
			}
			oldPath = path1
			path = path2
		}

		entries = append(entries, gitRawEntry{
			srcMode: srcMode,
			dstMode: dstMode,
			status:  status,
			path:    path,
			oldPath: oldPath,
		})
	}

	return entries, i, nil
}

func parseGitNumstat(body []byte, rawEntries []gitRawEntry) ([]gitNumstat, error) {
	stats := make([]gitNumstat, 0, len(rawEntries))
	i := 0
	for idx := range rawEntries {
		added, ok, err := readNumstatInt(body, &i, '\t')
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, fmt.Errorf("unexpected git --numstat format (added)")
		}

		deleted, ok, err := readNumstatInt(body, &i, '\t')
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, fmt.Errorf("unexpected git --numstat format (deleted)")
		}

		// Consume file name(s) (we use paths from --raw as the source of truth).
		if _, ok := readStringUntilNUL(body, &i); !ok {
			return nil, fmt.Errorf("unexpected git --numstat format (path)")
		}
		if s := rawEntries[idx].status; len(s) > 0 && (s[0] == 'R' || s[0] == 'C') {
			if _, ok := readStringUntilNUL(body, &i); !ok {
				return nil, fmt.Errorf("unexpected git --numstat format (rename path)")
			}
		}

		stats = append(stats, gitNumstat{added: added, deleted: deleted})
	}

	return stats, nil
}

func parseGitFileMode(s string) (filemode.FileMode, error) {
	if s == "" {
		return filemode.Empty, nil
	}
	// Modes are printed as octal (e.g. 100644, 120000, 160000, 000000).
	v, err := strconv.ParseUint(s, 8, 32)
	if err != nil {
		return filemode.Empty, fmt.Errorf("parse file mode %q: %w", s, err)
	}
	return filemode.FileMode(v), nil
}

func kindFromGitStatus(status, oldPath string) (ChangeKind, string) {
	if status == "" {
		return ChangeKindModified, ""
	}
	switch status[0] {
	case 'A':
		return ChangeKindAdded, ""
	case 'D':
		return ChangeKindDeleted, ""
	case 'R':
		return ChangeKindRenamed, oldPath
	default:
		return ChangeKindModified, ""
	}
}

func readUntilNUL(b []byte, i *int) ([]byte, bool) {
	if *i >= len(b) {
		return nil, false
	}
	j := bytes.IndexByte(b[*i:], 0)
	if j == -1 {
		return nil, false
	}
	start := *i
	end := *i + j
	*i = end + 1
	return b[start:end], true
}

func readStringUntilNUL(b []byte, i *int) (string, bool) {
	raw, ok := readUntilNUL(b, i)
	if !ok {
		return "", false
	}
	return string(raw), true
}

func readNumstatInt(b []byte, i *int, delim byte) (int, bool, error) {
	if *i >= len(b) {
		return 0, false, nil
	}
	j := bytes.IndexByte(b[*i:], delim)
	if j == -1 {
		return 0, false, nil
	}
	field := b[*i : *i+j]
	*i = *i + j + 1

	if len(field) == 1 && field[0] == '-' {
		return 0, true, nil
	}
	n, err := strconv.Atoi(string(field))
	if err != nil {
		return 0, true, fmt.Errorf("parse numstat int %q: %w", string(field), err)
	}
	return n, true, nil
}
