package cmd

import (
	"fmt"
	"time"

	"github.com/masmgr/bugspots-go/config"
	"github.com/masmgr/bugspots-go/internal/git"
	"github.com/masmgr/bugspots-go/internal/output"
	"github.com/urfave/cli/v2"
)

// CommandContext holds common state for command execution.
// It encapsulates the shared setup logic across all analysis commands.
type CommandContext struct {
	Config     *config.Config
	RepoPath   string
	Since      *time.Time
	Until      time.Time
	Branch     string
	ChangeSets []git.CommitChangeSet
}

// NewCommandContext creates a context from CLI flags.
// It performs configuration loading, date parsing, repository opening, and history reading.
func NewCommandContext(c *cli.Context) (*CommandContext, error) {
	// Load configuration
	cfg, err := loadConfig(c)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Parse date flags
	since, err := parseDateFlag(c.String("since"))
	if err != nil {
		return nil, fmt.Errorf("invalid since date: %w", err)
	}
	until, err := parseDateFlag(c.String("until"))
	if err != nil {
		return nil, fmt.Errorf("invalid until date: %w", err)
	}

	untilTime := time.Now()
	if until != nil {
		untilTime = *until
	}

	// Set up Git reader
	repoPath := c.String("repo")
	branch := c.String("branch")

	reader, err := git.NewHistoryReader(git.ReadOptions{
		RepoPath: repoPath,
		Branch:   branch,
		Since:    since,
		Until:    until,
		Include:  cfg.Filters.Include,
		Exclude:  cfg.Filters.Exclude,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	// Read commit changes
	changeSets, err := reader.ReadChanges()
	if err != nil {
		return nil, fmt.Errorf("failed to read history: %w", err)
	}

	return &CommandContext{
		Config:     cfg,
		RepoPath:   repoPath,
		Since:      since,
		Until:      untilTime,
		Branch:     branch,
		ChangeSets: changeSets,
	}, nil
}

// HasCommits returns true if commits were found in the specified range.
func (ctx *CommandContext) HasCommits() bool {
	return len(ctx.ChangeSets) > 0
}

// PrintNoCommitsMessage prints a message when no commits are found.
func (ctx *CommandContext) PrintNoCommitsMessage() {
	fmt.Println("No commits found in the specified range.")
}

// OutputOptions creates OutputOptions from CLI flags.
func OutputOptions(c *cli.Context) output.OutputOptions {
	return output.OutputOptions{
		Format:     getOutputFormat(c.String("format")),
		Top:        c.Int("top"),
		OutputPath: c.String("output"),
		Explain:    c.Bool("explain"),
	}
}
