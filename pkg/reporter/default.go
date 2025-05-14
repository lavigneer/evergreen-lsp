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

type Default struct{}

func (d *Default) ReportDiagnostics(ctx context.Context, diagnostics lint.ExecutorDiagnostics) {
	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', 0)
	totalWarnings := 0
	totalErrors := 0
	for f, ds := range diagnostics {
		if len(ds) == 0 {
			continue
		}
		fmt.Println(string(f.URI))
		for _, d := range ds {
			switch d.Severity {
			case protocol.DiagnosticSeverityWarning:
				totalWarnings++
			case protocol.DiagnosticSeverityError:
				totalErrors++
			}
			fmt.Fprintf(writer, "\t%d:%d\t%s\t\t%s\t%s\n", d.Range.Start.Line, d.Range.Start.Character, strings.ToLower(d.Severity.String()), d.Message, d.Source)
		}
		writer.Flush()
		fmt.Println()
	}
	fmt.Printf("%d problems (%d errors, %d warnings)\n\n", totalWarnings+totalErrors, totalErrors, totalWarnings)
}
