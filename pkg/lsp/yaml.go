package lsp

import "github.com/goccy/go-yaml/ast"

// NodePathVisitor is a custom visitor to find the path to a YAML node based on position
type NodePathVisitor struct {
	TargetLine   int
	TargetColumn int
	FoundNode    ast.Node
	RootNode     ast.Node
}

// Visit is called for each AST node during traversal
func (v *NodePathVisitor) Visit(node ast.Node) ast.Visitor {
	// Check if the position overlaps with this node
	tkn := node.GetToken()
	start := tkn.Position

	if start.Line <= v.TargetLine && start.Column <= v.TargetColumn {
		switch n := node.(type) {
		case *ast.CommentNode:
			v.FoundNode = n
		case *ast.NullNode:
			v.FoundNode = n
		case *ast.IntegerNode:
			v.FoundNode = n
		case *ast.FloatNode:
			v.FoundNode = n
		case *ast.StringNode:
			v.FoundNode = n
		case *ast.MergeKeyNode:
			v.FoundNode = n
		case *ast.BoolNode:
			v.FoundNode = n
		case *ast.InfinityNode:
			v.FoundNode = n
		case *ast.NanNode:
			v.FoundNode = n
		case *ast.LiteralNode:
			v.FoundNode = n
		}
	}
	return v
}
