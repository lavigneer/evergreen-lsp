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
