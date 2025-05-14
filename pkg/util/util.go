package util

import (
	"context"

	"github.com/a-h/templ/lsp/protocol"
	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
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

func RangeFromNode(n ast.Node) protocol.Range {
	token := n.GetToken()
	line := uint32(token.Position.Line) - 1
	character := uint32(token.Position.Column) - 1
	return protocol.Range{
		Start: protocol.Position{
			Line:      line,
			Character: character,
		},
		End: protocol.Position{
			Line:      line,
			Character: character + uint32(len(token.Origin)) - 1,
		},
	}
}
