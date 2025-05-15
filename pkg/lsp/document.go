package lsp

import (
	"context"
	"encoding/json"

	"github.com/a-h/templ/lsp/protocol"
	"github.com/lavigneer/evergreen-lsp/pkg/lint"
	"github.com/sourcegraph/jsonrpc2"
)

func (h *Handler) handleTextDocumentDidChange(ctx context.Context, req *jsonrpc2.Request) error {
	var params protocol.DidChangeTextDocumentParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return err
	}
	if res, ok := h.config.FindProjDoc(params.TextDocument.URI); ok {
		_, err := res.Project.UpdateDocument(ctx, params.TextDocument, params.ContentChanges[0])
		return err
	}
	return nil
}

func (h *Handler) handleTextDocumentDidOpen(ctx context.Context, req *jsonrpc2.Request) error {
	var params protocol.DidOpenTextDocumentParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return err
	}
	err := h.notifyDiagnostics(ctx, params.TextDocument.URI)
	return err
}

func (h *Handler) notifyDiagnostics(ctx context.Context, docURI protocol.DocumentURI) error {
	if res, ok := h.config.FindProjDoc(docURI); ok {
		lintExecutor := lint.New(res.Project, h.config.Lint)
		diagnostics, err := lintExecutor.LintDocument(res.Document.URI)
		if err != nil {
			return err
		}
		err = h.conn.Notify(ctx, protocol.MethodTextDocumentPublishDiagnostics, protocol.PublishDiagnosticsParams{
			URI: res.Document.URI,
			//nolint:gosec
			Version:     uint32(res.Document.Version),
			Diagnostics: diagnostics,
		})
		return err
	}
	return nil
}

func (h *Handler) handleTextDocumentDidSave(ctx context.Context, req *jsonrpc2.Request) error {
	var params protocol.DidSaveTextDocumentParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return err
	}
	err := h.notifyDiagnostics(ctx, params.TextDocument.URI)
	return err
}
