package lsp

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/a-h/templ/lsp/protocol"
	"github.com/evergreen-ci/evergreen/agent/command"
	"github.com/evergreen-ci/evergreen/model"
	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"github.com/sourcegraph/jsonrpc2"
)

func (h *Handler) handleTextDocumentCompletion(ctx context.Context, req *jsonrpc2.Request) (any, error) {
	var params protocol.CompletionParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}
	if res, ok := h.config.FindProjDoc(params.TextDocument.URI); ok {
		node, err := res.Document.NodeFromLocation(params.Position)
		if err != nil {
			return nil, err
		}
		r := res.Document.LocationFromNode(node).Range
		items := []protocol.CompletionItem{}
		parent := ast.Parent(res.Document.RootNode(), node)
		if parent.Type() == ast.MappingValueType {
			//nolint:forcetypeassert // we already check the type above
			parentNode := parent.(*ast.MappingValueNode)
			switch parentNode.Key.GetToken().Value {
			case "func":
				items = funcComplete(ctx, res.Project.Data, r)
			case "command":
				items = commandComplete()
			}

		}
		return protocol.CompletionList{
			IsIncomplete: false,
			Items:        items,
		}, nil
	}
	return nil, ErrDocumentNotFound
}

func funcComplete(ctx context.Context, project *model.Project, r protocol.Range) []protocol.CompletionItem {
	items := make([]protocol.CompletionItem, 0, len(project.Functions))
	for name, f := range project.Functions {
		l := f.List()
		for c := range l {
			l[c].ParamsYAML = ""
		}
		detail, _ := yaml.MarshalContext(ctx, l, yaml.UseLiteralStyleIfMultiline(true))
		items = append(items, protocol.CompletionItem{
			Label:            name,
			Kind:             protocol.CompletionItemKindFunction,
			Documentation:    string(detail),
			InsertTextFormat: protocol.InsertTextFormatSnippet,
			FilterText:       name,
			TextEdit: &protocol.TextEditOrInsertReplaceEdit{
				TextEdit: &protocol.TextEdit{
					NewText: name,
					Range:   r,
				},
			},
		})
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
	if res, ok := h.config.FindProjDoc(params.TextDocument.URI); ok {
		node, err := res.Document.NodeFromLocation(params.Position)
		if err != nil {
			return nil, err
		}

		nodeStr := node.GetToken().Value
		def := res.Project.Definition(ctx, nodeStr)
		return def, nil
	}
	return nil, ErrDocumentNotFound
}

func (h *Handler) handleTextDocumentHover(ctx context.Context, req *jsonrpc2.Request) (any, error) {
	var params protocol.HoverParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}

	if res, ok := h.config.FindProjDoc(params.TextDocument.URI); ok {
		node, err := res.Document.NodeFromLocation(params.Position)
		if err != nil {
			return nil, err
		}

		nodeStr := node.GetToken().Value
		def := res.Project.Hover(ctx, nodeStr)
		return def, nil
	}
	return nil, ErrDocumentNotFound
}

func (h *Handler) handleTextDocumentReferences(ctx context.Context, req *jsonrpc2.Request) (any, error) {
	var params protocol.ReferenceParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}
	if res, ok := h.config.FindProjDoc(params.TextDocument.URI); ok {
		node, err := res.Document.NodeFromLocation(params.Position)
		if err != nil {
			return nil, err
		}

		nodeStr := node.GetToken().Value
		refs := res.Project.References(ctx, nodeStr)
		return refs, nil
	}
	return nil, ErrDocumentNotFound
}

var ErrDocumentNotFound = errors.New("document not found")
