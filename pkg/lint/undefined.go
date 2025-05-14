package lint

import (
	"fmt"
	"slices"

	"github.com/a-h/templ/lsp/protocol"
	"github.com/evergreen-ci/evergreen/agent/command"
	"github.com/goccy/go-yaml/ast"
	"github.com/lavigneer/evergreen-lsp/pkg/config"
	"github.com/lavigneer/evergreen-lsp/pkg/util"
)

type UndefinedLinter struct {
	executor *Executor
}

func (l *UndefinedLinter) Register(executor *Executor) {
	l.executor = executor
}

func (l *UndefinedLinter) Enabled(_ config.Lint) bool {
	return true
}

func (l *UndefinedLinter) Check(node ast.Node) []protocol.Diagnostic {
	diagnostics := []protocol.Diagnostic{}
	if n, ok := node.(*ast.MappingValueNode); ok {
		switch n.Key.GetToken().Value {
		case "func":
			nodeStr := n.Value.GetToken().Value
			_, ok := l.executor.workspace.Data.Functions[nodeStr]
			if !ok {
				diagnostics = append(diagnostics, protocol.Diagnostic{
					Severity: protocol.DiagnosticSeverityError,
					Source:   "no-undefined",
					Message:  fmt.Sprintf("function %q is not defined", nodeStr),
					Range:    util.RangeFromNode(n.Value),
				})
			}

		case "command":
			nodeStr := n.Value.GetToken().Value
			commands := command.RegisteredCommandNames()
			ok := slices.Contains(commands, nodeStr)
			if !ok {
				diagnostics = append(diagnostics, protocol.Diagnostic{
					Severity: protocol.DiagnosticSeverityError,
					Source:   "no-undefined",
					Message:  fmt.Sprintf("command %q is not defined", nodeStr),
					Range:    util.RangeFromNode(n.Value),
				})
			}

		}
	}
	return diagnostics
}
