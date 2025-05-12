package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/a-h/templ/lsp/protocol"
	"github.com/evergreen-ci/evergreen/model"
	"github.com/sourcegraph/jsonrpc2"
	"gopkg.in/yaml.v3"
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

	var root yaml.Node
	err := yaml.Unmarshal([]byte(doc.Text), &root) // root.Kind will be yaml.DocumentNode
	if err != nil {
		return nil, err
	}
	node := FindDeepestOverlappingNode(&root, int(params.Position.Line)+1, int(params.Position.Character))
	for c, n := range node {
		slog.Info(fmt.Sprintf("###########%d", c), "value", n.Value, "kind", n.Kind, "tag",  n.Tag)
	}
	return nil, nil
}

// type nodeLocationVisitor struct {
// 	nodes          []*ast.Node
// 	targetPosition protocol.Position
// }
//
// func (n *nodeLocationVisitor) Visit(node ast.Node) ast.Visitor {
// 	tkn := node.GetToken()
// 	nextTkn := tkn.Next
//
// 	if n.targetPosition.Line < uint32(tkn.Position.Line) || n.targetPosition.Line > uint32(nextTkn.Position.Line) {
// 		return nil
// 	}
//
// 	if n.targetPosition.Line < uint32(tkn.Position.Line) || n.targetPosition.Line > uint32(nextTkn.Position.Line) {
// 		return nil
// 	}
// 	return n
// }
//
// func findTokenAtPosition(docText string, position protocol.Position) (*token.Token, error) {
// 	tokens := lexer.Tokenize(docText)
// 	for _, t := range tokens {
// 		line := int(position.Line + 1)
// 		ch := int(position.Character + 1)
// 		if t.Position.Line > line {
// 			break
// 		}
// 		if t.Position.Line < line {
// 			continue
// 		}
// 		slog.Info("TOKEN", "token", fmt.Sprintf(
// 			"[TYPE]:%q [CHARTYPE]:%q [INDICATOR]:%q [VALUE]:%q [ORG]:%q [POS(line:column:level:offset)]: %d:%d:%d:%d\n",
// 			t.Type, t.CharacterType, t.Indicator, t.Value, t.Origin, t.Position.Line, t.Position.Column, t.Position.IndentLevel, t.Position.Offset,
// 		))
//
// 	}
// 	return nil, nil
// }

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

// DoesPositionOverlap determines if the position (line, column) overlaps with the range of the node's position.
func DoesPositionOverlap(node *yaml.Node, targetLine, targetColumn int) bool {
	// A node starts at (node.Line, node.Column).
	startLine := node.Line
	startColumn := node.Column

	// Calculate the approximate "end" position:
	// If the node has children, its end position can be approximated
	// based on the last child node's `Line`.
	endLine := node.Line
	endColumn := node.Column

	// Special handling for scalar nodes
	if node.Kind == yaml.ScalarNode && node.Value != "" {
		// Scalars are single-line and their end column is determined by the content length
		endColumn = startColumn + len(node.Value) - 1
	} else if len(node.Content) > 0 {
		// For compound nodes, calculate end position based on the last child
		lastChild := node.Content[len(node.Content)-1]
		endLine = lastChild.Line
		endColumn = math.MaxInt
	}

	// Check if the target position is within the node's range.
	if targetLine > startLine || (targetLine == startLine && targetColumn >= startColumn) {
		if targetLine < endLine || (targetLine == endLine && targetColumn <= endColumn) {
			return true
		}
	}

	return false
}

// FindDeepestOverlappingNode finds the deepest (leaf) node that overlaps the specified position.
func FindDeepestOverlappingNode(node *yaml.Node, targetLine, targetColumn int) []*yaml.Node {
	var overlappingNode []*yaml.Node
	// Check if the current node overlaps with the desired position
	if DoesPositionOverlap(node, targetLine, targetColumn) {
		overlappingNode = append(overlappingNode, node)
	}

	// Traverse recursively into child nodes
	for _, child := range node.Content {
		found := FindDeepestOverlappingNode(child, targetLine, targetColumn)
		if len(found) > 0 {
			overlappingNode = append(overlappingNode, found...)
		}
	}

	return overlappingNode

	//
	// // Traverse child nodes to check for deeper overlaps.
	// var overlappingNode []*yaml.Node
	// for _, child := range node.Content {
	// 	foundNode := FindDeepestOverlappingNode(child, targetLine, targetColumn)
	// 	if foundNode != nil {
	// 		overlappingNode = append(overlappingNode, foundNode...)
	// 	}
	// }
	//
	// // If no deeper overlapping node is found, check the current node.
	// if DoesPositionOverlap(node, targetLine, targetColumn) {
	// 	overlappingNode = append(overlappingNode, node)
	// }
	//
	// return overlappingNode
}
