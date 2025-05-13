package lsp

import (
	"encoding/json"

	"github.com/a-h/templ/lsp/protocol"
	"github.com/sourcegraph/jsonrpc2"
)

func (h *Handler) handleTextDocumentDidChange(req *jsonrpc2.Request) error {
	var params protocol.DidChangeTextDocumentParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return err
	}
	doc, ok := h.textDocuments[params.TextDocument.URI]
	if !ok {
		panic("fix this")
	}
	if doc.Version > params.TextDocument.Version {
		panic("uh oh! Old version came later!")
	}
	doc.Version = params.TextDocument.Version
	doc.Text = params.ContentChanges[0].Text
	h.textDocuments[params.TextDocument.URI] = doc
	return nil
}

func (h *Handler) handleTextDocumentDidOpen(req *jsonrpc2.Request) error {
	var params protocol.DidOpenTextDocumentParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return err
	}
	h.textDocuments[params.TextDocument.URI] = params.TextDocument
	return nil
}
