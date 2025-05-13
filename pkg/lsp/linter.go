package lsp

import (
	"context"
	"log/slog"

	"github.com/a-h/templ/lsp/protocol"
)

func (h *Handler) linter() {
	for {
		uri, ok := <-h.request
		if !ok {
			break
		}

		diagnostics, err := h.lint(uri)
		if err != nil {
			slog.Error("Failed to lint", "error", err)
			continue
		}

		if err := h.conn.Notify(
			context.Background(),
			"textDocument/publishDiagnostics",
			&protocol.PublishDiagnosticsParams{
				URI:         uri,
				Diagnostics: diagnostics,
			}); err != nil {
		}
	}
}

func (h *Handler) lint(uri protocol.DocumentURI) ([]protocol.Diagnostic, error) {
	diagnostics := []protocol.Diagnostic{}
	// doc, ok := h.textDocuments[uri]
	// if !ok {
	// 	return diagnostics, nil
	// }
	return diagnostics, nil
}
