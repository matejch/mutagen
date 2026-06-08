package mutator

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
)

func init() {
	Register(&ArithmeticMutator{})
}

// ArithmeticMutator swaps arithmetic operators: +/-, *//, %/*.
type ArithmeticMutator struct{}

func (m *ArithmeticMutator) Name() string { return "arithmetic" }

var arithmeticSwaps = map[token.Token][]token.Token{
	token.ADD: {token.SUB},
	token.SUB: {token.ADD},
	token.MUL: {token.QUO},
	token.QUO: {token.MUL},
	token.REM: {token.MUL},
}

func (m *ArithmeticMutator) Mutate(fset *token.FileSet, file *ast.File, info *types.Info) []Mutation {
	var mutations []Mutation

	ast.Inspect(file, func(n ast.Node) bool {
		expr, ok := n.(*ast.BinaryExpr)
		if !ok {
			return true
		}

		replacements, ok := arithmeticSwaps[expr.Op]
		if !ok {
			return true
		}

		// Skip string concatenation
		if info != nil {
			if t := info.TypeOf(expr.X); t != nil {
				if basic, ok := t.Underlying().(*types.Basic); ok {
					if basic.Info()&types.IsString != 0 {
						return true
					}
				}
			}
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
