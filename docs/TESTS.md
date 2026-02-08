# Test Suite Documentation

This document describes the test suite for bugspots-go.

## Overview

- **Test files**: 23
- **Test functions**: 91+
- **Test cases/subtests**: 280+
- **Benchmark functions**: 6
- **Coverage areas**: All packages (cmd, config, internal/*)

## Test Strategy

### Test Types

The project employs three types of tests.

| Type | Purpose | Examples |
|------|---------|----------|
| Unit tests | Verify correctness of individual functions and methods | `TestClamp`, `TestNormLog` |
| Integration tests | Validate behavior with real Git CLI and repositories | `TestReadDiff_Integration`, `TestHistoryReader_ReadChanges_RespectsBranch` |
| Benchmarks | Measure performance characteristics | `BenchmarkHistoryReader_ReadChanges_Full` |

### Design Principles

**1. Table-Driven Tests**

Over 95% of tests use the table-driven pattern. Test cases are defined in a `[]struct` slice and executed as subtests via `t.Run()`, enabling descriptive output and selective execution by name.

```go
tests := []struct {
    name     string
    input    string
    expected string
}{
    {name: "Single word", input: "fix", expected: "fix"},
    {name: "Two words", input: "fix,close", expected: "fix|close"},
}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) { ... })
}
```

**2. Minimal Mocking**

No mocking framework is used. Hand-written mocks are created only where necessary. `MockHistoryReader` (`internal/git/mock_reader_test.go`) is the sole mock in the codebase. Git operations are verified through integration tests using the real CLI, ensuring test fidelity matches production behavior.

**3. Programmatic Fixture Generation**

No static test data directories (`testdata/`) exist. Fixtures are generated dynamically at test time. Git repositories are created in temporary directories via `t.TempDir()` and populated using helper functions:

- `createTestRepo(t)` - Creates a temporary Git repository
- `addCommitToRepo(t, dir, msg, files, time)` - Adds commits with controlled timestamps
- `makeChangeSets()`, `makeChangeSet()` - Factory functions for test data

**4. Interface Compliance Verification**

Compile-time interface implementation checks use the standard Go idiom:

```go
var _ RepositoryReader = (*MockHistoryReader)(nil)
```

### Coverage Strategy

Test cases are structured in four layers:

**Happy Path**: Verifies basic functionality for each feature.

**Boundary Values**: Explicitly tests values at and around thresholds. Example: risk classification distinguishes `0.7` (high boundary) from `0.69` (just below high).

**Edge Cases**: Exhaustively covers boundary input conditions such as nil, empty slices, single elements, and unsorted input. Example: burst detection tests descending order, unsorted order, and input immutability.

**Mathematical Properties**: Scoring algorithms verify mathematical invariants including monotonicity (`TestNormLog_Monotonicity`, `TestRecencyDecay_MonotonicDecrease`) and weight sum constraints (`TestDefaultConfig_WeightsSum`). Floating-point comparisons use tolerance-based assertions.

### CI/CD

GitHub Actions (`.github/workflows/ci.yml`) runs the following checks automatically:

- `go test -v ./...` - Full test execution
- `go test -coverprofile=coverage.out ./...` - Coverage measurement with Codecov upload
- `golangci-lint run --timeout=5m` - Static analysis
- `gofmt -l .` - Format check

Triggered on push and pull requests to `main` and `develop` branches.

### Conventions

- Tests follow Go conventions: functions named `Test*`, benchmarks named `Benchmark*`
- Uses `testing.T` for assertions and error reporting
- All test files use `_test.go` suffix for automatic discovery
- Tests are isolated and can run in any order
- Integration tests use temporary directories via `t.TempDir()`
- No external dependencies required beyond the `git` binary
- Git operations use `os/exec` (no go-git dependency)

## Running the Tests

### Prerequisites

- Go 1.24 or later
- Git installed and configured

### Commands

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests with coverage
go test -cover ./...

# Run tests with detailed coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific test function
go test -v -run TestCalcScore ./...

# Run benchmarks
go test -bench=. ./internal/git/

# Run specific package tests
go test -v ./internal/scoring/...
go test -v ./internal/bugfix/...
go test -v ./cmd/...
```

## Test Coverage by Package

| Package | Test File(s) | Test Functions |
|---------|-------------|----------------|
| config | config_test.go | 3 |
| internal/aggregation | file_metrics_test.go, commit_metrics_test.go | 17 |
| internal/bugfix | detector_test.go | 10 |
| internal/burst | sliding_window_test.go | 13 |
| internal/coupling | analyzer_test.go | 12 |
| internal/entropy | shannon_test.go | 6 |
| internal/git | 7 test files | 16 + 6 benchmarks |
| internal/output | 3 test files | 9 |
| internal/scoring | 3 test files | 16 |
| (root) | testhelpers_test.go | 4 helpers |

## Test Files by Package

### 1. `config/config_test.go` - Configuration

| Test Function | Purpose | Cases |
|---------------|---------|-------|
| TestRiskThresholds_Classify | Risk level classification (high/medium/low) at boundary values | 9 |
| TestDefaultConfig | Validates all default configuration values | 15 |
| TestDefaultConfig_WeightsSum | File and commit scoring weights each sum to 1.0 | 2 |

### 2. `internal/aggregation/file_metrics_test.go` - File Metrics

| Test Function | Purpose | Cases |
|---------------|---------|-------|
| TestNewFileMetrics | FileMetrics initialization | 6 |
| TestFileMetrics_ChurnTotal | Total churn calculation (added + deleted) | 1 |
| TestFileMetrics_ContributorCount | Counting unique contributors | 1 |
| TestFileMetrics_OwnershipRatio | Ownership ratio for single/multiple contributors and no-commit case | 4 |
| TestFileMetrics_OwnershipRatio_Caching | Cache mechanism and invalidation on AddCommit | 1 |
| TestFileMetrics_AddCommit | Updating metrics when adding commits | 1 |
| TestFileMetrics_AddCommit_CommitTimesDisabled | Commit time collection can be disabled | 1 |
| TestFileMetricsAggregator_Process | Aggregating metrics from multiple commit change sets | 1 |
| TestFileMetricsAggregator_Process_DeletedFiles | Deleted files excluded from metrics | 1 |
| TestFileMetricsAggregator_Process_Renames | File rename tracking and metric merging | 1 |
| TestFileMetricsAggregator_Process_Renames_ReverseOrder | Rename handling with newest-first history | 1 |
| TestApplyBugfixCounts / WithRenames | Applying bugfix counts to file metrics | 2 |
| TestMergeMetrics_BugfixCount | Bugfix count merging | 1 |

### 3. `internal/aggregation/commit_metrics_test.go` - Commit Metrics

| Test Function | Purpose | Cases |
|---------------|---------|-------|
| TestCommitMetrics_TotalChurn | Churn calculation (added + deleted) | 3 |
| TestExtractPathComponents | Path parsing into directory and subsystem components | 6 |
| TestTruncateMessage | Message truncation to 100 chars (including LF/CRLF) | 6 |

### 4. `internal/bugfix/detector_test.go` - Bugfix Detection

| Test Function | Purpose | Cases |
|---------------|---------|-------|
| TestNewDetector_ValidPatterns | Detector creation with valid patterns | 1 |
| TestNewDetector_InvalidPattern | Error handling for invalid regex | 1 |
| TestNewDetector_EmptyPatterns | Empty pattern list | 1 |
| TestNewDetector_SkipsBlankPatterns | Blank patterns filtered out | 1 |
| TestIsBugfix | Bugfix message detection (fix, fixed, bug, hotfix, case insensitivity) | 11 |
| TestIsBugfix_NoPatterns | Behavior with no patterns configured | 1 |
| TestDetect | Complete detection workflow: counts, file bugfix counts, deleted files | 1 |
| TestDetect_NoPatterns / EmptyChangeSets | Edge cases in detection | 2 |
| TestDetect_MultiplePatterns | Varying pattern counts | 3 |

### 5. `internal/burst/sliding_window_test.go` - Burst Detection

| Test Function | Purpose | Cases |
|---------------|---------|-------|
| TestNewCalculator | Calculator creation with various window sizes | 3 |
| TestCalculateBurstScore_* | Burst score: empty, single, one window, spread, clusters, descending, unsorted, immutability | 8 |
| TestIsSortedAscending | Sort order detection (ascending) | 6 |
| TestIsSortedDescending | Sort order detection (descending) | 6 |
| TestReverse | Slice reversal | 3 |

### 6. `internal/coupling/analyzer_test.go` - Coupling Analysis

| Test Function | Purpose | Cases |
|---------------|---------|-------|
| TestNewFilePair_ConsistentOrdering | File pair ordering consistency | 3 |
| TestNewFilePair_Symmetry | Pair symmetry (A,B == B,A) | 3 |
| TestAnalyzer_Analyze_* | Empty input, single-file commits, perfect/partial coupling, min co-commits/Jaccard filters, max files filter, deleted files, top pairs limit, sorting | 10 |

### 7. `internal/entropy/shannon_test.go` - Entropy

| Test Function | Purpose | Cases |
|---------------|---------|-------|
| TestCalculateCommitEntropy_* | Shannon entropy: empty, single file, uniform/skewed distribution, zero churn, bounded range | 12+ |

### 8. `internal/git/` - Git Interface (7 files)

**diff_test.go**

| Test Function | Purpose | Cases |
|---------------|---------|-------|
| TestParseDiffSpec_* | Diff spec parsing (three-dot, two-dot, empty head/base, no dots, empty) | 6 |
| TestParseDiffNameStatus | Diff name-status output parsing (M/A/D, renames, empty) | 3 |
| TestReadDiff_Integration | Integration test with temporary git repository | 1 |

**mock_reader_test.go**

| Test Function | Purpose | Cases |
|---------------|---------|-------|
| TestMockHistoryReader_ReadChanges | Mock reader returns/errors | 2 |
| TestMockHistoryReader_ImplementsInterface | Interface compliance check | 1 |

**models_test.go**

| Test Function | Purpose | Cases |
|---------------|---------|-------|
| TestAuthorInfo_ContributorKey | Email normalization | 4 |
| TestFileChange_Churn | Churn calculation | 5 |
| TestChangeKind_String | Change kind string representation | 5 |

**reader_bench_test.go**

6 benchmark functions testing `ReadChanges` performance with various configurations (full/paths-only detail, rename detection, include/exclude filters, time window early termination).

**reader_branch_test.go**

| Test Function | Purpose | Cases |
|---------------|---------|-------|
| TestHistoryReader_ReadChanges_RespectsBranch | Multi-branch repo with branch-specific commit filtering | 1 |

**reader_filters_test.go**

| Test Function | Purpose | Cases |
|---------------|---------|-------|
| TestHistoryReader_matchesFilters_InvalidPatternsReturnError | Error handling for invalid glob patterns | 2 |

**reader_gitcli_test.go**

| Test Function | Purpose | Cases |
|---------------|---------|-------|
| TestParseGitRawAndNumstat_RenameAndModify | Parsing git raw+numstat output for modified and renamed files | 1 |
| TestKindFromGitStatus | Git status code mapping (A/M/D/R100) | 4 |

### 9. `internal/output/` - Output Formats (3 files)

**ci_test.go**

| Test Function | Purpose | Cases |
|---------------|---------|-------|
| TestCIFileWriter_Write | CI/NDJSON output format (1 summary + 3 files) | 1 |
| TestCIFileWriter_RiskLevelClassification | Risk level classification in output | 1 |
| TestCIFileWriter_TopOption | Top limit in output | 1 |

**formatter_test.go**

| Test Function | Purpose | Cases |
|---------------|---------|-------|
| TestNewFileReportWriter | Writer factory for Console/JSON/CSV/Markdown/Unknown/Empty | 6 |
| TestNewCommitReportWriter | Commit report writer factory | 5 |
| TestNewCouplingReportWriter | Coupling report writer factory | 5 |

**helpers_test.go**

| Test Function | Purpose | Cases |
|---------------|---------|-------|
| TestTruncateMessage_Output | Message truncation | 4 |
| TestGetRiskLevelEmoji | Emoji assignment for risk levels | 5 |
| TestEscapeMarkdown | Markdown character escaping | 7 |

### 10. `internal/scoring/` - Scoring Algorithms (3 files)

**file_scorer_test.go**

| Test Function | Purpose | Cases |
|---------------|---------|-------|
| TestFromMetrics_Empty / Multiple | Scoring context creation | 2 |
| TestFileScorer_ScoreAndRank_* | Empty input, ordering, explain breakdown, bugfix effect, bugfix zero, recency effect | 6 |

**commit_scorer_test.go**

| Test Function | Purpose | Cases |
|---------------|---------|-------|
| TestCommitContextFromMetrics_Empty / Multiple | Context creation | 2 |
| TestCommitScorer_ScoreAndRank_* | Empty input, ordering, explain breakdown, score bounds | 4 |
| TestFilterByRiskLevel | Filtering by risk level | 5 |

**normalization_test.go**

| Test Function | Purpose | Cases |
|---------------|---------|-------|
| TestMinMax_Range | Range calculation | 5 |
| TestMinMax_IsSingleValue | Single value detection | 6 |
| TestClamp | Value clamping to [0,1] | 7 |
| TestNormLog | Logarithmic normalization | 7+ |
| TestNormLog_Monotonicity | Monotonic increase | 1 |
| TestNormMinMax | Linear normalization | 7 |
| TestRecencyDecay | Exponential decay with half-life | 7+ |
| TestRecencyDecay_MonotonicDecrease | Monotonic decrease | 1 |

### 11. `testhelpers_test.go` - Root-Level Test Utilities

| Helper Function | Purpose |
|-----------------|---------|
| createTestRepo | Creates a temporary git repository via git CLI |
| addCommitToRepo | Adds commits with custom timestamps using `GIT_AUTHOR_DATE` / `GIT_COMMITTER_DATE` |
| suppressOutput / discardOutput | Suppresses stdout during test execution |
| runGit | Runs git commands in a specified directory |
