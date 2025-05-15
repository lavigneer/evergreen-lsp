package reporter

import (
	"context"
	"log/slog"

	"github.com/a-h/templ/lsp/protocol"
)

type Reporter interface {
	Init()
	ReportDiagnostics(ctx context.Context, diagnostics []protocol.Diagnostic)
	ReportSummary(ctx context.Context)
}

func diagnosticSeverityToLogLevel(s protocol.DiagnosticSeverity) slog.Level {
	switch s {
	case protocol.DiagnosticSeverityInformation:
		return slog.LevelInfo
	case protocol.DiagnosticSeverityWarning:
		return slog.LevelWarn
	case protocol.DiagnosticSeverityError:
		return slog.LevelError
	case protocol.DiagnosticSeverityHint:
		return slog.LevelInfo
	}
	return slog.LevelInfo
}
