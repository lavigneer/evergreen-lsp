package reporter

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/a-h/templ/lsp/protocol"
	"github.com/lavigneer/evergreen-lsp/pkg/lint"
)

type Default struct {
	writer              *tabwriter.Writer
	totalByType         map[string]int
	totalWarnings       int
	totalErrors         int
	totalDiagnosticSets int
}

func (d *Default) Init() {
	d.totalByType = make(map[string]int)
	d.totalErrors = 0
	d.totalWarnings = 0
	d.totalDiagnosticSets = 0
	d.writer = tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', 0)
}

func (d *Default) ReportDiagnostics(ctx context.Context, diagnostics lint.ExecutorDiagnostics) {
	warningsCount := 0
	errorsCount := 0
	for f, diags := range diagnostics {
		if len(diags) == 0 {
			continue
		}
		fmt.Println(string(f.URI))
		for _, diag := range diags {
			d.totalByType[diag.Source]++
			switch diag.Severity {
			case protocol.DiagnosticSeverityWarning:
				warningsCount++
			case protocol.DiagnosticSeverityError:
				errorsCount++
			}
			fmt.Fprintf(d.writer, "\t%d:%d\t%s\t\t%s\t%s\n", diag.Range.Start.Line, diag.Range.Start.Character, strings.ToLower(diag.Severity.String()), diag.Message, diag.Source)
		}
		d.writer.Flush()
		fmt.Println()
	}
	fmt.Printf("%d problems (%d errors, %d warnings)\n\n", warningsCount+errorsCount, errorsCount, warningsCount)
	d.totalErrors += errorsCount
	d.totalWarnings += warningsCount
	d.totalDiagnosticSets++
}

func (d *Default) ReportSummary(ctx context.Context) {
	fmt.Printf("*** Summary ***:\n")
	d.writer.Write([]byte("\t[Rule]\t[Count]\n"))
	for t, c := range d.totalByType {
		fmt.Fprintf(d.writer, "\t%s\t%d\n", t, c)
	}
	d.writer.Flush()
	fmt.Println()
	fmt.Printf("%d problems across all %d project(s) (%d errors, %d warnings)\n\n", d.totalWarnings+d.totalErrors, d.totalDiagnosticSets, d.totalErrors, d.totalWarnings)
}
