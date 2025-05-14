package lsp

import (
	"context"
	"encoding/json"

	"github.com/a-h/templ/lsp/protocol"
	"github.com/sourcegraph/jsonrpc2"
)

func (h *Handler) handleTextDocumentDidChange(ctx context.Context, req *jsonrpc2.Request) error {
	var params protocol.DidChangeTextDocumentParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return err
	}
	err := h.workspace.UpdateDocument(ctx, params.TextDocument, params.ContentChanges[0])
	return err
}

func (h *Handler) handleTextDocumentDidOpen(ctx context.Context, req *jsonrpc2.Request) error {
	var params protocol.DidOpenTextDocumentParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return err
	}
	err := h.workspace.AddDocument(ctx, params.TextDocument)
	return err
}

func (h *Handler) handleTextDocumentDidSave(ctx context.Context, req *jsonrpc2.Request) error {
	var params protocol.DidSaveTextDocumentParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return err
	}
	doc, ok := h.workspace.TextDocuments[params.TextDocument.URI]
	if !ok {
		return ErrDocumentNotFound
	}
	err := h.conn.Notify(ctx, protocol.MethodTextDocumentPublishDiagnostics, protocol.PublishDiagnosticsParams{
		URI: doc.URI,
		//nolint:gosec
		Version:     uint32(doc.Version),
		Diagnostics: doc.Diagnostics,
	})
	return err
}
