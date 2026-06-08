package mutator

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
)

func init() {
	Register(&BitwiseMutator{})
}

type BitwiseMutator struct{}

func (m *BitwiseMutator) Name() string { return "bitwise" }

var bitwiseSwaps = map[token.Token][]token.Token{
	token.AND: {token.OR},
	token.OR:  {token.AND},
	token.XOR: {token.AND},
	token.SHL: {token.SHR},
	token.SHR: {token.SHL},
}

func (m *BitwiseMutator) Mutate(fset *token.FileSet, file *ast.File, info *types.Info) []Mutation {
	var mutations []Mutation

	ast.Inspect(file, func(n ast.Node) bool {
		expr, ok := n.(*ast.BinaryExpr)
		if !ok {
			return true
		}

		replacements, ok := bitwiseSwaps[expr.Op]
		if !ok {
			return true
		}

		for _, replacement := range replacements {
			pos := fset.Position(expr.OpPos)
			mutations = append(mutations, Mutation{
				Type:        MutBinaryOp,
				File:        pos.Filename,
				Line:        pos.Line,
				Column:      pos.Column,
				Original:    expr.Op.String(),
				Replacement: replacement.String(),
				Description: fmt.Sprintf("replaced %s with %s", expr.Op, replacement),
			})
		}
		return true
	})

	return mutations
}
