package bugfix

import (
	"regexp"
	"strings"

	"github.com/masmgr/bugspots-go/internal/git"
)

// BugfixResult holds the result of bugfix detection for a set of commits.
type BugfixResult struct {
	// BugfixCommits is the set of commit SHAs identified as bugfix commits.
	BugfixCommits map[string]struct{}
	// FileBugfixCounts maps file paths to the number of bugfix commits that touched them.
	FileBugfixCounts map[string]int
	// TotalBugfixes is the total number of bugfix commits detected.
	TotalBugfixes int
}

// Detector detects bugfix commits by matching commit messages against regex patterns.
type Detector struct {
	patterns []*regexp.Regexp
}

// NewDetector creates a new Detector from a list of regex pattern strings.
// Patterns are compiled as case-insensitive. Returns an error if any pattern fails to compile.
func NewDetector(patterns []string) (*Detector, error) {
	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		// Add case-insensitive flag if not already present
		if !strings.HasPrefix(p, "(?i)") {
			p = "(?i)" + p
		}
		re, err := regexp.Compile(p)
		if err != nil {
			return nil, err
		}
		compiled = append(compiled, re)
	}
	return &Detector{patterns: compiled}, nil
}

// IsBugfix returns true if the given commit message matches any of the detector's patterns.
func (d *Detector) IsBugfix(message string) bool {
	for _, re := range d.patterns {
		if re.MatchString(message) {
			return true
		}
	}
	return false
}

// Detect scans the given change sets and returns the bugfix detection result.
// A commit is classified as a bugfix if its message matches any of the configured patterns.
func (d *Detector) Detect(changeSets []git.CommitChangeSet) *BugfixResult {
	result := &BugfixResult{
		BugfixCommits:    make(map[string]struct{}),
		FileBugfixCounts: make(map[string]int),
	}

	if len(d.patterns) == 0 {
		return result
	}

	for _, cs := range changeSets {
		if !d.IsBugfix(cs.Commit.Message) {
			continue
		}

		result.BugfixCommits[cs.Commit.SHA] = struct{}{}
		result.TotalBugfixes++

		for _, change := range cs.Changes {
			if change.Kind == git.ChangeKindDeleted {
				continue
			}
			result.FileBugfixCounts[change.Path]++
		}
	}

	return result
}
