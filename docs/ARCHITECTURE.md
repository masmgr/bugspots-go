# Architecture

This document describes the architecture of bugspots-go v2.

---

## Table of Contents

1. [Directory Structure](#1-directory-structure)
2. [Entry Point](#2-entry-point)
3. [CLI Layer (cmd/)](#3-cli-layer-cmd)
4. [Internal Packages](#4-internal-packages)
5. [Configuration (config/)](#5-configuration-config)
6. [Key Interfaces and Data Structures](#6-key-interfaces-and-data-structures)
7. [Data Flow](#7-data-flow)
8. [Design Patterns](#8-design-patterns)
9. [Dependencies](#9-dependencies)

---

## 1. Directory Structure

```
bugspots-go/
├── app.go                        # Entry point
├── go.mod / go.sum               # Module definition and dependency checksums
├── testhelpers_test.go           # Root-level test utilities
│
├── cmd/                          # CLI command definitions
│   ├── root.go                   # App structure, common flags, config loading
│   ├── context.go                # CommandContext (shared setup logic)
│   ├── analyze.go                # 6-factor file hotspot analysis
│   ├── commits.go                # JIT commit risk analysis
│   ├── coupling.go               # File change coupling analysis
│   └── calibrate.go              # Score weight calibration
│
├── config/                       # Configuration management
│   ├── config.go                 # Config structs, loading, defaults
│   └── config_test.go
│
├── internal/                     # Core packages (not importable externally)
│   ├── git/                      # Git CLI interface
│   │   ├── interfaces.go         # RepositoryReader interface
│   │   ├── models.go             # CommitInfo, FileChange, CommitChangeSet
│   │   ├── reader.go             # HistoryReader, ReadOptions, glob filtering
│   │   ├── reader_gitcli.go      # Git CLI output parsing
│   │   ├── diff.go               # Diff reading for PR/CI integration
│   │   ├── filemode.go           # Git file mode parsing
│   │   └── mock_reader.go        # Mock for testing
│   │
│   ├── aggregation/              # Metrics aggregation
│   │   ├── file_metrics.go       # Per-file metrics (commits, churn, ownership)
│   │   └── commit_metrics.go     # Per-commit metrics (diffusion, size, entropy)
│   │
│   ├── scoring/                  # Risk scoring algorithms
│   │   ├── file_scorer.go        # 6-factor weighted file risk scoring
│   │   ├── commit_scorer.go      # JIT commit risk scoring
│   │   └── normalization.go      # NormLog, NormMinMax, RecencyDecay, Clamp
│   │
│   ├── bugfix/                   # Bugfix commit detection
│   │   └── detector.go           # Regex pattern matching on commit messages
│   │
│   ├── burst/                    # Burst detection
│   │   └── sliding_window.go     # Two-pointer sliding window algorithm
│   │
│   ├── entropy/                  # Shannon entropy
│   │   └── shannon.go            # Normalized entropy for change distribution
│   │
│   ├── coupling/                 # File change coupling
│   │   └── analyzer.go           # Jaccard coefficient-based analysis
│   │
│   └── output/                   # Multi-format output writers
│       ├── formatter.go          # Writer interfaces and report structures
│       ├── console.go            # Colored table output
│       ├── json.go               # JSON output
│       ├── csv.go                # CSV output
│       ├── markdown.go           # Markdown table output
│       └── ci.go                 # CI/NDJSON streaming output
│
├── docs/                         # Documentation
│   ├── ARCHITECTURE.md           # This file
│   ├── SCORING.md                # Scoring algorithms reference
│   └── FEATURE_ROADMAP.md        # Feature plans
│
└── .github/workflows/
    └── ci.yml                    # GitHub Actions CI
```

---

## 2. Entry Point

`app.go` is a minimal entry point that delegates all work to the `cmd` package.

```go
func main() {
    app := cmd.App()
    if err := app.Run(os.Args); err != nil {
        log.Fatal(err)
    }
}
```

---

## 3. CLI Layer (cmd/)

The CLI is built with `urfave/cli/v2`. Each command is defined in its own file.

### root.go

- `App()` creates the CLI application and registers subcommands
- `commonFlags()` returns flags shared across commands (`--repo`, `--since`, `--until`, `--format`, `--output`, `--explain`, `--top`)
- Utility functions: `parseDateFlag()`, `getOutputFormat()`, `loadConfig()`

### context.go

`CommandContext` encapsulates the shared setup logic used by all commands:

1. Load configuration from `.bugspots.json` or defaults
2. Apply CLI flag overrides
3. Parse date range flags
4. Initialize `HistoryReader` with `ReadOptions`
5. Read Git history into `[]CommitChangeSet`

Helper methods: `HasCommits()`, `PrintNoCommitsMessage()`, `LogCompletion()`.

### Command Files

| File | Subcommand | Purpose |
|------|-----------|---------|
| `analyze.go` | `analyze` | 6-factor file hotspot analysis. Supports `--diff` for PR/CI and `--ci-threshold` for quality gates |
| `commits.go` | `commits` | JIT defect prediction scoring individual commits |
| `coupling.go` | `coupling` | File change coupling analysis using Jaccard coefficient |
| `calibrate.go` | `calibrate` | Score weight calibration using historical bugfix data |

---

## 4. Internal Packages

### internal/git

Abstracts Git history reading via the Git CLI (replaced go-git library in v2).

- **`RepositoryReader`** interface with `ReadChanges(ctx) ([]CommitChangeSet, error)`
- **`HistoryReader`** implements `RepositoryReader` by parsing `git log --raw -z --numstat -z` output
- Supports branch selection, date range filtering, rename detection (off / simple / aggressive), and glob-based file include/exclude patterns
- **`ReadDiff()`** parses `git diff --name-status -z` for PR/CI integration
- Filter results and ownership ratios are cached for performance

### internal/aggregation

Aggregates raw commit data into per-file and per-commit metrics.

- **`FileMetricsAggregator`** processes `[]CommitChangeSet` and produces `map[string]*FileMetrics`
  - Tracks commit count, churn, contributors, commit times, bugfix count
  - Handles file renames via path aliasing (merges metrics when renames are detected)
- **`CommitMetricsCalculator`** produces `[]CommitMetrics`
  - Extracts NF (files), ND (directories), NS (subsystems), churn, and Shannon entropy per commit

### internal/scoring

Risk scoring algorithms that transform metrics into `[0, 1]` risk scores.

- **`FileScorer`** applies 6-factor weighted scoring: commit frequency, churn, recency, burst, ownership dispersion, bugfix count
- **`CommitScorer`** applies 3-factor weighted scoring: diffusion, size, entropy. Classifies results into risk levels (high / medium / low)
- **Normalization utilities**: `NormLog()`, `NormMinMax()`, `RecencyDecay()`, `Clamp()`

See [SCORING.md](SCORING.md) for formula details.

### internal/bugfix

Detects bugfix commits by matching commit messages against configurable regex patterns (e.g., `\bfix(ed|es)?\b`, `\bbug\b`). Returns per-file bugfix counts for integration with file scoring.

### internal/burst

Calculates burst scores using a two-pointer sliding window algorithm (O(n) complexity). Measures how concentrated commits are within a configurable time window (default: 7 days). Returns `maxCommitsInWindow / totalCommits` as a `[0, 1]` score.

### internal/entropy

Computes normalized Shannon entropy for commit change distribution: `-sum(p_i * log2(p_i)) / log2(n)`. A score of 0 means changes are focused in one file; 1 means evenly distributed.

### internal/coupling

Analyzes implicit dependencies between files by tracking co-occurrence in commits. Calculates Jaccard coefficient, confidence, and lift for file pairs. Filters by configurable thresholds (minimum co-commits, minimum Jaccard, maximum files per commit).

### internal/output

Multi-format output writers implementing three interfaces:

| Interface | Formats |
|-----------|---------|
| `FileReportWriter` | Console, JSON, CSV, Markdown, CI |
| `CommitReportWriter` | Console, JSON, CSV, Markdown, CI |
| `CouplingReportWriter` | Console, JSON, CSV, Markdown |

Factory functions (`NewFileReportWriter()`, etc.) create writers by format.

---

## 5. Configuration (config/)

JSON-based configuration via `.bugspots.json` files (searched in project root, then home directory). CLI flags override config values.

```go
type Config struct {
    Scoring       ScoringConfig       // File hotspot weights & half-life
    Burst         BurstConfig         // Window days
    Bugfix        BugfixConfig        // Regex patterns
    CommitScoring CommitScoringConfig // Commit risk weights & thresholds
    Coupling      CouplingConfig      // Min co-commits, Jaccard threshold
    Filters       FilterConfig        // Include/exclude glob patterns
}
```

`DefaultConfig()` provides sensible defaults. `RiskThresholds.Classify()` maps scores to risk levels (high / medium / low).

---

## 6. Key Interfaces and Data Structures

### Interfaces

```
git.RepositoryReader
├── ReadChanges(ctx) → []CommitChangeSet

output.FileReportWriter
├── Write(*FileAnalysisReport, OutputOptions) → error

output.CommitReportWriter
├── Write(*CommitAnalysisReport, OutputOptions) → error

output.CouplingReportWriter
├── Write(*CouplingAnalysisReport, OutputOptions) → error
```

### Core Data Structures

```
git.CommitChangeSet
├── Commit: CommitInfo {SHA, When, Author, Message}
└── Changes: []FileChange {Path, OldPath, LinesAdded, LinesDeleted, Kind}

aggregation.FileMetrics
├── Path, CommitCount, AddedLines, DeletedLines
├── Contributors, ContributorCommitCounts
├── CommitTimes, BurstScore, BugfixCount
└── OwnershipRatio() → float64

aggregation.CommitMetrics
├── SHA, When, Author, Message
├── FileCount, DirectoryCount, SubsystemCount
├── LinesAdded, LinesDeleted, ChangeEntropy

scoring.FileRiskItem
├── Path, RiskScore
├── Metrics: *FileMetrics
└── Breakdown: *ScoreBreakdown

scoring.CommitRiskItem
├── Metrics: CommitMetrics
├── RiskScore, RiskLevel
└── Breakdown: *CommitRiskBreakdown

coupling.ChangeCoupling
├── FileA, FileB
├── CoCommitCount, FileACommits, FileBCommits
├── JaccardCoefficient, Confidence, Lift
```

---

## 7. Data Flow

### analyze (file hotspots)

```
CLI flags
  │
  ▼
CommandContext
  ├── Load config (.bugspots.json or defaults)
  ├── Parse date range
  └── Initialize HistoryReader
        │
        ▼
  git log --raw -z --numstat -z
        │
        ▼
  []CommitChangeSet
        │
        ├──► FileMetricsAggregator ──► map[string]*FileMetrics
        │                                      │
        ├──► Bugfix Detector ──► per-file counts ──┘
        │                                      │
        ├──► Burst Calculator ──► burst scores ──┘
        │                                      │
        │                                      ▼
        │                              FileScorer (6-factor)
        │                                      │
        │                                      ▼
        │                              []FileRiskItem (sorted)
        │                                      │
        ├──► ReadDiff (optional) ──► filter to changed files
        │                                      │
        │                                      ▼
        └──────────────────────────► FileReportWriter ──► output
```

### commits (JIT prediction)

```
[]CommitChangeSet
  │
  ▼
CommitMetricsCalculator
  ├── Extract NF, ND, NS per commit
  ├── Calculate churn (LA + LD)
  └── Calculate Shannon entropy
        │
        ▼
  []CommitMetrics
        │
        ▼
  CommitScorer (3-factor)
  ├── Diffusion: avg(NormLog(NF), NormLog(ND), NormLog(NS))
  ├── Size: NormLog(totalChurn)
  └── Entropy: changeEntropy
        │
        ▼
  []CommitRiskItem (with risk level classification)
        │
        ▼
  Risk level filter (optional)
        │
        ▼
  CommitReportWriter ──► output
```

### coupling

```
[]CommitChangeSet (paths only, no line stats)
  │
  ▼
Coupling Analyzer
  ├── Track file co-occurrences
  ├── Calculate Jaccard, confidence, lift
  └── Filter by thresholds
        │
        ▼
  []ChangeCoupling (top N pairs)
        │
        ▼
  CouplingReportWriter ──► output
```

---

## 8. Design Patterns

| Pattern | Usage |
|---------|-------|
| **Repository** | `RepositoryReader` interface abstracts Git access, enabling mock-based testing |
| **Strategy** | Multiple output writers implement common interfaces; selected at runtime by format flag |
| **Factory** | `NewFileReportWriter()` and related functions create writers by format |
| **Template Method** | `CommandContext` encapsulates the shared setup sequence (config → dates → reader → history) |
| **Composition** | Commands compose aggregators, scorers, and writers without deep inheritance |

### Performance Optimizations

- Git CLI with NUL-separated binary output parsing (avoids ambiguity in filenames)
- Two-pointer sliding window for burst detection (O(n))
- `ChangeDetailPathsOnly` mode for coupling analysis (skips line stat parsing)
- Glob filter result caching in `HistoryReader`
- Ownership ratio caching in `FileMetrics`

---

## 9. Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/urfave/cli/v2` | CLI framework |
| `github.com/fatih/color` | Colored console output |
| `github.com/bmatcuk/doublestar/v4` | Glob pattern matching for file filters |

No external Git library is used. All Git operations are performed by invoking the `git` CLI directly and parsing its output.
