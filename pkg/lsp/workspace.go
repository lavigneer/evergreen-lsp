package lsp

import (
	"context"
	"os"

	"github.com/a-h/templ/lsp/protocol"
	"github.com/evergreen-ci/evergreen/model"
	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"
)

type Workspace struct {
	path          string
	Project       *model.Project
	TextDocuments map[protocol.DocumentURI]*WorkspaceDocument
}

func NewWorkspace(path string) *Workspace {
	return &Workspace{
		Project:       &model.Project{},
		path:          path,
		TextDocuments: make(map[protocol.DocumentURI]*WorkspaceDocument),
	}
}

func (w *Workspace) Init(ctx context.Context) error {
	cfg, err := os.ReadFile(w.path)
	if err != nil {
		return err
	}
	_, err = model.LoadProjectInto(ctx, cfg, nil, "id", w.Project)
	if err != nil {
		return err
	}
	return nil
}

func (w *Workspace) AddDocument(ctx context.Context, doc protocol.TextDocumentItem) error {
	d := &WorkspaceDocument{TextDocumentItem: doc, Workspace: w}
	w.TextDocuments[doc.URI] = d
	err := d.Parse()
	return err
}

func (w *Workspace) RemoveDocument(ctx context.Context, docID protocol.TextDocumentIdentifier) {
	delete(w.TextDocuments, docID.URI)
}

func (w *Workspace) UpdateDocument(ctx context.Context, docID protocol.VersionedTextDocumentIdentifier, textChanges protocol.TextDocumentContentChangeEvent) error {
	doc, ok := w.TextDocuments[docID.URI]
	if !ok {
		panic("fix this")
	}
	doc.UpdateText(textChanges.Text, docID.Version)
	err := doc.Parse()
	return err
}

func (w *Workspace) References(ctx context.Context, nodeStr string) []protocol.Location {
	references := []protocol.Location{}
	for _, d := range w.TextDocuments {
		refs := d.References[nodeStr]
		for _, r := range refs {
			references = append(references, r.Location)
		}
	}
	return references
}

func (w *Workspace) Definition(ctx context.Context, nodeStr string) *protocol.Location {
	for _, d := range w.TextDocuments {
		def, ok := d.Definitions[nodeStr]
		if ok {
			return &def.Location
		}
	}
	return nil
}

func (w *Workspace) Hover(ctx context.Context, nodeStr string) *protocol.Hover {
	for _, d := range w.TextDocuments {
		def, ok := d.Hovers[nodeStr]
		if ok {
			defYaml, err := nodeToDedentedYaml(ctx, def.Node)
			if err != nil {
				return nil
			}

			return &protocol.Hover{
				Contents: protocol.MarkupContent{
					Kind:  protocol.PlainText,
					Value: defYaml,
				},
			}
		}
	}
	return nil
}

type DocumentNodeLocation struct {
	Location protocol.Location
	Node     ast.Node
}

type WorkspaceDocument struct {
	protocol.TextDocumentItem
	References  map[string][]DocumentNodeLocation
	Definitions map[string]DocumentNodeLocation
	Hovers      map[string]DocumentNodeLocation
	Diagnostics []protocol.Diagnostic
	AST         *ast.File
	Workspace   *Workspace
}

func (d *WorkspaceDocument) RootNode() ast.Node {
	return d.AST.Docs[0].Body
}

func (d *WorkspaceDocument) UpdateText(content string, version int32) {
	if d.Version > version {
		panic("uh oh! Old version came later!")
	}
	d.Version = version
	d.Text = content
}

func (d *WorkspaceDocument) Parse() error {
	astFile, err := parser.ParseBytes([]byte(d.Text), parser.ParseComments)
	if err != nil {
		return err
	}
	d.AST = astFile
	d.Definitions = make(map[string]DocumentNodeLocation)
	d.Hovers = make(map[string]DocumentNodeLocation)
	d.References = make(map[string][]DocumentNodeLocation)
	d.Diagnostics = make([]protocol.Diagnostic, 0)
	for _, doc := range astFile.Docs {
		ast.Walk(d, doc.Body)
	}
	return nil
}

func (d *WorkspaceDocument) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.StringNode:
		nodeStr := n.Token.Value
		_, ok := d.Workspace.Project.Functions[nodeStr]
		if ok {
			references := d.References[nodeStr]
			references = append(references, DocumentNodeLocation{
				Node:     n,
				Location: d.locationFromNode(n),
			})
			d.References[nodeStr] = references
		}
	case *ast.MappingNode:
		if n.Path == "$.functions" {
			for _, v := range n.Values {
				nodeStr := v.Key.String()
				_, ok := d.Workspace.Project.Functions[nodeStr]
				if ok {
					d.Definitions[nodeStr] = DocumentNodeLocation{
						Node:     v.Key,
						Location: d.locationFromNode(v.Key),
					}
					d.Hovers[nodeStr] = DocumentNodeLocation{
						Node:     v.Value,
						Location: d.locationFromNode(v.Value),
					}
				}
			}
		}
	}
	return d
}

func (d *WorkspaceDocument) locationFromNode(n ast.Node) protocol.Location {
	token := n.GetToken()
	line := uint32(token.Position.Line) - 1
	character := uint32(token.Position.Column) - 1
	return protocol.Location{
		URI: d.URI,
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
	}
}

func (d *WorkspaceDocument) nodeFromLocation(position protocol.Position) (ast.Node, error) {
	root := d.RootNode()
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
	return visitor.FoundNode, nil
}
