package main

import (
	"fmt"
	"math"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type Fix struct {
	message string
	date    time.Time
	files   []string
}

type Spot struct {
	file  string
	score float64
}

func Scan(repo string, branch string, depth int, regex *regexp.Regexp) {
	color.Green("Scanning %v repo", repo)

	r, err := git.PlainOpen(repo)
	if err != nil {
		color.Red("Invalid Git repository - please run from or specify the full path to the root of the project.")
	}

	/*
		w, err := r.Worktree()
		CheckIfError(err)

		err = w.Checkout(&git.CheckoutOptions{Branch: plumbing.ReferenceName(branch)})
		CheckIfError(err)
	*/

	ref, err := r.Head()
	CheckIfError(err)

	// ... retrieves the commit history
	until := time.Now()
	since := until.AddDate(-3, 0, 0)
	cIter, err := r.Log(&git.LogOptions{From: ref.Hash(), Since: &since, Until: &until})
	CheckIfError(err)

	fixes := getFixes(cIter, regex)

	hotspots := map[string]float64{}
	for _, fix := range fixes {
		for _, file := range fix.files {

			if _, is_exists := hotspots[file]; !is_exists {
				hotspots[file] = 0
			}
			hotspots[file] += CalcScore(until, since, fix.date)
		}
	}

	ShowResult(fixes, hotspots)
}

func getFixes(cIter object.CommitIter, regex *regexp.Regexp) []Fix {
	var fixes []Fix

	err := cIter.ForEach(func(c *object.Commit) error {
		if regex != nil && len(regex.FindStringSubmatch(c.Message)) == 0 {
			return nil
		}

		// Skip commits without parents (initial commit)
		if c.NumParents() == 0 {
			return nil
		}

		tree, err := c.Tree()
		CheckIfError(err)

		parent, err2 := c.Parent(0)
		CheckIfError(err2)

		parentTree, err2 := parent.Tree()
		CheckIfError(err2)

		changes, err2 := object.DiffTree(parentTree, tree)
		CheckIfError(err2)

		var files []string
		for _, change := range changes {
			if change.From.Name != "" {
				files = append(files, change.From.Name)
			} else {
				files = append(files, change.To.Name)
			}
		}

		fixes = append(fixes, Fix{message: strings.Split(c.Message, "\n")[0], date: c.Committer.When, files: files})

		return nil
	})
	CheckIfError(err)
	return fixes
}

func CalcScore(currentDate time.Time, oldestDate time.Time, fixDate time.Time) float64 {
	/*
	* The timestamp used in the equation is normalized from 0 to 1, where
	* 0 is the earliest point in the code base, and 1 is now (where now is
	* when the algorithm was run). Note that the score changes over time
	* with this algorithm due to the moving normalization; it's not meant
	* to provide some objective score, only provide a means of comparison
	* between one file and another at any one point in time
	 */
	t := 1 - (float64(currentDate.Sub(fixDate).Seconds()) / currentDate.Sub(oldestDate).Seconds())
	return 1 / (1 + math.Exp((-12*t)+12))
}

func ShowResult(fixes []Fix, hotspots map[string]float64) {
	var spots []Spot
	for k, v := range hotspots {
		spots = append(spots, Spot{file: k, score: v})
	}

	sort.Slice(spots, func(i, j int) bool {
		return spots[i].score > spots[j].score
	})

	fmt.Print("\t")
	color.Yellow("Found %v bugfix commits, with %v hotspots:", len(fixes), len(spots))
	fmt.Println("")

	colorTitle := color.New(color.FgGreen).Add(color.Underline)

	fmt.Print("\t")
	colorTitle.Println("Fixes:")
	for _, fix := range fixes {
		fmt.Print("\t\t")
		var buff strings.Builder
		buff.WriteString("- ")
		buff.WriteString(fix.date.String())
		buff.WriteString(" ")
		buff.WriteString(fix.message)
		color.Yellow(buff.String())
	}

	fmt.Println("")
	fmt.Print("\t")
	colorTitle.Println("Hotspots:")

	max_num_spots := 100
	num_spots := minInt(len(spots), max_num_spots)

	colorSpot := color.New(color.FgRed)
	colorScore := color.New(color.FgYellow)

	for _, spot := range spots[:num_spots] {
		fmt.Print("\t\t")
		colorSpot.Print(color.RedString("%v", spot.file))
		colorScore.Println(color.YellowString(" - %.3f", spot.score))
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// CheckIfError should be used to naively panics if an error is not nil.
func CheckIfError(err error) {
	if err == nil {
		return
	}

	color.Red(fmt.Sprintf("error: %s", err))
	os.Exit(1)
}

// Info should be used to describe the example commands that are about to run.
func Info(format string, args ...interface{}) {
	color.Blue(fmt.Sprintf(format, args...))
}

func Warning(format string, args ...interface{}) {
	color.Cyan(fmt.Sprintf(format, args...))
}
