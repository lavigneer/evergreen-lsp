package lint

import (
	"github.com/a-h/templ/lsp/protocol"
	"github.com/goccy/go-yaml/ast"
	"github.com/lavigneer/evergreen-lsp/pkg/config"
	"github.com/lavigneer/evergreen-lsp/pkg/project"
)

type ExecutorDiagnostics map[*project.Document][]protocol.Diagnostic

type Executor struct {
	workspace   *project.Project
	settings    config.Lint
	diagnostics ExecutorDiagnostics
	linters     []Linter
}

type Linter interface {
	Check(node ast.Node) []protocol.Diagnostic
	Enabled(settings config.Lint) bool
	Register(executor *Executor)
}

var linters = []Linter{
	&DeprecatedLinter{},
	&UndefinedLinter{},
	&EnforceTagsLinter{},
	&NoInlineScriptsLinter{},
}

func New(workspace *project.Project, settings config.Lint) *Executor {
	executor := &Executor{
		workspace:   workspace,
		settings:    settings,
		diagnostics: make(map[*project.Document][]protocol.Diagnostic),
		linters:     linters,
	}
	for _, l := range executor.linters {
		l.Register(executor)
	}

	return executor
}

func (e *Executor) Lint() (ExecutorDiagnostics, error) {
	for _, v := range e.workspace.TextDocuments {
		visitor := LintVisitor{
			diagnostics: make([]protocol.Diagnostic, 0),
			linters:     e.linters,
		}
		ast.Walk(&visitor, v.RootNode())
		e.diagnostics[v] = visitor.diagnostics

	}
	return e.diagnostics, nil
}

type LintVisitor struct {
	diagnostics []protocol.Diagnostic
	linters     []Linter
}

func (l *LintVisitor) Visit(node ast.Node) ast.Visitor {
	for _, linter := range l.linters {
		linterDiagnostics := linter.Check(node)
		l.diagnostics = append(l.diagnostics, linterDiagnostics...)
	}
	return l
}
