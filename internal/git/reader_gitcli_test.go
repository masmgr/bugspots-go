package git

import "testing"

func TestParseGitRawAndNumstat_RenameAndModify(t *testing.T) {
	// Body bytes are what comes after the pretty header line.
	// For -z formats, entries are NUL-separated and concatenated.
	body := []byte{}

	// Modify a.txt
	body = append(body, []byte(":100644 100644 1111111 2222222 M")...)
	body = append(body, 0)
	body = append(body, []byte("a.txt")...)
	body = append(body, 0)

	// Rename old.go -> new.go
	body = append(body, []byte(":100644 100644 3333333 4444444 R100")...)
	body = append(body, 0)
	body = append(body, []byte("old.go")...)
	body = append(body, 0)
	body = append(body, []byte("new.go")...)
	body = append(body, 0)

	// Numstat for a.txt
	body = append(body, []byte("1\t2\ta.txt")...)
	body = append(body, 0)

	// Numstat for rename: with -z, git writes an empty path then old\0new\0
	body = append(body, []byte("3\t4\t")...)
	body = append(body, 0) // empty path signals rename
	body = append(body, []byte("old.go")...)
	body = append(body, 0)
	body = append(body, []byte("new.go")...)
	body = append(body, 0)

	raw, pos, err := parseGitRawEntries(body)
	if err != nil {
		t.Fatalf("parseGitRawEntries: %v", err)
	}
	if len(raw) != 2 {
		t.Fatalf("raw entries = %d, expected 2", len(raw))
	}
	if raw[0].status != "M" || raw[0].path != "a.txt" || raw[0].oldPath != "" {
		t.Fatalf("raw[0] = %#v", raw[0])
	}
	if raw[1].status != "R100" || raw[1].path != "new.go" || raw[1].oldPath != "old.go" {
		t.Fatalf("raw[1] = %#v", raw[1])
	}

	stats, err := parseGitNumstat(body[pos:], raw)
	if err != nil {
		t.Fatalf("parseGitNumstat: %v", err)
	}
	if len(stats) != 2 {
		t.Fatalf("stats = %d, expected 2", len(stats))
	}
	if stats[0].added != 1 || stats[0].deleted != 2 {
		t.Fatalf("stats[0] = %#v, expected 1/2", stats[0])
	}
	if stats[1].added != 3 || stats[1].deleted != 4 {
		t.Fatalf("stats[1] = %#v, expected 3/4", stats[1])
	}
}

func TestParseGitNumstat_LeadingNewline(t *testing.T) {
	// Real git output has a newline separating --raw from --numstat sections.
	body := []byte{}

	// --raw entry: modify foo.js
	body = append(body, []byte(":100644 100644 aaa bbb M")...)
	body = append(body, 0)
	body = append(body, []byte("External/foo.js")...)
	body = append(body, 0)

	// Newline separator (as real git produces)
	body = append(body, '\n')

	// --numstat entry
	body = append(body, []byte("5\t3\tExternal/foo.js")...)
	body = append(body, 0)

	raw, pos, err := parseGitRawEntries(body)
	if err != nil {
		t.Fatalf("parseGitRawEntries: %v", err)
	}
	if len(raw) != 1 {
		t.Fatalf("raw entries = %d, expected 1", len(raw))
	}

	stats, err := parseGitNumstat(body[pos:], raw)
	if err != nil {
		t.Fatalf("parseGitNumstat: %v", err)
	}
	if len(stats) != 1 {
		t.Fatalf("stats = %d, expected 1", len(stats))
	}
	if stats[0].added != 5 || stats[0].deleted != 3 {
		t.Fatalf("stats[0] = %#v, expected 5/3", stats[0])
	}
}

func TestKindFromGitStatus(t *testing.T) {
	tests := []struct {
		status   string
		oldPath  string
		wantKind ChangeKind
		wantOld  string
	}{
		{status: "A", wantKind: ChangeKindAdded},
		{status: "M", wantKind: ChangeKindModified},
		{status: "D", wantKind: ChangeKindDeleted},
		{status: "R100", oldPath: "old.go", wantKind: ChangeKindRenamed, wantOld: "old.go"},
	}

	for _, tt := range tests {
		gotKind, gotOld := kindFromGitStatus(tt.status, tt.oldPath)
		if gotKind != tt.wantKind || gotOld != tt.wantOld {
			t.Fatalf("kindFromGitStatus(%q,%q) = (%v,%q), want (%v,%q)", tt.status, tt.oldPath, gotKind, gotOld, tt.wantKind, tt.wantOld)
		}
	}
}
