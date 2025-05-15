package lint

import (
	"fmt"
	"slices"
	"strings"

	"github.com/a-h/templ/lsp/protocol"
	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"github.com/lavigneer/evergreen-lsp/pkg/config"
	"github.com/lavigneer/evergreen-lsp/pkg/util"
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
	if n, ok := node.(*ast.MappingValueNode); ok {
		if n.Key.GetToken().Value == "tasks" && n.GetPath() != "$.tasks" {
			path, _ := yaml.PathString("$[*].name")
			nameNodes := []ast.Node{}
			_ = path.Read(n.Value, &nameNodes)
			for _, name := range nameNodes {
				names := strings.Split(name.GetToken().Value, " ")
				usesDirectName := slices.ContainsFunc(names, func(n string) bool {
					return n != "*" && !strings.HasPrefix(n, "!.") && !strings.HasPrefix(n, ".")
				})
				if usesDirectName {
					diagnostics = append(diagnostics, protocol.Diagnostic{
						Source:   "enforce-tags",
						Message:  fmt.Sprintf("task reference %q does not use tag", name.GetToken().Value),
						Severity: protocol.DiagnosticSeverityWarning,
						Range:    util.RangeFromNode(name, n),
					})
				}
			}
		}
	}

	return diagnostics
}
