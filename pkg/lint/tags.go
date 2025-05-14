package lint

import (
	"github.com/a-h/templ/lsp/protocol"
	"github.com/goccy/go-yaml/ast"
	"github.com/lavigneer/evergreen-lsp/pkg/config"
)

type EnforceTagsLinter struct {
	executor *Executor
}

func (l *EnforceTagsLinter) Register(executor *Executor) {
	l.executor = executor
}

func (l *EnforceTagsLinter) Enabled(settings config.Lint) bool {
	return settings.EnforceTags
}

func (l *EnforceTagsLinter) Check(node ast.Node) []protocol.Diagnostic {
	diagnostics := []protocol.Diagnostic{}
	return diagnostics
}
