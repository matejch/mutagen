package mutator

import (
	"go/ast"
	"go/token"
	"go/types"
)

func init() {
	Register(&NilCheckMutator{})
}

// NilCheckMutator removes `if err != nil { return ... }` guard blocks.
// Only targets the strict pattern: binary != nil check, no else, no init,
// single return statement in the body.
type NilCheckMutator struct{}

func (m *NilCheckMutator) Name() string { return "nilcheck" }

func (m *NilCheckMutator) Mutate(fset *token.FileSet, file *ast.File, info *types.Info) []Mutation {
	var mutations []Mutation

	ast.Inspect(file, func(n ast.Node) bool {
		ifStmt, ok := n.(*ast.IfStmt)
		if !ok {
			return true
		}

		if ifStmt.Else != nil || ifStmt.Init != nil {
			return true
		}

		binExpr, ok := ifStmt.Cond.(*ast.BinaryExpr)
		if !ok || binExpr.Op != token.NEQ {
			return true
		}

		rhsIdent, ok := binExpr.Y.(*ast.Ident)
		if !ok || rhsIdent.Name != "nil" {
			return true
		}

		if _, ok := binExpr.X.(*ast.Ident); !ok {
			return true
		}

		if len(ifStmt.Body.List) != 1 {
			return true
		}
		if _, ok := ifStmt.Body.List[0].(*ast.ReturnStmt); !ok {
			return true
		}

		pos := fset.Position(ifStmt.Pos())
		mutations = append(mutations, Mutation{
			Type:        MutNilCheck,
			File:        pos.Filename,
			Line:        pos.Line,
			Column:      pos.Column,
			Original:    "if err != nil { return ... }",
			Replacement: "if err != nil { }",
			Description: "removed nil check guard",
		})

		return true
	})

	return mutations
}
