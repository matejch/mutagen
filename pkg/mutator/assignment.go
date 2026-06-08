package mutator

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
)

func init() {
	Register(&AssignmentMutator{})
}

type AssignmentMutator struct{}

func (m *AssignmentMutator) Name() string { return "assignment" }

var assignmentSwaps = map[token.Token]token.Token{
	token.ADD_ASSIGN: token.SUB_ASSIGN,
	token.SUB_ASSIGN: token.ADD_ASSIGN,
	token.MUL_ASSIGN: token.QUO_ASSIGN,
	token.QUO_ASSIGN: token.MUL_ASSIGN,
	token.REM_ASSIGN: token.MUL_ASSIGN,
}

func (m *AssignmentMutator) Mutate(fset *token.FileSet, file *ast.File, info *types.Info) []Mutation {
	var mutations []Mutation

	ast.Inspect(file, func(n ast.Node) bool {
		stmt, ok := n.(*ast.AssignStmt)
		if !ok {
			return true
		}

		repl, ok := assignmentSwaps[stmt.Tok]
		if !ok {
			return true
		}

		pos := fset.Position(stmt.TokPos)
		mutations = append(mutations, Mutation{
			Type:        MutAssignOp,
			File:        pos.Filename,
			Line:        pos.Line,
			Column:      pos.Column,
			Original:    stmt.Tok.String(),
			Replacement: repl.String(),
			Description: fmt.Sprintf("replaced %s with %s", stmt.Tok, repl),
		})

		return true
	})

	return mutations
}
