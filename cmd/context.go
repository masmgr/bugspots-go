package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/masmgr/bugspots-go/config"
	"github.com/masmgr/bugspots-go/internal/git"
	"github.com/masmgr/bugspots-go/internal/output"
)

func parseRenameDetectFlag(s string, detail git.ChangeDetailLevel) (git.RenameDetectMode, error) {
	normalized := strings.ToLower(strings.TrimSpace(s))

	switch normalized {
	case "", "auto":
		// Performance default: exact renames only.
		// Callers can opt back into similarity-based detection via "aggressive".
		return git.RenameDetectSimple, nil
	case "off", "none", "false", "0":
		return git.RenameDetectOff, nil
	case "simple", "exact":
		return git.RenameDetectSimple, nil
	case "aggressive", "similar", "similarity":
		return git.RenameDetectAggressive, nil
	default:
		_ = detail // reserved for potential future auto-tuning
		return git.RenameDetectAggressive, fmt.Errorf("invalid --rename-detect %q (expected auto|off|simple|aggressive)", s)
	}
}

// CommandContext holds common state for command execution.
// It encapsulates the shared setup logic across all analysis commands.
type CommandContext struct {
	Config     *config.Config
	RepoPath   string
	Since      *time.Time
	Until      time.Time
	Branch     string
	ChangeSets []git.CommitChangeSet
	StartTime  time.Time
}

type commandExecutor func(ctx *CommandContext, c *cli.Context) error

// NewCommandContext creates a context from CLI flags.
// It performs configuration loading, date parsing, repository opening, and history reading.
func NewCommandContext(c *cli.Context) (*CommandContext, error) {
	return NewCommandContextWithGitDetail(c, git.ChangeDetailFull)
}

// NewCommandContextWithGitDetail is like NewCommandContext, but allows callers to control
// the Git history detail level for performance-sensitive commands.
func NewCommandContextWithGitDetail(c *cli.Context, detail git.ChangeDetailLevel) (*CommandContext, error) {
	start := time.Now()

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

	renameDetect, err := parseRenameDetectFlag(c.String("rename-detect"), detail)
	if err != nil {
		return nil, err
	}

	reader, err := git.NewHistoryReader(git.ReadOptions{
		RepoPath:     repoPath,
		Branch:       branch,
		Since:        since,
		Until:        until,
		Include:      cfg.Filters.Include,
		Exclude:      cfg.Filters.Exclude,
		DetailLevel:  detail,
		RenameDetect: renameDetect,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	// Read commit changes
	changeSets, err := reader.ReadChanges(context.Background())
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
		StartTime:  start,
	}, nil
}

func executeWithContext(c *cli.Context, detail git.ChangeDetailLevel, exec commandExecutor) error {
	ctx, err := NewCommandContextWithGitDetail(c, detail)
	if err != nil {
		return err
	}
	defer ctx.LogCompletion()

	if !ctx.HasCommits() {
		ctx.PrintNoCommitsMessage()
		return nil
	}

	ctx.ApplyCLIOverrides(c)
	return exec(ctx, c)
}

// ApplyCLIOverrides applies command-specific CLI flag values to the config.
// It uses c.IsSet() to only override values explicitly provided by the user,
// avoiding silent ignoring of valid zero values.
func (ctx *CommandContext) ApplyCLIOverrides(c *cli.Context) {
	if c.IsSet("half-life") {
		ctx.Config.Scoring.HalfLifeDays = c.Int("half-life")
	}
	if c.IsSet("window-days") {
		ctx.Config.Burst.WindowDays = c.Int("window-days")
	}
	if c.IsSet("min-co-commits") {
		ctx.Config.Coupling.MinCoCommits = c.Int("min-co-commits")
	}
	if c.IsSet("min-jaccard") {
		ctx.Config.Coupling.MinJaccardThreshold = c.Float64("min-jaccard")
	}
	if c.IsSet("max-files") {
		ctx.Config.Coupling.MaxFilesPerCommit = c.Int("max-files")
	}
	if c.IsSet("top-pairs") {
		ctx.Config.Coupling.TopPairs = c.Int("top-pairs")
	}
}

// HasCommits returns true if commits were found in the specified range.
func (ctx *CommandContext) HasCommits() bool {
	return len(ctx.ChangeSets) > 0
}

// PrintNoCommitsMessage prints a message when no commits are found.
func (ctx *CommandContext) PrintNoCommitsMessage() {
	fmt.Println("No commits found in the specified range.")
}

// LogCompletion prints the elapsed time since the command started.
func (ctx *CommandContext) LogCompletion() {
	fmt.Fprintf(os.Stderr, "\nCompleted in %s\n", time.Since(ctx.StartTime))
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
