# bugspots-go

A Go implementation of Git repository hotspot analysis, based on the [Bugspots bug prediction heuristic](http://google-engtools.blogspot.com/2011/12/bug-prediction-at-google.html). This tool analyzes Git repositories to identify files and commits most likely to contain bugs based on historical change patterns.

## Overview

bugspots-go provides three analysis modes:

### File Hotspot Analysis (`analyze`)
Examines your Git repository's commit history and calculates risk scores for each file based on:

- **Commit Frequency** (30%): Files changed frequently may have more issues
- **Code Churn** (25%): High volume of added/deleted lines indicates instability
- **Recency** (20%): Recently changed files are more likely to have new bugs
- **Burst Activity** (15%): Concentrated changes in short time periods suggest rushed work
- **Ownership** (10%): Many contributors may indicate unclear ownership

### JIT Commit Risk Analysis (`commits`)
Analyzes individual commits for Just-In-Time (JIT) defect prediction based on research-backed metrics:

- **Diffusion Metrics** (35%): Number of files (NF), directories (ND), and subsystems (NS) affected
- **Size Metrics** (35%): Lines added (LA) and lines deleted (LD)
- **Change Entropy** (30%): How spread out the changes are across files (Shannon entropy)

### Change Coupling Analysis (`coupling`)
Detects file pairs that frequently change together, indicating hidden dependencies:

- **Jaccard Coefficient**: Similarity measure between file change sets
- **Confidence**: Probability that file B changes when file A changes
- **Lift**: How much more likely files change together than by chance

### Legacy Bugspots Mode
The original bugspots algorithm using sigmoid-weighted scoring of bugfix commits.

## Installation

### Build from source

```bash
cd bugspots-go
go build -o bugspots-go .
```

## Usage

### File Hotspot Analysis

```bash
# Analyze current directory
./bugspots-go analyze

# Analyze specific repository
./bugspots-go analyze --repo /path/to/repo

# Analyze with date range
./bugspots-go analyze --repo /path/to/repo --since 2025-01-01 --top 30

# Show score breakdown
./bugspots-go analyze --repo /path/to/repo --explain

# Export to JSON
./bugspots-go analyze --repo /path/to/repo --format json --output hotspots.json
```

### JIT Commit Risk Analysis

```bash
# Analyze commits for risk
./bugspots-go commits

# Analyze commits with date range
./bugspots-go commits --repo /path/to/repo --since 2025-01-01 --top 20

# Filter by risk level
./bugspots-go commits --repo /path/to/repo --risk-level high

# Show detailed score breakdown
./bugspots-go commits --repo /path/to/repo --explain

# Export to JSON
./bugspots-go commits --repo /path/to/repo --format json --output commits.json
```

### Change Coupling Analysis

```bash
# Analyze file coupling patterns
./bugspots-go coupling

# Analyze with minimum thresholds
./bugspots-go coupling --repo /path/to/repo --min-co-commits 5 --min-jaccard 0.2

# Show top coupling pairs
./bugspots-go coupling --repo /path/to/repo --top-pairs 100

# Skip large refactoring commits
./bugspots-go coupling --repo /path/to/repo --max-files 30

# Export to JSON
./bugspots-go coupling --repo /path/to/repo --format json --output coupling.json
```

### Output Formats

```bash
# Console output (default)
./bugspots-go analyze --repo /path/to/repo

# JSON output
./bugspots-go analyze --repo /path/to/repo --format json --output hotspots.json

# CSV output
./bugspots-go analyze --repo /path/to/repo --format csv --output hotspots.csv

# Markdown output (great for PR comments)
./bugspots-go analyze --repo /path/to/repo --format markdown --output hotspots.md
```

### Filtering Files

```bash
# Include only specific paths (in config file)
./bugspots-go analyze --config config.json

# Example config.json with filters:
# {
#   "filters": {
#     "include": ["src/**", "apps/**"],
#     "exclude": ["**/vendor/**", "**/testdata/**"]
#   }
# }
```

### Legacy Mode (Original Bugspots)

```bash
# Scan default branch (master) with default bugfix indicators
./bugspots-go /path/to/repo

# Scan specific branch
./bugspots-go -b develop /path/to/repo

# Use custom bugfix indicator words
./bugspots-go -w "fixes,closed,resolved" /path/to/repo

# Use custom regex pattern for bugfix detection
./bugspots-go -r "fix(es|ed)?" /path/to/repo

# Show timestamps of identified fix commits
./bugspots-go --display-timestamps /path/to/repo
```

## CLI Options

### Common Options (all commands)

| Option | Alias | Description | Default |
|--------|-------|-------------|---------|
| `--repo <PATH>` | `-r` | Path to Git repository | Current directory |
| `--since <DATE>` | | Start date for analysis (YYYY-MM-DD) | All history |
| `--until <DATE>` | | End date for analysis | Now |
| `--branch <NAME>` | `-b` | Branch to analyze | HEAD |
| `--top <N>` | `-n` | Number of top results | 20 |
| `--format <FORMAT>` | `-f` | Output format: console, json, csv, markdown | console |
| `--output <PATH>` | `-o` | Output file path | stdout |
| `--explain` | `-e` | Include score breakdown | false |
| `--config <PATH>` | `-c` | Configuration file path | .bugspots.json |

### `analyze` Command Options

| Option | Description | Default |
|--------|-------------|---------|
| `--half-life <DAYS>` | Half-life for recency decay (days) | 30 |

### `commits` Command Options

| Option | Alias | Description | Default |
|--------|-------|-------------|---------|
| `--risk-level <LEVEL>` | `-l` | Filter by risk: high, medium, all | all |

### `coupling` Command Options

| Option | Description | Default |
|--------|-------------|---------|
| `--min-co-commits <N>` | Minimum co-commits to consider coupling | 3 |
| `--min-jaccard <FLOAT>` | Minimum Jaccard coefficient threshold | 0.1 |
| `--max-files <N>` | Maximum files per commit (skip large commits) | 50 |
| `--top-pairs <N>` | Number of top coupled pairs to report | 50 |

## Configuration File

Create a `.bugspots.json` or specify with `--config`:

```json
{
  "scoring": {
    "halfLifeDays": 30,
    "weights": {
      "commit": 0.30,
      "churn": 0.25,
      "recency": 0.20,
      "burst": 0.15,
      "ownership": 0.10
    }
  },
  "burst": {
    "windowDays": 7
  },
  "commitScoring": {
    "weights": {
      "diffusion": 0.35,
      "size": 0.35,
      "entropy": 0.30
    },
    "thresholds": {
      "high": 0.7,
      "medium": 0.4
    }
  },
  "coupling": {
    "minCoCommits": 3,
    "minJaccardThreshold": 0.1,
    "maxFilesPerCommit": 50,
    "topPairs": 50
  },
  "filters": {
    "include": ["src/**", "apps/**"],
    "exclude": ["**/vendor/**", "**/testdata/**", "**/*.min.js"]
  }
}
```

## Risk Score Formulas

### File Hotspot Score

The risk score for each file is calculated as:

```
RiskScore =
  0.30 * norm_log(CommitCount)
+ 0.25 * norm_log(ChurnTotal)
+ 0.20 * recency_decay(days_since_modified)
+ 0.15 * BurstScore
+ 0.10 * OwnershipDispersion
```

Where:
- `norm_log(x)`: Logarithmic normalization: `(log(1+x) - log(1+min)) / (log(1+max) - log(1+min))`
- `recency_decay(d)`: Exponential decay: `exp(-ln(2) * d / halfLifeDays)`
- `BurstScore`: Proportion of commits in the densest N-day window
- `OwnershipDispersion`: `1 - (maxContributorCommits / totalCommits)`

### JIT Commit Risk Score

The risk score for each commit is calculated as:

```
RiskScore =
  0.35 * DiffusionScore
+ 0.35 * SizeScore
+ 0.30 * EntropyScore
```

Where:
- `DiffusionScore`: Normalized combination of file count, directory count, and subsystem count
- `SizeScore`: Normalized combination of lines added and lines deleted
- `EntropyScore`: Shannon entropy of change distribution across files (0 = focused, 1 = spread)

Risk levels:
- **High**: Score >= 0.7
- **Medium**: Score >= 0.4
- **Low**: Score < 0.4

### Coupling Metrics

- **Jaccard Coefficient**: `|A intersection B| / |A union B|`
- **Confidence(A->B)**: `CoCommitCount(A,B) / CommitCount(A)`
- **Lift**: `P(A,B) / (P(A) * P(B))`

## Output Examples

### Console Output (File Hotspots)

```
File Hotspots (Top 10)
Repository: /path/to/repo
Period: 2025-01-01 to 2025-02-04

+----+-----------------------------+-------+---------+-----------+--------------+--------------+-------+
| #  | Path                        | Score | Commits | Churn     | Last Modified| Contributors | Burst |
+----+-----------------------------+-------+---------+-----------+--------------+--------------+-------+
| 1  | src/core/engine.go          |  0.82 |      18 | +420/-390 |   2025-02-01 |            5 |  0.73 |
| 2  | src/api/controller.go       |  0.77 |      15 | +300/-200 |   2025-01-29 |            4 |  0.65 |
| 3  | src/services/handler.go     |  0.71 |      12 | +250/-180 |   2025-01-28 |            3 |  0.58 |
+----+-----------------------------+-------+---------+-----------+--------------+--------------+-------+

Note: Risk score is an indicator, not a definitive measure of bugs.
```

### JSON Output (File Hotspots with --explain)

```json
{
  "repo": "/path/to/repo",
  "since": "2025-01-01T00:00:00Z",
  "until": "2025-02-04T00:00:00Z",
  "generatedAt": "2025-02-04T12:34:56Z",
  "items": [
    {
      "path": "src/core/engine.go",
      "riskScore": 0.82,
      "metrics": {
        "commitCount": 18,
        "addedLines": 420,
        "deletedLines": 390,
        "lastModified": "2025-02-01T09:10:00Z",
        "contributorCount": 5,
        "burstScore": 0.73
      },
      "breakdown": {
        "commitComponent": 0.25,
        "churnComponent": 0.22,
        "recencyComponent": 0.18,
        "burstComponent": 0.11,
        "ownershipComponent": 0.06
      }
    }
  ]
}
```

### JSON Output (JIT Commit Risk)

```json
{
  "repo": "/path/to/repo",
  "since": "2025-01-01T00:00:00Z",
  "until": "2025-02-04T00:00:00Z",
  "generatedAt": "2025-02-04T12:34:56Z",
  "items": [
    {
      "sha": "abc1234",
      "message": "Refactor auth module",
      "author": "Developer <dev@example.com>",
      "when": "2025-02-01T10:00:00Z",
      "riskScore": 0.85,
      "riskLevel": "high",
      "metrics": {
        "fileCount": 12,
        "directoryCount": 5,
        "subsystemCount": 3,
        "linesAdded": 450,
        "linesDeleted": 200,
        "changeEntropy": 0.78
      },
      "breakdown": {
        "diffusionComponent": 0.30,
        "sizeComponent": 0.32,
        "entropyComponent": 0.23
      }
    }
  ]
}
```

### JSON Output (Coupling Analysis)

```json
{
  "repo": "/path/to/repo",
  "since": "2025-01-01T00:00:00Z",
  "until": "2025-02-04T00:00:00Z",
  "generatedAt": "2025-02-04T12:34:56Z",
  "totalCommitsAnalyzed": 150,
  "pairs": [
    {
      "fileA": "src/api/handler.go",
      "fileB": "src/api/handler_test.go",
      "coCommits": 25,
      "jaccard": 0.83,
      "confidence": 0.89,
      "lift": 4.2
    }
  ]
}
```

## Use Cases

### Weekly Hotspot Report

Run weekly to identify files that need attention:

```bash
./bugspots-go analyze --repo . --since $(date -d "7 days ago" +%Y-%m-%d) --format markdown --output weekly-hotspots.md
```

### CI Integration

Add to your CI pipeline to warn about changes to high-risk files:

```yaml
- name: Check Hotspots
  run: |
    ./bugspots-go analyze --repo . --format json --output hotspots.json
    ./bugspots-go commits --repo . --risk-level high --format json --output risky-commits.json
```

### AI Review Focus

Generate a list of high-risk files for AI code review:

```bash
./bugspots-go analyze --repo . --top 10 --format json | jq '.items[].path'
```

### Detect Hidden Dependencies

Find files that should be reviewed together:

```bash
./bugspots-go coupling --repo . --min-jaccard 0.5 --format markdown
```

## Important Notes

- **Risk is not Bugs**: The risk score is an indicator based on change patterns, not a definitive measure of bugs. Use it to prioritize review efforts, not as absolute truth.
- **Context Matters**: A high-risk score might indicate a file that's actively being improved, not necessarily one that's problematic.
- **Large Commits**: The coupling analysis automatically skips commits with many files (configurable via `--max-files`) to avoid noise from refactoring or merge commits.

## Testing

```bash
go test ./...
```

## Code Style

```bash
go fmt ./...
```

## Dependencies

- `github.com/urfave/cli/v2` - Command-line interface framework
- `github.com/go-git/go-git/v5` - Git repository interaction
- `github.com/fatih/color` - Colored console output
- `github.com/olekukonko/tablewriter` - Table output formatting
- `github.com/bmatcuk/doublestar/v4` - Glob pattern matching

## Project Structure

```
bugspots-go/
├── app.go                      # Entry point (legacy mode support)
├── bugspots.go                 # Legacy bugspots algorithm
├── cmd/
│   ├── root.go                 # CLI configuration, common flags
│   ├── analyze.go              # File hotspot analysis command
│   ├── commits.go              # JIT commit risk analysis command
│   └── coupling.go             # Change coupling analysis command
├── config/
│   └── config.go               # Configuration structures
├── internal/
│   ├── git/
│   │   ├── models.go           # CommitInfo, FileChange, CommitChangeSet
│   │   └── reader.go           # Git history reader (go-git)
│   ├── scoring/
│   │   ├── normalization.go    # NormLog, RecencyDecay, MinMax
│   │   ├── file_scorer.go      # 5-factor file scoring
│   │   └── commit_scorer.go    # JIT commit scoring
│   ├── aggregation/
│   │   ├── file_metrics.go     # File-level metrics aggregation
│   │   └── commit_metrics.go   # Commit-level metrics calculation
│   ├── burst/
│   │   └── sliding_window.go   # O(n) burst score calculation
│   ├── entropy/
│   │   └── shannon.go          # Shannon entropy calculation
│   ├── coupling/
│   │   └── analyzer.go         # Change coupling analysis
│   └── output/
│       ├── formatter.go        # Output interfaces
│       ├── console.go          # Console table output
│       ├── json.go             # JSON output
│       ├── csv.go              # CSV output
│       └── markdown.go         # Markdown output
└── go.mod
```

## Related Projects

- [igrigorik/bugspots](https://github.com/igrigorik/bugspots) - Original Ruby implementation
- [Google Engineering Tools Blog](http://google-engtools.blogspot.com/2011/12/bug-prediction-at-google.html) - Original research

## License

MIT License

## Acknowledgments

Inspired by the original [bugspots](https://github.com/igrigorik/bugspots) algorithm by Ilya Grigorik and JIT defect prediction research.
