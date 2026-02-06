# Test Suite Documentation

This document describes the comprehensive test suite for bugspots-go.

## Overview

- **Test files**: 23
- **Test functions**: 91+
- **Test cases/subtests**: 280+
- **Benchmark functions**: 6
- **Coverage areas**: All packages (cmd, config, internal/*)

## Test Files by Package

### 1. `cmd/scan_test.go` - CLI Argument Processing Tests

Tests for the `convertToRegex()` function in the scan command.

#### TestConvertToRegex
- **Purpose**: Tests conversion of comma-separated words to regex alternation
- **Test Cases**: Single word, two words, multiple words, words with spaces, empty string
- **Coverage**: 5 test cases

#### TestConvertToRegex_RegexValidity
- **Purpose**: Ensures output compiles as valid regex via `regexp.Compile()`
- **Coverage**: 3 test cases

#### TestConvertToRegex_RegexMatching
- **Purpose**: Tests that converted regex patterns match expected strings
- **Test Cases**: Single word matching, multiple word alternation, case sensitivity
- **Coverage**: 3 test cases

#### TestConvertToRegex_EdgeCases
- **Purpose**: Tests boundary conditions (trailing/leading/multiple commas, single word)
- **Coverage**: 4 test cases

### 2. `config/config_test.go` - Configuration Tests

#### TestRiskThresholds_Classify
- **Purpose**: Tests risk level classification (high/medium/low) at boundary values
- **Coverage**: 9 subtests

#### TestDefaultConfig
- **Purpose**: Validates all default configuration values (scoring weights, thresholds, etc.)
- **Coverage**: 15 assertions

#### TestDefaultConfig_WeightsSum
- **Purpose**: Verifies file and commit scoring weights each sum to 1.0
- **Coverage**: 2 validations with tolerance

### 3. `internal/aggregation/file_metrics_test.go` - File Metrics Tests

#### TestNewFileMetrics
- Validates FileMetrics initialization (6 assertions)

#### TestFileMetrics_ChurnTotal
- Tests total churn calculation (added + deleted lines)

#### TestFileMetrics_ContributorCount
- Tests counting unique contributors

#### TestFileMetrics_OwnershipRatio
- Tests ownership ratio for single/multiple contributors and no-commit case
- **Coverage**: 4 subtests

#### TestFileMetrics_OwnershipRatio_Caching
- Tests caching mechanism and cache invalidation on AddCommit

#### TestFileMetrics_AddCommit
- Tests updating metrics when adding commits

#### TestFileMetrics_AddCommit_CommitTimesDisabled
- Tests commit time collection can be disabled

#### TestFileMetricsAggregator_Process
- Tests aggregating file metrics from multiple commit change sets

#### TestFileMetricsAggregator_Process_DeletedFiles
- Tests that deleted files are excluded from metrics

#### TestFileMetricsAggregator_Process_Renames
- Tests file rename tracking and metric merging

#### TestFileMetricsAggregator_Process_Renames_ReverseOrder
- Tests rename handling when history is processed newest-first

#### TestApplyBugfixCounts / TestApplyBugfixCounts_WithRenames
- Tests applying bugfix counts to file metrics (with and without renames)

#### TestMergeMetrics_BugfixCount
- Tests bugfix count merging

### 4. `internal/aggregation/commit_metrics_test.go` - Commit Metrics Tests

#### TestCommitMetrics_TotalChurn
- Tests churn calculation (added + deleted). 3 subtests

#### TestExtractPathComponents
- Tests path parsing into directory and subsystem components. 6 subtests (including Windows paths)

#### TestTruncateMessage
- Tests message truncation to 100 chars. 6 subtests (including multi-line with LF/CRLF)

### 5. `internal/bugfix/detector_test.go` - Bugfix Detection Tests

#### TestNewDetector_ValidPatterns / InvalidPattern / EmptyPatterns / SkipsBlankPatterns
- Tests detector creation with various pattern inputs

#### TestIsBugfix
- Tests bugfix message detection (fix, fixed, bug, hotfix, case insensitivity, etc.)
- **Coverage**: 11 subtests

#### TestIsBugfix_NoPatterns
- Tests behavior when no patterns configured

#### TestDetect
- Tests complete detection workflow: total counts, file bugfix counts, deleted files

#### TestDetect_NoPatterns / EmptyChangeSets
- Tests edge cases in detection

#### TestDetect_MultiplePatterns
- Tests with varying pattern counts (3 scenarios)

### 6. `internal/burst/sliding_window_test.go` - Burst Detection Tests

#### TestNewCalculator
- Tests calculator creation with various window sizes. 3 subtests

#### TestCalculateBurstScore_*
- Tests burst score calculation for: empty input, single commit, all in one window, spread across windows, two clusters, descending/unsorted order, immutability
- **Coverage**: 8 test functions

#### TestIsSortedAscending / TestIsSortedDescending
- Tests sort order detection. 6 subtests each

#### TestReverse
- Tests slice reversal. 3 subtests

### 7. `internal/coupling/analyzer_test.go` - Coupling Analysis Tests

#### TestNewFilePair_ConsistentOrdering
- Tests file pair ordering consistency. 3 subtests

#### TestNewFilePair_Symmetry
- Tests pair symmetry (A,B == B,A). 3 pairs tested

#### TestAnalyzer_Analyze_*
- Tests coupling analysis for: empty input, single-file commits, perfect coupling (Jaccard=1.0), partial coupling, min co-commits filter, min Jaccard filter, max files per commit filter, deleted files exclusion, top pairs limit, sorting by Jaccard
- **Coverage**: 10 test functions

### 8. `internal/entropy/shannon_test.go` - Entropy Tests

#### TestCalculateCommitEntropy_*
- Tests Shannon entropy calculation for: empty changes, single file, uniform distribution, skewed distribution, zero churn, bounded range
- **Coverage**: 6 test functions with 12+ subtests

### 9. `internal/git/` - Git Interface Tests (7 files)

#### diff_test.go
- **TestParseDiffSpec_*** - Tests diff spec parsing (three-dot, two-dot, empty head/base, no dots, empty). 6 tests
- **TestParseDiffNameStatus** - Tests diff name-status output parsing (M/A/D status, renames, empty). 3 tests
- **TestReadDiff_Integration** - Integration test with temporary git repository

#### mock_reader_test.go
- **TestMockHistoryReader_ReadChanges** - Tests mock reader returns/errors. 2 subtests
- **TestMockHistoryReader_ImplementsInterface** - Interface compliance check

#### models_test.go
- **TestAuthorInfo_ContributorKey** - Tests email normalization. 4 subtests
- **TestFileChange_Churn** - Tests churn calculation. 5 subtests
- **TestChangeKind_String** - Tests change kind string representation. 5 subtests

#### reader_bench_test.go
- **Benchmarks**: 6 benchmark functions testing `ReadChanges` performance with various configurations (full/paths-only detail, rename detection, include/exclude filters, time window early termination)

#### reader_branch_test.go
- **TestHistoryReader_ReadChanges_RespectsBranch** - Integration test: creates multi-branch repo, verifies branch-specific commit filtering

#### reader_filters_test.go
- **TestHistoryReader_matchesFilters_InvalidPatternsReturnError** - Tests error handling for invalid include/exclude glob patterns. 2 subtests

#### reader_gitcli_test.go
- **TestParseGitRawAndNumstat_RenameAndModify** - Tests parsing git raw+numstat output for modified and renamed files
- **TestKindFromGitStatus** - Tests git status code mapping (A/M/D/R100). 4 subtests

### 10. `internal/output/` - Output Format Tests (3 files)

#### ci_test.go
- **TestCIFileWriter_Write** - Tests CI/NDJSON output format (4 lines: 1 summary + 3 files)
- **TestCIFileWriter_RiskLevelClassification** - Tests risk level classification in output
- **TestCIFileWriter_TopOption** - Tests Top limit in output

#### formatter_test.go
- **TestNewFileReportWriter** - Tests writer factory for Console/JSON/CSV/Markdown/Unknown/Empty. 6 subtests
- **TestNewCommitReportWriter** - Tests commit report writer factory. 5 subtests
- **TestNewCouplingReportWriter** - Tests coupling report writer factory. 5 subtests

#### helpers_test.go
- **TestTruncateMessage_Output** - Tests message truncation. 4 subtests
- **TestGetRiskLevelEmoji** - Tests emoji assignment for risk levels. 5 subtests
- **TestEscapeMarkdown** - Tests markdown character escaping. 7 subtests

### 11. `internal/scoring/` - Scoring Algorithm Tests (4 files)

#### legacy_test.go
- **TestLegacySigmoidScore** - Tests sigmoid scoring at different time positions (t=0, t=1, t≈0.667, t≈0.333). 4 subtests
- **TestLegacySigmoidScore_SigmoidProperties** - Tests recent fixes score higher than older fixes
- **TestCalculateLegacyHotspots** - Tests hotspot calculation from fix data
- **TestRankLegacyHotspots** - Tests ranking by score (descending)
- **TestRankLegacyHotspots_MaxSpots** - Tests maxSpots limit

#### file_scorer_test.go
- **TestFromMetrics_Empty / Multiple** - Tests scoring context creation
- **TestFileScorer_ScoreAndRank_*** - Tests: empty input, ordering, explain breakdown, bugfix effect, bugfix zero, recency effect
- **Coverage**: 8 test functions

#### commit_scorer_test.go
- **TestCommitContextFromMetrics_Empty / Multiple** - Tests context creation
- **TestCommitScorer_ScoreAndRank_*** - Tests: empty input, ordering, explain breakdown, score bounds
- **TestFilterByRiskLevel** - Tests filtering by risk level. 4 subtests + empty test

#### normalization_test.go
- **TestMinMax_Range** - Tests range calculation. 5 subtests
- **TestMinMax_IsSingleValue** - Tests single value detection. 6 subtests
- **TestClamp** - Tests value clamping to [0,1]. 7 subtests
- **TestNormLog** - Tests logarithmic normalization. 7+ subtests
- **TestNormLog_Monotonicity** - Tests monotonic increase
- **TestNormMinMax** - Tests linear normalization. 7 subtests
- **TestRecencyDecay** - Tests exponential decay with half-life. 7+ subtests
- **TestRecencyDecay_MonotonicDecrease** - Tests monotonic decrease

### 12. `testhelpers_test.go` - Root-Level Test Utilities

#### createTestRepo
- Creates a temporary git repository for testing using git CLI
- Returns the directory path

#### addCommitToRepo
- Adds commits with custom timestamps using `GIT_AUTHOR_DATE` / `GIT_COMMITTER_DATE`

#### suppressOutput / discardOutput
- Suppresses stdout during test execution

#### runGit
- Runs git commands in a specified directory

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
| cmd | scan_test.go | 4 |
| config | config_test.go | 3 |
| internal/aggregation | file_metrics_test.go, commit_metrics_test.go | 17 |
| internal/bugfix | detector_test.go | 10 |
| internal/burst | sliding_window_test.go | 13 |
| internal/coupling | analyzer_test.go | 12 |
| internal/entropy | shannon_test.go | 6 |
| internal/git | 7 test files | 16 + 6 benchmarks |
| internal/output | 3 test files | 9 |
| internal/scoring | 4 test files | 21 |
| (root) | testhelpers_test.go | 4 helpers |

## Notes

- Tests follow Go testing conventions (functions named `Test*`, benchmarks named `Benchmark*`)
- Uses `testing.T` for assertions and error reporting
- All test files use `_test.go` suffix for automatic discovery
- Tests are isolated and can run in any order
- Integration tests (git operations) use temporary directories via `t.TempDir()`
- No external dependencies required beyond `git` binary
- Git operations use `os/exec` (no go-git dependency)
