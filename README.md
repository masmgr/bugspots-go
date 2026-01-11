# bugspots-go

A Go implementation of the [Bugspots bug prediction heuristic](http://google-engtools.blogspot.com/2011/12/bug-prediction-at-google.html) originally created by [igrigorik](https://github.com/igrigorik/bugspots). This tool analyzes Git repositories to identify files most likely to contain bugs based on historical fix commits.

## Overview

Bugspots uses a weighted scoring algorithm to identify "hotspot" files in your codebase—files that have been frequently modified to fix bugs. The scoring algorithm weights recent fixes more heavily than older fixes, giving you a data-driven way to identify areas of your code that may need additional testing or refactoring.

## Installation

### Build from source

```bash
go build -o bugspots-go .
```

## Usage

```bash
./bugspots-go [flags] /path/to/git/repo
```

### Common Examples

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

## How It Works

Bugspots analyzes your Git history to find commits that fixed bugs, then calculates a "hotspot score" for each file based on:

1. **Frequency**: How many times a file has been modified in bugfix commits
2. **Recency**: More recent fixes are weighted higher than older fixes using a sigmoid function
3. **Repository Age**: Scoring is normalized relative to the repository's total history

The algorithm then displays:
- A list of all detected bugfix commits (sorted by recency)
- The top 100 hotspot files ranked by cumulative bugfix score

### Key Components

- **app.go**: CLI entry point that parses flags and command-line arguments
- **bugspots.go**: Core algorithm implementation

## Testing

```bash
go test ./...
```

## Code Style

```bash
go fmt ./...
golint ./...
```

## Dependencies

- `github.com/urfave/cli/v2` - Command-line interface framework
- `github.com/go-git/go-git/v5` - Git repository interaction
- `github.com/fatih/color` - Colored console output

## Related Projects

- [igrigorik/bugspots](https://github.com/igrigorik/bugspots) - Original Ruby implementation
- [Google Engineering Tools Blog](http://google-engtools.blogspot.com/2011/12/bug-prediction-at-google.html) - Original research

