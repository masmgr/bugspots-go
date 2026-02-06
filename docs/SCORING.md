# Scoring & Calculation Methods

bugspots-go v2 provides three analysis capabilities: file hotspot analysis, commit risk analysis, and file coupling analysis. This document describes the scoring algorithms and calculation methods used by each feature.

---

## Table of Contents

1. [File Hotspot Analysis (analyze)](#1-file-hotspot-analysis-analyze)
2. [JIT Commit Risk Analysis (commits)](#2-jit-commit-risk-analysis-commits)
3. [File Coupling Analysis (coupling)](#3-file-coupling-analysis-coupling)
4. [Legacy Scan (scan)](#4-legacy-scan-scan)
5. [Normalization Methods](#5-normalization-methods)
6. [Burst Detection](#6-burst-detection)
7. [Shannon Entropy](#7-shannon-entropy)
8. [Bugfix Commit Detection](#8-bugfix-commit-detection)
9. [Configuration Reference](#9-configuration-reference)

---

## 1. File Hotspot Analysis (analyze)

Six-factor file risk scoring executed by the `bugspots-go analyze` command.

### Overall Score

```
total_score = commit_component + churn_component + recency_component
            + burst_component + ownership_component + bugfix_component
```

Each component is calculated as `weight × normalized_value`, and the overall score falls within the `[0, 1]` range.

### The Six Factors

| # | Factor | Default Weight | Formula | Description |
|---|--------|---------------|---------|-------------|
| 1 | Commit Frequency | 0.25 | `w × NormLog(commitCount)` | Number of times the file was modified |
| 2 | Code Churn | 0.20 | `w × NormLog(addedLines + deletedLines)` | Total lines added and deleted |
| 3 | Recency | 0.15 | `w × RecencyDecay(daysSinceLastModified)` | Days since last modification |
| 4 | Burst | 0.10 | `w × burstScore` | Temporal clustering of commits |
| 5 | Ownership | 0.10 | `w × (1 - ownershipRatio)` | Dispersion of contributors (dispersed = higher risk) |
| 6 | Bugfix | 0.20 | `w × NormLog(bugfixCount)` | Number of bugfix commits |

#### Commit Frequency

The total number of commits touching a file, log-normalized. Files that are modified frequently receive higher scores.

#### Code Churn

The sum of lines added and deleted (`ChurnTotal = LinesAdded + LinesDeleted`), log-normalized. Files with large cumulative changes receive higher scores.

#### Recency

An exponential decay function is applied to the number of days since the file was last modified. Recently changed files receive higher scores. The default half-life is 30 days.

```
RecencyDecay(days) = exp(-ln(2) × days / halfLifeDays)
```

| Days Elapsed | Score (half-life = 30 days) |
|-------------|---------------------------|
| 0 days      | 1.000 |
| 15 days     | 0.707 |
| 30 days     | 0.500 |
| 60 days     | 0.250 |
| 90 days     | 0.125 |

#### Burst

Measures how concentrated commits are within a specific time window using a sliding window approach. See [6. Burst Detection](#6-burst-detection) for details.

#### Ownership

The commit ratio of the top contributor is calculated as `ownershipRatio`, and `1 - ownershipRatio` is used. This is based on the hypothesis that files concentrated under a single developer (high ownershipRatio) are lower risk, while files modified by many developers (low ownershipRatio) are higher risk.

```
ownershipRatio = topContributorCommits / totalCommits
ownershipComponent = weight × (1 - ownershipRatio)
```

#### Bugfix

The number of times a file was changed in bugfix commits, log-normalized. See [8. Bugfix Commit Detection](#8-bugfix-commit-detection) for details.

---

## 2. JIT Commit Risk Analysis (commits)

Scoring that evaluates the risk of individual commits, executed by the `bugspots-go commits` command. Based on Just-In-Time (JIT) defect prediction research.

### Overall Score

```
total_score = clamp(diffusion_component + size_component + entropy_component)
```

The score is clamped to `[0, 1]`.

### The Three Factors

| # | Factor | Default Weight | Formula | Description |
|---|--------|---------------|---------|-------------|
| 1 | Diffusion | 0.35 | `w × avg(NormLog(NF), NormLog(ND), NormLog(NS))` | Spatial spread of changes |
| 2 | Size | 0.35 | `w × NormLog(totalChurn)` | Total lines changed |
| 3 | Entropy | 0.30 | `w × changeEntropy` | Distribution of changes |

#### Diffusion

The average of log-normalized values for three metrics:

- **NF** (Number of Files): Number of files changed
- **ND** (Number of Directories): Number of directories changed
- **NS** (Number of Subsystems): Number of top-level directories changed

```
diffusion = weight × (NormLog(NF) + NormLog(ND) + NormLog(NS)) / 3
```

Changes that are widely spread across the codebase carry a higher risk of introducing defects.

#### Size

The total churn of a commit (lines added + lines deleted), log-normalized. Larger changes carry a higher defect risk.

#### Entropy

Measures how evenly changes are distributed across files within a commit using Shannon entropy. See [7. Shannon Entropy](#7-shannon-entropy) for details.

### Risk Level Classification

| Risk Level | Score Range | Default Threshold |
|-----------|------------|------------------|
| High      | `score >= 0.7` | 0.7 |
| Medium    | `0.4 <= score < 0.7` | 0.4 |
| Low       | `score < 0.4` | - |

---

## 3. File Coupling Analysis (coupling)

Analyzes the degree of change coupling between files, executed by the `bugspots-go coupling` command.

### Metrics

#### Jaccard Coefficient

Measures how frequently two files are changed together in the same commit.

```
jaccard(A, B) = |A ∩ B| / |A ∪ B|
             = coCommitCount / (fileACommits + fileBCommits - coCommitCount)
```

| Jaccard Coefficient | Interpretation |
|--------------------|---------------|
| 1.0 | Perfect coupling (always changed together) |
| 0.5 | Moderate coupling |
| 0.0 | No coupling (never changed together) |

#### Confidence

The conditional probability that file B is changed when file A is changed.

```
confidence(A → B) = coCommitCount / fileACommits
```

#### Lift

Indicates how strongly the changes of two files are associated compared to independent events.

```
lift(A, B) = P(A, B) / (P(A) × P(B))
           = (coCommitCount / totalCommits) / ((fileACommits / totalCommits) × (fileBCommits / totalCommits))
```

| Lift Value | Interpretation |
|-----------|---------------|
| > 1.0 | Positive correlation (coupled) |
| = 1.0 | Independent (no correlation) |
| < 1.0 | Negative correlation |

### Filtering

| Setting | Default | Description |
|---------|---------|-------------|
| MinCoCommits | 3 | Minimum number of co-commits to consider a pair |
| MinJaccardThreshold | 0.1 | Minimum Jaccard coefficient |
| MaxFilesPerCommit | 50 | Commits exceeding this are skipped (excludes refactoring) |
| TopPairs | 50 | Maximum number of pairs to display |

---

## 4. Legacy Scan (scan)

Backward-compatible mode with the original bugspots algorithm, executed by the `bugspots-go scan` command.

### Sigmoid Score

Weights time using a sigmoid function, assigning higher scores to more recent bugfixes.

```
score = 1 / (1 + exp(-12t + 12))
```

Where `t` is a normalized time value in `[0, 1]`:

```
t = 1 - (currentDate - fixDate) / (currentDate - oldestDate)
```

| t Value | Meaning | Score (approx.) |
|---------|---------|----------------|
| 0.0 | Oldest commit | ≈ 0.000006 |
| 0.5 | Midpoint | ≈ 0.0025 |
| 0.8 | Fairly recent | ≈ 0.12 |
| 0.9 | Recent | ≈ 0.50 |
| 1.0 | Present | ≈ 1.00 |

The final score for a file is the sum of scores from all bugfix commits associated with that file.

---

## 5. Normalization Methods

All scoring components are normalized to the `[0, 1]` range.

### Logarithmic Normalization (NormLog)

A normalization method that suppresses the impact of outliers while preserving relative ordering. Used as the primary normalization method for both file scoring and commit scoring.

```
NormLog(x) = (log(1 + x) - log(1 + min)) / (log(1 + max) - log(1 + min))
```

**Special cases:**
- When `min == max`: returns `1.0` if `x > 0`, otherwise `0.0`
- Results are always clamped to `[0, 1]`

### Linear Normalization (NormMinMax)

```
NormMinMax(x) = (x - min) / (max - min)
```

### Recency Decay

An exponential decay function that applies temporal weighting based on a half-life.

```
RecencyDecay(days) = exp(-ln(2) × days / halfLifeDays)
```

**Property:** When `days == halfLifeDays`, the result is exactly `0.5`.

### Clamp

```
clamp(x) = max(0.0, min(1.0, x))
```

---

## 6. Burst Detection

Detects whether commits to a file are concentrated within a specific time period.

### Algorithm: Sliding Window

**Default window size:** 7 days

```
BurstScore = maxCommitsInWindow / totalCommits
```

1. Sort commit timestamps in ascending order
2. Manage the window using two pointers (left, right)
3. At each right position, count the number of commits within the window
4. Record the maximum number of commits in any window
5. Divide the maximum by the total number of commits

**Time complexity:** O(n) (two-pointer technique on a sorted array)

| BurstScore | Interpretation |
|-----------|---------------|
| 1.0 | All commits concentrated within a single window |
| 0.5 | Half of commits within the densest window |
| 0.1 | Only 10% of commits concentrated |

**Special cases:**
- No commits: `0.0`
- Single commit: `1.0`

---

## 7. Shannon Entropy

Measures how evenly changes are distributed across files within a commit.

### Formula

```
normalized_entropy = (-Σ(pᵢ × log₂(pᵢ))) / log₂(n)
```

- `pᵢ = fileᵢ_churn / total_churn` (churn ratio for each file)
- `n` = number of files in the commit
- `log₂(n)` is the maximum entropy (value when perfectly uniformly distributed)

### Examples

For a commit modifying 4 files (maximum entropy = log₂(4) = 2.0):

| Scenario | Change Distribution | Entropy | Normalized | Interpretation |
|----------|-------------------|---------|-----------|---------------|
| Focused | [100, 0, 0, 0] | 0.0 | 0.0 | Concentrated in 1 file |
| Skewed | [50, 25, 15, 10] | ≈ 1.61 | ≈ 0.805 | Somewhat dispersed |
| Uniform | [25, 25, 25, 25] | 2.0 | 1.0 | Perfectly even |

**Special cases:**
- No changes: `0.0`
- Single file only: `0.0`

---

## 8. Bugfix Commit Detection

Bugfix commits are identified by matching commit messages against specific patterns.

### Default Patterns (analyze / commits commands)

| Pattern | Matches |
|---------|---------|
| `\bfix(ed\|es)?\b` | fix, fixed, fixes |
| `\bbug\b` | bug |
| `\bhotfix\b` | hotfix |
| `\bpatch\b` | patch |

### Legacy Patterns (scan command)

| Pattern | Matches |
|---------|---------|
| `\b(fix(es\|ed)?\|close(s\|d)?)\b` | fix, fixed, fixes, close, closes, closed |

All patterns are case-insensitive (`(?i)` flag).

Custom patterns can be defined in the `.bugspots.json` configuration file.

---

## 9. Configuration Reference

All settings can be overridden via the `.bugspots.json` file or command-line flags.

### File Scoring

| Setting | Default | Description |
|---------|---------|-------------|
| `scoring.halfLifeDays` | 30 | Half-life for recency decay (days) |
| `scoring.weights.commit` | 0.25 | Weight for commit frequency |
| `scoring.weights.churn` | 0.20 | Weight for code churn |
| `scoring.weights.recency` | 0.15 | Weight for recency |
| `scoring.weights.burst` | 0.10 | Weight for burst |
| `scoring.weights.ownership` | 0.10 | Weight for ownership |
| `scoring.weights.bugfix` | 0.20 | Weight for bugfix |

### Commit Scoring

| Setting | Default | Description |
|---------|---------|-------------|
| `commitScoring.weights.diffusion` | 0.35 | Weight for diffusion |
| `commitScoring.weights.size` | 0.35 | Weight for size |
| `commitScoring.weights.entropy` | 0.30 | Weight for entropy |
| `commitScoring.thresholds.high` | 0.7 | Threshold for High risk classification |
| `commitScoring.thresholds.medium` | 0.4 | Threshold for Medium risk classification |

### Burst Detection

| Setting | Default | Description |
|---------|---------|-------------|
| `burst.windowDays` | 7 | Sliding window width (days) |

### File Coupling Analysis

| Setting | Default | Description |
|---------|---------|-------------|
| `coupling.minCoCommits` | 3 | Minimum number of co-commits |
| `coupling.minJaccardThreshold` | 0.1 | Minimum Jaccard coefficient |
| `coupling.maxFilesPerCommit` | 50 | Maximum files per commit |
| `coupling.topPairs` | 50 | Maximum number of pairs to display |

### Legacy Settings

| Setting | Default | Description |
|---------|---------|-------------|
| `legacy.analysisWindowYears` | 3 | Analysis window (years) |
| `legacy.maxHotspots` | 100 | Maximum number of hotspots to display |
| `legacy.defaultBranch` | `"master"` | Default branch |
| `legacy.defaultBugfixRegex` | `\b(fix(es\|ed)?\|close(s\|d)?)\b` | Bugfix detection pattern |

---

## References

- [Bug Prediction at Google](http://google-engtools.blogspot.com/2011/12/bug-prediction-at-google.html) - The original bugspots algorithm
- Kamei, Y. et al. "A Large-Scale Empirical Study of Just-In-Time Quality Assurance" - JIT defect prediction research
