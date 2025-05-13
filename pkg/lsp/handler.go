package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/a-h/templ/lsp/protocol"
	"github.com/sourcegraph/jsonrpc2"
)

type Handler struct {
	conn             *jsonrpc2.Conn
	request          chan protocol.DocumentURI
	workspaceFolders []protocol.WorkspaceFolder
	workspace        *Workspace
}

//nolint:ireturn
func NewHandler() jsonrpc2.Handler {
	handler := &Handler{
		request: make(chan protocol.DocumentURI),
	}
	// go handler.linter()
	return jsonrpc2.HandlerWithError(handler.Handle)
}

// Handle implements jsonrpc2.Handler.
//
//nolint:nilnil
func (h *Handler) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (any, error) {
	slog.Debug("Handling request", "request", req)
	switch req.Method {
	case protocol.MethodInitialize:
		return h.handleInitialize(ctx, conn, req)
	case protocol.MethodInitialized:
		return nil, nil
	case protocol.MethodTextDocumentDidOpen:
		return nil, h.handleTextDocumentDidOpen(ctx, req)
	case protocol.MethodTextDocumentDidChange:
		return nil, h.handleTextDocumentDidChange(ctx, req)
	case protocol.MethodTextDocumentCompletion:
		return h.handleTextDocumentCompletion(ctx, req)
	case protocol.MethodTextDocumentDefinition:
		return h.handleTextDocumentDefinition(ctx, req)
	case protocol.MethodTextDocumentHover:
		return h.handleTextDocumentHover(ctx, req)
	case protocol.MethodTextDocumentReferences:
		return h.handleTextDocumentReferences(ctx, req)
	}
	return nil, &jsonrpc2.Error{
		Code:    jsonrpc2.CodeMethodNotFound,
		Message: fmt.Sprintf("method not supported: %s", req.Method),
	}
}

func (h *Handler) handleInitialize(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (any, error) {
	var params protocol.InitializeParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}

	h.workspaceFolders = params.WorkspaceFolders
	h.conn = conn

	slog.Debug("Initialized", "workspaceFolders", params.WorkspaceFolders)

	var workspacePath string
	for _, f := range params.WorkspaceFolders {
		fPath := uriToPath(f.URI)
		dirEntries, err := os.ReadDir(fPath)
		if err != nil {
			continue
		}
		for _, d := range dirEntries {
			if d.Name() == "evergreen.yml" || d.Name() == "evergreen.yaml" {
				workspacePath = filepath.Join(fPath, d.Name())
				break
			}
		}
		if workspacePath != "" {
			break
		}
	}
	h.workspace = NewWorkspace(workspacePath)
	err := h.workspace.Init(ctx)
	if err != nil {
		return nil, err
	}

	return protocol.InitializeResult{
		Capabilities: protocol.ServerCapabilities{
			TextDocumentSync: protocol.TextDocumentSyncOptions{
				Change:    protocol.TextDocumentSyncKindFull,
				OpenClose: true,
				Save:      &protocol.SaveOptions{IncludeText: true},
			},
			CompletionProvider: &protocol.CompletionOptions{
				TriggerCharacters: strings.Split("qwertyuiopasdfghjklzxcvbnm. ", ""),
			},
			DefinitionProvider: &protocol.DefinitionOptions{},
			HoverProvider:      &protocol.HoverOptions{},
			ReferencesProvider: &protocol.ReferenceOptions{},
		},
	}, nil
}
