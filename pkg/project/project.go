package project

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/a-h/templ/lsp/protocol"
	"github.com/evergreen-ci/evergreen/agent/command"
	"github.com/evergreen-ci/evergreen/model"
	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"
	"github.com/lavigneer/evergreen-lsp/pkg/util"
)

type Project struct {
	rootPath      string
	BasePath      string `yaml:"path"`
	Data          *model.Project
	TextDocuments map[protocol.DocumentURI]*Document
}

func New(path string) *Project {
	return &Project{
		Data:          &model.Project{},
		BasePath:      path,
		rootPath:      "",
		TextDocuments: make(map[protocol.DocumentURI]*Document),
	}
}

func (w *Project) Path() string {
	return filepath.Join(w.rootPath, w.BasePath)
}

func (w *Project) SetRoot(rootPath string) {
	w.rootPath = rootPath
}

func (w *Project) Init(ctx context.Context) error {
	w.TextDocuments = make(map[protocol.DocumentURI]*Document)
	path := w.Path()
	cfg, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	err = w.loadProject(ctx, cfg)
	if err != nil {
		return err
	}

	w.AddDocument(ctx, protocol.TextDocumentItem{
		URI:        protocol.DocumentURI(w.Path()),
		LanguageID: "yaml",
		Version:    0,
		Text:       string(cfg),
	})

	includesPath, err := yaml.PathString("$.include[*]")
	if err != nil {
		return err
	}
	includes := []struct {
		FileName string `yaml:"filename,omitempty"`
		Module   string `yaml:"module,omitempty"`
	}{}
	err = includesPath.Read(strings.NewReader(string(cfg)), &includes)
	if err != nil {
		// No includes, we are done and don't error
		if errors.Is(err, yaml.ErrNotFoundNode) {
			return nil
		}
		return err
	}
	for _, i := range includes {
		p := filepath.Join(w.rootPath, i.FileName)
		docText, err := os.ReadFile(p)
		if err != nil {
			slog.Error("Uh oh!")
			continue
		}
		_, err = w.AddDocument(ctx, protocol.TextDocumentItem{
			URI:        protocol.DocumentURI(p),
			LanguageID: "yaml",
			Version:    0,
			Text:       string(docText),
		})
		if err != nil {
			slog.Error("Uh oh!")
			continue
		}
	}
	return nil
}

func (w *Project) loadProject(ctx context.Context, cfg []byte) error {
	path := w.Path()
	w.Data = &model.Project{}
	// Hacky workaround since evergreen loads relative to cwd
	if w.rootPath != "" {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		os.Chdir(w.rootPath)
		defer os.Chdir(cwd)
	}
	_, err := model.LoadProjectInto(ctx, cfg, &model.GetProjectOpts{
		ReadFileFrom: model.ReadFromLocal,
		RemotePath:   path,
		Ref: &model.ProjectRef{
			RemotePath: path,
		},
	}, "id", w.Data)
	if err != nil {
		return err
	}
	return nil
}

func (w *Project) AddDocument(ctx context.Context, doc protocol.TextDocumentItem) (*Document, error) {
	d := &Document{TextDocumentItem: doc, Workspace: w}
	w.TextDocuments[doc.URI] = d
	err := d.Parse()
	return d, err
}

func (w *Project) RemoveDocument(ctx context.Context, docID protocol.TextDocumentIdentifier) {
	delete(w.TextDocuments, docID.URI)
}

func (w *Project) UpdateDocument(ctx context.Context, docID protocol.VersionedTextDocumentIdentifier, textChanges protocol.TextDocumentContentChangeEvent) (*Document, error) {
	doc, ok := w.TextDocuments[docID.URI]
	if !ok {
		panic("fix this")
	}
	doc.UpdateText(textChanges.Text, docID.Version)
	err := doc.Parse()
	return doc, err
}

func (w *Project) References(ctx context.Context, nodeStr string) []protocol.Location {
	references := []protocol.Location{}
	for _, d := range w.TextDocuments {
		refs := d.References[nodeStr]
		for _, r := range refs {
			references = append(references, r.Location)
		}
	}
	return references
}

func (w *Project) Definition(ctx context.Context, nodeStr string) *protocol.Location {
	for _, d := range w.TextDocuments {
		def, ok := d.Definitions[nodeStr]
		if ok {
			return &def.Location
		}
	}
	return nil
}

func (w *Project) Hover(ctx context.Context, nodeStr string) *protocol.Hover {
	for _, d := range w.TextDocuments {
		def, ok := d.Hovers[nodeStr]
		if ok {
			defYaml, err := util.NodeToDedentedYaml(ctx, def.Node)
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

type Document struct {
	protocol.TextDocumentItem
	References  map[string][]DocumentNodeLocation
	Definitions map[string]DocumentNodeLocation
	Hovers      map[string]DocumentNodeLocation
	Diagnostics []protocol.Diagnostic
	AST         *ast.File
	Workspace   *Project
}

var deprecatedCommands = []string{"shell.exec"}

func (d *Document) RootNode() ast.Node {
	return d.AST.Docs[0].Body
}

func (d *Document) UpdateText(content string, version int32) {
	if d.Version > version {
		panic("uh oh! Old version came later!")
	}
	d.Version = version
	d.Text = content
}

func (d *Document) Parse() error {
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

//nolint:ireturn
func (d *Document) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.StringNode:
		nodeStr := n.Token.Value
		_, ok := d.Workspace.Data.Functions[nodeStr]
		if ok {
			references := d.References[nodeStr]
			references = append(references, DocumentNodeLocation{
				Node:     n,
				Location: d.LocationFromNode(n),
			})
			d.References[nodeStr] = references
		}
	case *ast.MappingValueNode:
		switch n.Key.GetToken().Value {
		case "func":
			nodeStr := n.Value.GetToken().Value
			_, ok := d.Workspace.Data.Functions[nodeStr]
			if !ok {
				d.Diagnostics = append(d.Diagnostics, protocol.Diagnostic{
					Source:  "evergreenlsp",
					Message: fmt.Sprintf("function %q is not defined", nodeStr),
					Range:   util.RangeFromNode(n.Value, nil),
				})
			}

		case "command":
			nodeStr := n.Value.GetToken().Value
			deprecated := slices.Contains(deprecatedCommands, nodeStr)
			if deprecated {
				d.Diagnostics = append(d.Diagnostics, protocol.Diagnostic{
					Source:   "evergreenlsp",
					Message:  fmt.Sprintf("command %q is deprecated", nodeStr),
					Severity: protocol.DiagnosticSeverityWarning,
					Range:    util.RangeFromNode(n.Value, nil),
				})
			} else {
				commands := command.RegisteredCommandNames()
				ok := slices.Contains(commands, nodeStr)
				if !ok {
					d.Diagnostics = append(d.Diagnostics, protocol.Diagnostic{
						Source:  "evergreenlsp",
						Message: fmt.Sprintf("command %q is not defined", nodeStr),
						Range:   util.RangeFromNode(n.Value, nil),
					})
				}
			}

		}
	case *ast.MappingNode:
		if n.Path == "$.functions" {
			for _, v := range n.Values {
				nodeStr := v.Key.String()
				_, ok := d.Workspace.Data.Functions[nodeStr]
				if ok {
					d.Definitions[nodeStr] = DocumentNodeLocation{
						Node:     v.Key,
						Location: d.LocationFromNode(v.Key),
					}
					d.Hovers[nodeStr] = DocumentNodeLocation{
						Node:     v.Value,
						Location: d.LocationFromNode(v.Value),
					}
				}
			}
		}
	}
	return d
}

func (d *Document) LocationFromNode(n ast.Node) protocol.Location {
	r := util.RangeFromNode(n, nil)
	return protocol.Location{
		URI:   d.URI,
		Range: r,
	}
}

func (d *Document) NodeFromLocation(position protocol.Position) (ast.Node, error) {
	root := d.RootNode()
	visitor := &util.NodePathVisitor{
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
