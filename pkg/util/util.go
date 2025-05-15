package util

import (
	"context"

	"github.com/a-h/templ/lsp/protocol"
	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/token"
)

func NodeToDedentedYaml(ctx context.Context, n ast.Node) (string, error) {
	// Convert to an any to force yaml to dedent
	var val any
	err := yaml.NodeToValue(n, &val)
	if err != nil {
		return "", err
	}
	defYaml, err := yaml.MarshalContext(ctx, val, yaml.UseLiteralStyleIfMultiline(true))
	if err != nil {
		return "", err
	}
	return string(defYaml), nil
}

func RangeFromNode(n ast.Node, offsetNode ast.Node) protocol.Range {
	var t *token.Token
	switch node := n.(type) {
	case *ast.MappingValueNode:
		t = node.Key.GetToken()
	default:
		t = node.GetToken()
	}
	line := uint32(t.Position.Line) - 1
	character := uint32(t.Position.Column) - 1
	if offsetNode != nil {
		offsetNodeToken := offsetNode.GetToken()
		line += uint32(offsetNodeToken.Position.Line) + 1
	}
	if n.GetComment() != nil {
		line++
		// line += uint32(n.GetComment().GetToken().Position.Line)
	}
	return protocol.Range{
		Start: protocol.Position{
			Line:      line,
			Character: character,
		},
		End: protocol.Position{
			Line:      line,
			Character: character + uint32(len(t.Origin)) - 1,
		},
	}
}
