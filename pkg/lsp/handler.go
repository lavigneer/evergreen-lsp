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
	"github.com/evergreen-ci/evergreen/agent/command"
	"github.com/evergreen-ci/evergreen/model"
	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"
	"github.com/sourcegraph/jsonrpc2"
)

type Handler struct {
	conn             *jsonrpc2.Conn
	request          chan protocol.DocumentURI
	workspaceFolders []protocol.WorkspaceFolder
	project          *model.Project
	rootYamlPath     string
	textDocuments    map[protocol.DocumentURI]protocol.TextDocumentItem
}

func NewHandler() jsonrpc2.Handler {
	handler := &Handler{
		request:       make(chan protocol.DocumentURI),
		textDocuments: make(map[protocol.DocumentURI]protocol.TextDocumentItem),
	}
	return jsonrpc2.HandlerWithError(handler.Handle)
}

// Handle implements jsonrpc2.Handler.
func (h *Handler) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (any, error) {
	slog.Debug("Handling request", "request", req)
	switch req.Method {
	case protocol.MethodInitialize:
		return h.handleInitialize(ctx, conn, req)
	case protocol.MethodInitialized:
		return nil, nil
	case protocol.MethodTextDocumentDidOpen:
		return h.handleTextDocumentDidOpen(ctx, conn, req)
	case protocol.MethodTextDocumentDidChange:
		return h.handleTextDocumentDidChange(ctx, conn, req)
	case protocol.MethodTextDocumentCompletion:
		return h.handleTextDocumentCompletion(ctx, conn, req)
	}
	return nil, &jsonrpc2.Error{
		Code:    jsonrpc2.CodeMethodNotFound,
		Message: fmt.Sprintf("method not supported: %s", req.Method),
	}
}

func (h *Handler) handleTextDocumentDidChange(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (any, error) {
	var params protocol.DidChangeTextDocumentParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
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
	return nil, nil
}

func (h *Handler) handleTextDocumentCompletion(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (any, error) {
	var params protocol.CompletionParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}
	doc, ok := h.textDocuments[params.TextDocument.URI]
	if !ok {
		return nil, nil
	}

	astFile, err := parser.ParseBytes([]byte(doc.Text), parser.ParseComments)
	if err != nil {
		return nil, err
	}
	// node := FindDeepestOverlappingNode(&astFile.Docs[0], int(params.Position.Line)+1, int(params.Position.Character))
	visitor := &NodePathVisitor{
		targetLine:   int(params.Position.Line) + 1,
		targetColumn: int(params.Position.Character),
	}

	root := astFile.Docs[0]
	// Traverse the AST with the visitor
	ast.Walk(visitor, root)
	if visitor.foundNode == nil {
		return nil, yaml.ErrNotFoundNode
	}

	parent := ast.Parent(root, visitor.foundNode)
	if parent.Type() == ast.MappingValueType {
		parentNode := parent.(*ast.MappingValueNode)
		switch parentNode.Key.String() {
		case "func":
			items := make([]protocol.CompletionItem, 0, len(h.project.Functions))
			for name, f := range h.project.Functions {
				l := f.List()
				for c := range l {
					l[c].ParamsYAML = ""
				}
				detail, _ := yaml.MarshalContext(ctx, l, yaml.UseLiteralStyleIfMultiline(true))
				items = append(items, protocol.CompletionItem{Label: name, Kind: protocol.CompletionItemKindFunction, Documentation: string(detail)})
			}
			return protocol.CompletionList{
				IsIncomplete: false,
				Items:        items,
			}, nil
		case "command":
			commands := command.RegisteredCommandNames()
			items := make([]protocol.CompletionItem, 0, len(commands))
			for _, c := range commands {
				items = append(items, protocol.CompletionItem{Label: c, Kind: protocol.CompletionItemKindFunction})
			}
			return protocol.CompletionList{
				IsIncomplete: false,
				Items:        items,
			}, nil
		}

	}
	return nil, nil
}

func (h *Handler) handleInitialize(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result any, err error) {
	var params protocol.InitializeParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}

	h.workspaceFolders = params.WorkspaceFolders
	h.conn = conn

	slog.Debug("Initialized", "workspaceFolders", params.WorkspaceFolders)

	for _, f := range params.WorkspaceFolders {
		fPath := uriToPath(f.URI)
		dirEntries, err := os.ReadDir(fPath)
		if err != nil {
			continue
		}
		for _, d := range dirEntries {
			if d.Name() == "evergreen.yml" || d.Name() == "evergreen.yaml" {
				h.rootYamlPath = filepath.Join(fPath, d.Name())
				break
			}
		}
		if h.rootYamlPath != "" {
			break
		}
	}
	cfg, err := os.ReadFile(h.rootYamlPath)
	if err != nil {
		return nil, err
	}
	h.project = &model.Project{}
	_, err = model.LoadProjectInto(ctx, cfg, nil, "id", h.project)
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
		},
	}, nil
}

func (h *Handler) handleTextDocumentDidOpen(_ context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result any, err error) {
	var params protocol.DidOpenTextDocumentParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}
	h.textDocuments[params.TextDocument.URI] = params.TextDocument
	return nil, nil
}

// NodePathVisitor is a custom visitor to find the path to a YAML node based on position
type NodePathVisitor struct {
	targetLine   int
	targetColumn int
	foundNode    ast.Node
}

// Visit is called for each AST node during traversal
func (v *NodePathVisitor) Visit(node ast.Node) ast.Visitor {
	// Check if the position overlaps with this node
	tkn := node.GetToken()
	start := tkn.Position

	if start.Line <= v.targetLine && start.Column <= v.targetColumn {
		switch n := node.(type) {
		case *ast.CommentNode:
			v.foundNode = n
		case *ast.NullNode:
			v.foundNode = n
		case *ast.IntegerNode:
			v.foundNode = n
		case *ast.FloatNode:
			v.foundNode = n
		case *ast.StringNode:
			v.foundNode = n
		case *ast.MergeKeyNode:
			v.foundNode = n
		case *ast.BoolNode:
			v.foundNode = n
		case *ast.InfinityNode:
			v.foundNode = n
		case *ast.NanNode:
			v.foundNode = n
		case *ast.LiteralNode:
			v.foundNode = n
		}
	}
	return v
}
