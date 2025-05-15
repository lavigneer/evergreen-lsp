package lint

import (
	"fmt"
	"slices"

	"github.com/a-h/templ/lsp/protocol"
	"github.com/goccy/go-yaml/ast"
	"github.com/lavigneer/evergreen-lsp/pkg/config"
	"github.com/lavigneer/evergreen-lsp/pkg/util"
)

var deprecatedCommands = []string{"shell.exec"}

type DeprecatedLinter struct {
	executor *Executor
}

func (l *DeprecatedLinter) Register(executor *Executor) {
	l.executor = executor
}

func (l *DeprecatedLinter) Enabled(_ config.Lint) bool {
	return true
}

func (l *DeprecatedLinter) Check(node ast.Node) []protocol.Diagnostic {
	diagnostics := []protocol.Diagnostic{}
	if n, ok := node.(*ast.MappingValueNode); ok {
		if n.Key.GetToken().Value == "command" {
			nodeStr := n.Value.GetToken().Value
			deprecated := slices.Contains(deprecatedCommands, nodeStr)
			if deprecated {
				diagnostics = append(diagnostics, protocol.Diagnostic{
					Source:   "deprecated-command",
					Message:  fmt.Sprintf("command %q is deprecated", nodeStr),
					Severity: protocol.DiagnosticSeverityWarning,
					Range:    util.RangeFromNode(n.Value, nil),
				})
			}
		}
	}
	return diagnostics
}
