package mutator

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
)

func init() {
	Register(&LogicalMutator{})
}

// LogicalMutator swaps logical operators: && to || and || to &&.
type LogicalMutator struct{}

func (m *LogicalMutator) Name() string { return "logical" }

var logicalSwaps = map[token.Token]token.Token{
	token.LAND: token.LOR,
	token.LOR:  token.LAND,
}

func (m *LogicalMutator) Mutate(fset *token.FileSet, file *ast.File, info *types.Info) []Mutation {
	var mutations []Mutation

	ast.Inspect(file, func(n ast.Node) bool {
		expr, ok := n.(*ast.BinaryExpr)
		if !ok {
			return true
		}

		repl, ok := logicalSwaps[expr.Op]
		if !ok {
			return true
		}

		pos := fset.Position(expr.OpPos)
		mutations = append(mutations, Mutation{
			Type:        MutBinaryOp,
			File:        pos.Filename,
			Line:        pos.Line,
			Column:      pos.Column,
			Original:    expr.Op.String(),
			Replacement: repl.String(),
			Description: fmt.Sprintf("replaced %s with %s", expr.Op, repl),
		})

		return true
	})

	return mutations
}
