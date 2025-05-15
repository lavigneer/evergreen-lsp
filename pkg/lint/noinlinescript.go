package lint

import (
	"fmt"

	"github.com/a-h/templ/lsp/protocol"
	"github.com/goccy/go-yaml/ast"
	"github.com/lavigneer/evergreen-lsp/pkg/config"
	"github.com/lavigneer/evergreen-lsp/pkg/util"
)

type NoInlineScriptsLinter struct {
	executor *Executor
}

func (l *NoInlineScriptsLinter) Register(executor *Executor) {
	l.executor = executor
}

func (l *NoInlineScriptsLinter) Enabled(settings config.Lint) bool {
	return settings.NoInlineScripts
}

func (l *NoInlineScriptsLinter) Check(node ast.Node) []protocol.Diagnostic {
	diagnostics := []protocol.Diagnostic{}
	if m, ok := node.(*ast.MappingValueNode); ok {
		child := m.Value
		if a, ok := child.(*ast.AnchorNode); ok {
			child = a.Value
		}
		switch n := child.(type) {
		case *ast.MappingNode:
			if l.CheckIfCommandWithInline(n) {
				diagnostics = append(diagnostics, protocol.Diagnostic{
					Source:   "no-inline-script",
					Message:  fmt.Sprintf("function %q uses an inline bash script", m.Key.GetToken().Value),
					Severity: protocol.DiagnosticSeverityWarning,
					Range:    util.RangeFromNode(m, nil),
				})
			}
		case *ast.SequenceNode:
			for _, nv := range n.Values {
				if l.CheckIfCommandWithInline(nv) {
					diagnostics = append(diagnostics, protocol.Diagnostic{
						Source:   "no-inline-script",
						Message:  fmt.Sprintf("function %q uses an inline bash script", m.Key.GetToken().Value),
						Severity: protocol.DiagnosticSeverityWarning,
						Range:    util.RangeFromNode(n, nil),
					})
				}
			}
		}
		// if n, ok := child.(*ast.MappingNode); ok {
		// 	usesSubprocess := false
		// 	usesBash := false
		// 	usesBashC := false
		// 	for _, v := range n.Values {
		// 		keyVal := v.Key.GetToken().Value
		// 		if keyVal == "command" && v.Value.GetToken().Value == "subprocess.exec" {
		// 			usesSubprocess = true
		// 		}
		// 		if keyVal == "params" {
		// 			if pn, ok := v.Value.(*ast.MappingNode); ok {
		// 				for _, pnv := range pn.Values {
		// 					keyVal := pnv.Key.GetToken().Value
		// 					if keyVal == "binary" && pnv.Value.GetToken().Value == "bash" {
		// 						usesBash = true
		// 					}
		// 					if keyVal == "args" {
		// 						if a, ok := pnv.Value.(*ast.SequenceNode); ok {
		// 							if len(a.Values) > 0 && a.Values[0].GetToken().Value == "-c" {
		// 								usesBashC = true
		// 							}
		// 						}
		// 					}
		// 				}
		// 			}
		// 		}
		// 		if usesSubprocess && usesBash && usesBashC {
		// 			diagnostics = append(diagnostics, protocol.Diagnostic{
		// 				Source:   "no-inline-script",
		// 				Message:  fmt.Sprintf("function %q uses an inline bash script", m.Key.GetToken().Value),
		// 				Severity: protocol.DiagnosticSeverityWarning,
		// 				Range:    util.RangeFromNode(m, nil),
		// 			})
		// 			return diagnostics
		// 		}
		// 	}
		// }
	}

	return diagnostics
}

func (l *NoInlineScriptsLinter) CheckIfCommandWithInline(n ast.Node) bool {
	node, ok := n.(*ast.MappingNode)
	if !ok {
		return false
	}

	usesSubprocess := false
	usesBashInline := false
	for _, v := range node.Values {
		keyVal := v.Key.GetToken().Value
		if keyVal == "command" && v.Value.GetToken().Value == "subprocess.exec" {
			usesSubprocess = true
		}
		if keyVal == "params" {
			if pn, ok := v.Value.(*ast.MappingNode); ok {
				usesBashInline = l.CheckIfParamsUseInline(pn)
			}
		}
		if usesSubprocess && usesBashInline {
			return true
		}
	}
	return false
}

func (l *NoInlineScriptsLinter) CheckIfParamsUseInline(node *ast.MappingNode) bool {
	usesBash := false
	usesBashC := false
	for _, n := range node.Values {
		keyVal := n.Key.GetToken().Value
		if keyVal == "binary" && n.Value.GetToken().Value == "bash" {
			usesBash = true
		}
		if keyVal == "args" {
			if a, ok := n.Value.(*ast.SequenceNode); ok {
				if len(a.Values) > 0 && a.Values[0].GetToken().Value == "-c" {
					usesBashC = true
				}
			}
		}
		if usesBash && usesBashC {
			return true
		}
	}
	return false
}
