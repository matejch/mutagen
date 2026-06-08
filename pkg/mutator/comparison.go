package mutator

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
)

func init() {
	Register(&ComparisonMutator{})
}

// ComparisonMutator swaps comparison operators: >/>=, </<=, ==/!=.
type ComparisonMutator struct{}

func (m *ComparisonMutator) Name() string { return "comparison" }

var comparisonSwaps = map[token.Token][]token.Token{
	token.GTR: {token.GEQ, token.LSS},
	token.GEQ: {token.GTR, token.LEQ},
	token.LSS: {token.LEQ, token.GTR},
	token.LEQ: {token.LSS, token.GEQ},
	token.EQL: {token.NEQ},
	token.NEQ: {token.EQL},
}

func (m *ComparisonMutator) Mutate(fset *token.FileSet, file *ast.File, info *types.Info) []Mutation {
	var mutations []Mutation

	ast.Inspect(file, func(n ast.Node) bool {
		expr, ok := n.(*ast.BinaryExpr)
		if !ok {
			return true
		}

		replacements, ok := comparisonSwaps[expr.Op]
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
