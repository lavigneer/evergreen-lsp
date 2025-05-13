package lsp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/a-h/templ/lsp/protocol"
	"github.com/evergreen-ci/evergreen/agent/command"
	"github.com/evergreen-ci/evergreen/model"
	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"
	"github.com/sourcegraph/jsonrpc2"
)

func (h *Handler) handleTextDocumentCompletion(ctx context.Context, req *jsonrpc2.Request) (any, error) {
	var params protocol.CompletionParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}
	visitor, err := h.getNodeAtPosition(params.TextDocument.URI, params.Position)
	if err != nil {
		return nil, err
	}

	items := []protocol.CompletionItem{}
	parent := ast.Parent(visitor.RootNode, visitor.FoundNode)
	if parent.Type() == ast.MappingValueType {
		//nolint:forcetypeassert // we already check the type above
		parentNode := parent.(*ast.MappingValueNode)
		switch parentNode.Key.String() {
		case "func":
			items = funcComplete(ctx, h.project)
		case "command":
			items = commandComplete()
		}

	}
	return protocol.CompletionList{
		IsIncomplete: false,
		Items:        items,
	}, nil
}

func funcComplete(ctx context.Context, project *model.Project) []protocol.CompletionItem {
	items := make([]protocol.CompletionItem, 0, len(project.Functions))
	for name, f := range project.Functions {
		l := f.List()
		for c := range l {
			l[c].ParamsYAML = ""
		}
		detail, _ := yaml.MarshalContext(ctx, l, yaml.UseLiteralStyleIfMultiline(true))
		items = append(items, protocol.CompletionItem{Label: name, Kind: protocol.CompletionItemKindFunction, Documentation: string(detail)})
	}
	return items
}

func commandComplete() []protocol.CompletionItem {
	commands := command.RegisteredCommandNames()
	items := make([]protocol.CompletionItem, 0, len(commands))
	for _, c := range commands {
		items = append(items, protocol.CompletionItem{Label: c, Kind: protocol.CompletionItemKindFunction})
	}
	return items
}

func (h *Handler) handleTextDocumentDefinition(ctx context.Context, req *jsonrpc2.Request) (any, error) {
	var params protocol.DefinitionParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}
	visitor, err := h.getNodeAtPosition(params.TextDocument.URI, params.Position)
	if err != nil {
		return nil, err
	}

	parent := ast.Parent(visitor.RootNode, visitor.FoundNode)
	if parent.Type() == ast.MappingValueType {
		//nolint:forcetypeassert // we already check the type above
		parentNode := parent.(*ast.MappingValueNode)
		switch parentNode.Key.String() {
		case "func":
			fs := visitor.FoundNode.String()
			nodePath, err := yaml.PathString(fmt.Sprintf("$.functions.%s", fs))
			if err != nil {
				return nil, err
			}
			n, err := nodePath.FilterNode(visitor.RootNode)
			if err != nil {
				return nil, err
			}
			if n == nil {
				return nil, yaml.ErrNotFoundNode
			}
			token := n.GetToken()
			line := uint32(token.Position.Line) - 2
			character := uint32(token.Position.IndentNum) - 2
			return protocol.Location{
				URI: params.TextDocument.URI,
				Range: protocol.Range{
					Start: protocol.Position{
						Line:      line,
						Character: character,
					},
					End: protocol.Position{
						Line:      line,
						Character: character + uint32(len(token.Origin)) - 1,
					},
				},
			}, nil
		}

	}
	return nil, nil
}

func (h *Handler) handleTextDocumentHover(ctx context.Context, req *jsonrpc2.Request) (any, error) {
	var params protocol.HoverParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}
	visitor, err := h.getNodeAtPosition(params.TextDocument.URI, params.Position)
	if err != nil {
		return nil, err
	}

	parent := ast.Parent(visitor.RootNode, visitor.FoundNode)
	if parent.Type() == ast.MappingValueType {
		//nolint:forcetypeassert // we already check the type above
		parentNode := parent.(*ast.MappingValueNode)
		switch parentNode.Key.String() {
		case "func":
			fs := visitor.FoundNode.String()
			nodePath, err := yaml.PathString(fmt.Sprintf("$.functions.%s", fs))
			if err != nil {
				return nil, err
			}
			n, err := nodePath.FilterNode(visitor.RootNode)
			if err != nil {
				return nil, err
			}
			if n == nil {
				return nil, yaml.ErrNotFoundNode
			}
			defYaml, err := nodeToDedentedYaml(ctx, n)
			if err != nil {
				return nil, err
			}
			return protocol.Hover{
				Contents: protocol.MarkupContent{
					Kind:  protocol.PlainText,
					Value: string(defYaml),
				},
			}, nil
		}

	}
	return nil, nil
}

var ErrDocumentNotFound = errors.New("document not found")

func (h *Handler) getNodeAtPosition(docURI protocol.DocumentURI, position protocol.Position) (*NodePathVisitor, error) {
	doc, ok := h.textDocuments[docURI]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrDocumentNotFound, docURI)
	}

	astFile, err := parser.ParseBytes([]byte(doc.Text), parser.ParseComments)
	if err != nil {
		return nil, err
	}

	root := astFile.Docs[0].Body
	visitor := &NodePathVisitor{
		TargetLine:   int(position.Line) + 1,
		TargetColumn: int(position.Character) + 1,
		RootNode:     root,
	}

	// Traverse the AST with the visitor
	ast.Walk(visitor, visitor.RootNode)
	if visitor.FoundNode == nil {
		return nil, yaml.ErrNotFoundNode
	}
	return visitor, nil
}
