# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

bugspots-go is a Go implementation of the [Bugspots bug prediction heuristic](http://google-engtools.blogspot.com/2011/12/bug-prediction-at-google.html) originally created by igrigorik. It analyzes Git repositories to identify files most likely to contain bugs based on historical fix commits. Version 2.0 extends the original concept with multi-factor file scoring, JIT commit risk analysis, and file coupling detection.

## Build and Run Commands

### Build the project
```bash
go build -o bugspots-go .
```

### Run the tool
```bash
./bugspots-go <command> [flags]
```

### Common usage examples
```bash
# Multi-factor file hotspot analysis (recommended)
./bugspots-go analyze --repo /path/to/repo
./bugspots-go analyze --repo /path/to/repo --since 2025-01-01
./bugspots-go analyze --repo /path/to/repo --format json --output report.json
./bugspots-go analyze --repo /path/to/repo --diff origin/main...HEAD
./bugspots-go analyze --repo /path/to/repo --include-complexity --explain

# JIT commit risk analysis
./bugspots-go commits --repo /path/to/repo
./bugspots-go commits --repo /path/to/repo --risk-level high

# File change coupling analysis
./bugspots-go coupling --repo /path/to/repo
./bugspots-go coupling --repo /path/to/repo --min-co-commits 5 --min-jaccard 0.3

# Score weight calibration
./bugspots-go calibrate --repo /path/to/repo --since 2025-01-01
./bugspots-go calibrate --repo /path/to/repo --top-percent 30
```

### Run tests
```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run with coverage
go test -cover ./...

# Generate detailed coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific test function
go test -v -run TestCalcScore ./...

# Run benchmarks
go test -bench=. ./internal/git/
```

### Lint code
```bash
# Format code
go fmt ./...

# Run golangci-lint (used in CI)
golangci-lint run --timeout=5m

# Check if code needs formatting
gofmt -l .
```

## Architecture

The project uses a modular architecture with four CLI commands backed by internal packages.

### app.go - Entry Point
- Minimal entry point that delegates to `cmd.App()`

### cmd/ - Command Definitions
Each command is defined in its own file:

- **root.go** - Defines the CLI app structure with `urfave/cli/v2`, common flags, config loading, and output format parsing
- **analyze.go** - `analyze` command: multi-factor file hotspot analysis (commit frequency, churn, recency, burst, ownership, bugfix, complexity). Supports `--diff` for PR/CI integration, `--ci-threshold` for automated gating, and `--include-complexity` for file size scoring
- **commits.go** - `commits` command: JIT defect prediction scoring individual commits by diffusion, size, and entropy. Supports `--risk-level` filtering
- **coupling.go** - `coupling` command: file change coupling analysis using Jaccard coefficient with configurable thresholds
- **calibrate.go** - `calibrate` command: weight calibration using historical bugfix data. Optimizes scoring weights via coordinate descent to maximize detection rate
- **context.go** - `CommandContext` struct encapsulating shared setup logic (config, date parsing, Git reader initialization) used by all commands

### internal/ - Core Packages

- **internal/git/** - Git CLI interface (replaced go-git library). `HistoryReader` reads commit history via `git log` with numstat/raw output. Supports branch selection, date range filtering, rename detection, and file include/exclude glob patterns
- **internal/bugfix/** - Pattern-based bugfix commit detection using configurable regex patterns
- **internal/scoring/** - Scoring algorithms:
  - `file_scorer.go` - Multi-factor weighted file risk scoring (7 factors including complexity)
  - `commit_scorer.go` - JIT commit risk scoring (diffusion, size, entropy)
  - `normalization.go` - Score normalization utilities (min-max, logarithmic, recency decay, clamping)
- **internal/aggregation/** - Metrics aggregation from commit history:
  - `file_metrics.go` - Per-file metrics (commit count, churn, contributors, ownership ratio, burst scores)
  - `commit_metrics.go` - Per-commit metrics (file count, directories, subsystems, total churn)
- **internal/output/** - Multi-format output writers (console, JSON, CSV, Markdown, CI/NDJSON) for file, commit, and coupling reports
- **internal/coupling/** - File change coupling analysis using Jaccard coefficient
- **internal/burst/** - Sliding window burst detection for commit frequency analysis
- **internal/entropy/** - Shannon entropy calculation for commit change distribution
- **internal/complexity/** - File complexity measurement via git cat-file (line count)
- **internal/calibration/** - Score weight calibration using coordinate descent optimization

### config/ - Configuration
- JSON-based configuration via `.bugspots.json` files (project root or home directory)
- Configurable scoring weights, bugfix patterns, and coupling thresholds
- CLI flags override config file values

## Key Data Structures

- **git.CommitChangeSet** - A commit with its associated file changes (path, lines added/deleted, change kind)
- **git.FileChange** - Individual file change within a commit (path, old path for renames, added/deleted lines, change kind)
- **scoring.FileRiskItem** - File risk result with score and optional breakdown
- **scoring.CommitRiskItem** - Commit risk result with score, risk level, and optional breakdown
- **aggregation.FileMetrics** - Accumulated per-file metrics across all commits
- **config.Config** - Root configuration structure with scoring, bugfix, coupling, and filter settings

## Test Suite

The test suite includes 23 test files across all packages with 90+ test functions and 280+ test cases. See [TESTS.md](docs/TESTS.md) for detailed test documentation.

### Key test areas:
- **config/** - Configuration defaults and risk level classification
- **internal/git/** - Git CLI output parsing, branch handling, filters, diff parsing, benchmarks
- **internal/scoring/** - File scoring, commit scoring, normalization
- **internal/aggregation/** - File and commit metrics aggregation, rename handling
- **internal/bugfix/** - Pattern detection and bugfix identification
- **internal/output/** - Output format writers and helpers
- **internal/coupling/** - Coupling analysis and Jaccard coefficient
- **internal/burst/** - Sliding window burst detection
- **internal/entropy/** - Shannon entropy calculation

### Root-level test helpers (`testhelpers_test.go`):
- `createTestRepo()` - Creates temporary git repositories via git CLI
- `addCommitToRepo()` - Adds commits with custom timestamps
- `suppressOutput()` / `discardOutput()` - Suppresses stdout during tests
- `runGit()` - Runs git commands in test directories

## Dependencies

- `github.com/urfave/cli/v2` - CLI framework
- `github.com/fatih/color` - Colored console output
- `github.com/bmatcuk/doublestar/v4` - Glob pattern matching for file filters
