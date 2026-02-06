package git

import (
	"fmt"
	"strconv"
)

// gitFileMode represents a Git file mode as an octal value.
// This is used when parsing git --raw output.
type gitFileMode uint32

const (
	gitFileModeEmpty   gitFileMode = 0
	gitFileModeRegular gitFileMode = 0100644
	gitFileModeExec    gitFileMode = 0100755
	gitFileModeSymlink gitFileMode = 0120000
)

// IsFile returns true if the mode represents a regular file or symlink.
func (m gitFileMode) IsFile() bool {
	return m == gitFileModeRegular || m == gitFileModeExec || m == gitFileModeSymlink
}

// parseGitFileMode parses an octal file mode string (e.g. "100644", "120000", "000000").
func parseGitFileMode(s string) (gitFileMode, error) {
	if s == "" {
		return gitFileModeEmpty, nil
	}
	v, err := strconv.ParseUint(s, 8, 32)
	if err != nil {
		return gitFileModeEmpty, fmt.Errorf("parse file mode %q: %w", s, err)
	}
	return gitFileMode(v), nil
}
