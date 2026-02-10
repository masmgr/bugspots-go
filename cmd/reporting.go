package cmd

import (
	"github.com/urfave/cli/v2"

	"github.com/masmgr/bugspots-go/internal/output"
)

func writeFileReport(c *cli.Context, report *output.FileAnalysisReport) error {
	opts := OutputOptions(c)
	writer := output.NewFileReportWriter(opts.Format)
	return writer.Write(report, opts)
}

func writeCommitReport(c *cli.Context, report *output.CommitAnalysisReport) error {
	opts := OutputOptions(c)
	writer := output.NewCommitReportWriter(opts.Format)
	return writer.Write(report, opts)
}

func writeCouplingReport(c *cli.Context, report *output.CouplingAnalysisReport) error {
	opts := OutputOptions(c)
	writer := output.NewCouplingReportWriter(opts.Format)
	return writer.Write(report, opts)
}
