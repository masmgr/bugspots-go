package coupling

import (
	"sort"
	"strings"

	"github.com/masmgr/bugspots-go/config"
	"github.com/masmgr/bugspots-go/internal/git"
)

// FilePair represents a pair of files for coupling analysis.
type FilePair struct {
	FileA string
	FileB string
}

// NewFilePair creates a new file pair with consistent ordering.
func NewFilePair(a, b string) FilePair {
	// Ensure consistent ordering (lexicographically smaller first)
	if strings.ToLower(a) > strings.ToLower(b) {
		a, b = b, a
	}
	return FilePair{FileA: a, FileB: b}
}

// ChangeCoupling represents the coupling metrics between two files.
type ChangeCoupling struct {
	FileA              string
	FileB              string
	CoCommitCount      int     // Number of times both files were changed together
	FileACommitCount   int     // Total commits touching FileA
	FileBCommitCount   int     // Total commits touching FileB
	JaccardCoefficient float64 // |A ∩ B| / |A ∪ B|
	Confidence         float64 // P(B|A) = CoCommitCount / FileACommitCount
	Lift               float64 // P(A,B) / (P(A) × P(B))
}

// CouplingAnalysisResult holds the results of coupling analysis.
type CouplingAnalysisResult struct {
	Couplings    []ChangeCoupling
	TotalCommits int
	TotalFiles   int
	TotalPairs   int
}

// Analyzer analyzes change coupling between files based on co-commit patterns.
type Analyzer struct {
	options config.CouplingConfig
}

// NewAnalyzer creates a new coupling analyzer.
func NewAnalyzer(options config.CouplingConfig) *Analyzer {
	return &Analyzer{options: options}
}

// Analyze performs coupling analysis on commit change sets.
func (a *Analyzer) Analyze(changeSets []git.CommitChangeSet) CouplingAnalysisResult {
	fileCommitCounts := make(map[string]int)
	pairCoCommitCounts := make(map[FilePair]int)
	totalCommits := 0

	for _, changeSet := range changeSets {
		totalCommits++

		// Get unique file paths from this commit (excluding deleted files)
		seenFiles := make(map[string]struct{})
		var filesForPairs []string
		uniqueFileCount := 0

		for _, change := range changeSet.Changes {
			if change.Kind == git.ChangeKindDeleted {
				continue
			}

			path := strings.ToLower(change.Path)
			if _, seen := seenFiles[path]; seen {
				continue
			}
			seenFiles[path] = struct{}{}

			uniqueFileCount++
			fileCommitCounts[path]++

			// Only keep files for pairs if within limit
			if uniqueFileCount <= a.options.MaxFilesPerCommit {
				filesForPairs = append(filesForPairs, path)
			}
		}

		// Skip commits with too many files (likely refactoring or merge commits)
		// or with less than 2 files (no pairs possible)
		if uniqueFileCount < 2 || uniqueFileCount > a.options.MaxFilesPerCommit {
			continue
		}

		// Update pair co-commit counts
		for i := 0; i < len(filesForPairs)-1; i++ {
			for j := i + 1; j < len(filesForPairs); j++ {
				pair := NewFilePair(filesForPairs[i], filesForPairs[j])
				pairCoCommitCounts[pair]++
			}
		}
	}

	// Calculate coupling metrics and filter
	var couplings []ChangeCoupling

	for pair, coCommitCount := range pairCoCommitCounts {
		// Filter by minimum co-commits
		if coCommitCount < a.options.MinCoCommits {
			continue
		}

		commitsA := fileCommitCounts[pair.FileA]
		commitsB := fileCommitCounts[pair.FileB]

		// Jaccard coefficient: |A ∩ B| / |A ∪ B|
		union := commitsA + commitsB - coCommitCount
		jaccard := float64(coCommitCount) / float64(union)

		// Filter by minimum Jaccard threshold
		if jaccard < a.options.MinJaccardThreshold {
			continue
		}

		// Association rule metrics
		supportA := float64(commitsA) / float64(totalCommits)
		supportB := float64(commitsB) / float64(totalCommits)
		supportAB := float64(coCommitCount) / float64(totalCommits)

		// Confidence: P(B|A) = P(A,B) / P(A) = CoCommitCount / FileACommitCount
		confidence := float64(coCommitCount) / float64(commitsA)

		// Lift: P(A,B) / (P(A) × P(B))
		lift := supportAB / (supportA * supportB)

		couplings = append(couplings, ChangeCoupling{
			FileA:              pair.FileA,
			FileB:              pair.FileB,
			CoCommitCount:      coCommitCount,
			FileACommitCount:   commitsA,
			FileBCommitCount:   commitsB,
			JaccardCoefficient: jaccard,
			Confidence:         confidence,
			Lift:               lift,
		})
	}

	// Sort by Jaccard coefficient descending
	sort.Slice(couplings, func(i, j int) bool {
		return couplings[i].JaccardCoefficient > couplings[j].JaccardCoefficient
	})

	// Return top N pairs
	if len(couplings) > a.options.TopPairs {
		couplings = couplings[:a.options.TopPairs]
	}

	return CouplingAnalysisResult{
		Couplings:    couplings,
		TotalCommits: totalCommits,
		TotalFiles:   len(fileCommitCounts),
		TotalPairs:   len(pairCoCommitCounts),
	}
}
