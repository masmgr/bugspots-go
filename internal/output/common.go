package output

import (
	"io"
	"os"
	"time"
)

const (
	reportDateLayout     = "2006-01-02"
	reportDateTimeLayout = "2006-01-02T15:04:05"
)

func limitTop[T any](items []T, top int) []T {
	if top <= 0 || top >= len(items) {
		return items
	}
	return items[:top]
}

func dateRangeLabelAndValue(since *time.Time, until time.Time) (string, string) {
	if since != nil {
		return "Period", since.Format(reportDateLayout) + " to " + until.Format(reportDateLayout)
	}
	return "Until", until.Format(reportDateLayout)
}

func formatSinceDate(since *time.Time) *string {
	if since == nil {
		return nil
	}
	formatted := since.Format(reportDateLayout)
	return &formatted
}

func openOutputWriter(outputPath string) (io.Writer, *os.File, error) {
	if outputPath == "" {
		return os.Stdout, nil, nil
	}
	file, err := os.Create(outputPath)
	if err != nil {
		return nil, nil, err
	}
	return file, file, nil
}
