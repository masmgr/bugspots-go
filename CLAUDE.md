# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

bugspots-go is a Go implementation of the [Bugspots bug prediction heuristic](http://google-engtools.blogspot.com/2011/12/bug-prediction-at-google.html) originally created by igrigorik. It analyzes Git repositories to identify files most likely to contain bugs based on historical fix commits.

## Build and Run Commands

### Build the project
```bash
go build -o bugspots-go .
```

### Run the tool
```bash
./bugspots-go [flags] /path/to/git/repo
```

### Common usage examples
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

### Run tests
```bash
go test ./...
```

### Lint code
```bash
go fmt ./...
golint ./...
```

## Architecture

The project has two main components:

### app.go - CLI Entry Point
- Defines the command-line interface using urfave/cli/v2
- Parses flags for branch selection, bugfix indicators (words or regex), and display options
- Converts word-based bugfix indicators into regex patterns
- Calls the main `Scan()` function with parsed parameters

### bugspots.go - Core Algorithm
The core algorithm works through these key functions:

1. **Scan()** - Main entry point that:
   - Opens the Git repository using go-git
   - Retrieves commit history from the past 3 years
   - Identifies bugfix commits using the provided regex pattern
   - Calculates hotspot scores for each modified file
   - Displays results

2. **getFixes()** - Identifies bugfix commits by:
   - Filtering commits matching the bugfix regex
   - Computing diffs between commits and their parents
   - Extracting the list of modified files for each fix commit
   - Returns a list of Fix structs containing message, date, and affected files

3. **CalcScore()** - Weights each bugfix commit by recency using a sigmoid function:
   - Recent fixes have higher weight (approaching 1)
   - Older fixes have lower weight (approaching 0)
   - Provides temporal scoring: newer bugs in a file increase its hotspot score more than older bugs
   - Normalizes time relative to the analysis date and repository age

4. **ShowResult()** - Formats and displays:
   - All identified bugfix commits sorted by recency
   - Top 100 hotspot files ranked by cumulative bugfix score

## Key Data Structures

- **Fix** - Represents a detected bugfix commit (message, timestamp, list of modified files)
- **Spot** - Represents a file's bugspot ranking (filename, calculated score)

## Dependencies

- `github.com/urfave/cli/v2` - CLI framework
- `github.com/go-git/go-git/v5` - Git repository interaction
- `github.com/fatih/color` - Colored output
