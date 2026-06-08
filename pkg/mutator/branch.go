package mutator

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
)

func init() {
	Register(&BranchMutator{})
}

type BranchMutator struct{}

func (m *BranchMutator) Name() string { return "branch" }

func (m *BranchMutator) Mutate(fset *token.FileSet, file *ast.File, info *types.Info) []Mutation {
	var mutations []Mutation

	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.IfStmt:
			if node.Else != nil {
				pos := fset.Position(node.Else.Pos())
				mutations = append(mutations, Mutation{
					Type:        MutRemoveElse,
					File:        pos.Filename,
					Line:        pos.Line,
					Column:      pos.Column,
					Original:    "else { ... }",
					Replacement: "",
					Description: "removed else block",
				})
			}

		case *ast.CaseClause:
			if len(node.Body) == 0 {
				break
			}
			// Skip fallthrough-only cases
			if len(node.Body) == 1 {
				if bs, ok := node.Body[0].(*ast.BranchStmt); ok && bs.Tok == token.FALLTHROUGH {
					break
				}
			}

			pos := fset.Position(node.Pos())
			label := "case"
			if node.List == nil {
				label = "default"
			}
			mutations = append(mutations, Mutation{
				Type:        MutRemoveCase,
				File:        pos.Filename,
				Line:        pos.Line,
				Column:      pos.Column,
				Original:    fmt.Sprintf("%s: ...", label),
				Replacement: fmt.Sprintf("%s:", label),
				Description: fmt.Sprintf("removed %s body", label),
			})
		}

		return true
	})

	return mutations
}
