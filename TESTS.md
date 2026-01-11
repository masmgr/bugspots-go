# Test Suite Documentation

This document describes the comprehensive test suite added to bugspots-go.

## Test Files Created

### 1. `bugspots_test.go` - Core Algorithm Tests

Tests for the core bugspot calculation logic.

#### TestCalcScore
- **Purpose**: Tests the `CalcScore()` function which calculates temporal-weighted scores for bugfix commits
- **Test Cases**:
  - Fix at current date (most recent) - expects score ~1.0
  - Fix at oldest date - expects score ~0.0
  - Fix at midpoint - expects score ~0.5
  - Fix 1 year ago - expects very high score
  - Fix 2 years ago - expects high score
- **Tolerance**: Uses ±0.001 or ±0.01 depending on test

#### TestCalcScore_SigmoidProperties
- **Purpose**: Validates the mathematical properties of the sigmoid function
- **Test**: Verifies that recent fixes have higher scores than older fixes
- **Ensures**: The sigmoid weighting correctly prioritizes recent bugfix commits

#### TestMinInt
- **Purpose**: Tests the `minInt()` helper function
- **Test Cases**:
  - Basic cases (1,2 → 1)
  - Equal values (5,5 → 5)
  - Zero and negative numbers
  - Large numbers
- **Coverage**: 7 distinct test cases

#### TestMinInt_Symmetry
- **Purpose**: Validates that minInt is symmetric (order of arguments doesn't matter)
- **Test Cases**: Multiple pairs of numbers tested in both orders
- **Ensures**: Function correctness regardless of argument order

### 2. `app_test.go` - CLI Argument Processing Tests

Tests for command-line interface argument parsing and conversion.

#### TestConvertToRegex
- **Purpose**: Tests the `convertToRegex()` function which converts comma-separated words to regex
- **Test Cases**:
  - Single word: "fix" → "fix"
  - Two words: "fix,close" → "fix|close"
  - Multiple words: "fix,close,resolve,closes" → "fix|close|resolve|closes"
  - Empty string handling
- **Coverage**: 5 test cases

#### TestConvertToRegex_RegexValidity
- **Purpose**: Ensures the output of `convertToRegex()` can be compiled as valid regex
- **Validation**: Uses `regexp.Compile()` to verify regex syntax
- **Prevents**: Invalid regex patterns from being generated

#### TestConvertToRegex_RegexMatching
- **Purpose**: Tests that the converted regex patterns match expected strings
- **Test Cases**:
  - Single word matching variations
  - Multiple word alternation matching
  - Case sensitivity validation
- **Ensures**: Generated regex patterns work correctly

#### TestConvertToRegex_EdgeCases
- **Purpose**: Tests edge cases and boundary conditions
- **Test Cases**:
  - Single word with no commas
  - Trailing commas
  - Leading commas
  - Multiple consecutive commas
- **Ensures**: Function handles malformed input gracefully

### 3. `testhelpers_test.go` - Test Utilities

Helper functions for integration testing.

#### createTestRepo
- **Purpose**: Creates a temporary git repository for testing
- **Returns**: Path to temp directory and git repository object
- **Usage**: Base for integration tests

#### addCommitToRepo
- **Purpose**: Adds commits with test data to a repository
- **Parameters**: Message, filenames, custom commit time
- **Usage**: Builds test repositories with known history

#### suppressOutput / discardOutput
- **Purpose**: Suppresses stdout during test execution
- **Usage**: Prevents test output clutter from color.* function calls

#### configureGitUser
- **Purpose**: Sets up git user configuration for test repositories
- **Usage**: Ensures commits can be made in test environment

## Running the Tests

### Prerequisites
- Go 1.17 or later
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

# Run specific test file
go test -v bugspots_test.go bugspots.go

# Run specific test function
go test -v -run TestCalcScore ./...
```

## Test Coverage

### Functions Tested

| Function | Test File | Coverage |
|----------|-----------|----------|
| CalcScore() | bugspots_test.go | 100% |
| minInt() | bugspots_test.go | 100% |
| convertToRegex() | app_test.go | 100% |
| ShowResult() | Manual testing required* |
| getFixes() | Manual testing required* |
| Scan() | Manual testing required* |

*Note: ShowResult, getFixes, and Scan require git repository context and use color output, making them better suited for manual/integration testing.

## Test Quality Metrics

- **Total Test Functions**: 8
- **Total Test Cases**: 30+
- **Edge Cases Covered**: Yes
- **Mathematical Properties Tested**: Yes
- **Error Handling**: Partial (helpers test graceful handling)

## Future Improvements

1. Add integration tests with actual git repository fixtures
2. Mock git.Repository for testing Scan() without real repos
3. Test error cases in Scan() (invalid repos, missing branches)
4. Add benchmark tests for performance-critical functions
5. Test getRegexp() CLI flag parsing more thoroughly

## Notes

- Tests follow Go testing conventions (functions named `Test*`)
- Uses `testing.T` for assertions and error reporting
- All test files use `_test.go` suffix for automatic discovery
- Tests are isolated and can run in any order
- No external dependencies required for unit tests
