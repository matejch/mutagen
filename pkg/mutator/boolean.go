package mutator

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
)

func init() {
	Register(&BooleanMutator{})
}

// BooleanMutator swaps true/false boolean literals.
type BooleanMutator struct{}

func (m *BooleanMutator) Name() string { return "boolean" }

func (m *BooleanMutator) Mutate(fset *token.FileSet, file *ast.File, info *types.Info) []Mutation {
	var mutations []Mutation

	ast.Inspect(file, func(n ast.Node) bool {
		ident, ok := n.(*ast.Ident)
		if !ok {
			return true
		}

		if ident.Name != "true" && ident.Name != "false" {
			return true
		}

		// Verify it's a builtin boolean constant, not a user-defined identifier.
		// In go/types, builtin true/false are *types.Const in types.Universe scope.
		if info != nil {
			obj, exists := info.Uses[ident]
			if !exists {
				// Not in Uses map — might be a definition, skip it
				return true
			}
			// Must be a Const whose parent scope is Universe (builtin)
			if obj.Parent() != types.Universe {
				return true
			}
		}

		var repl string
		if ident.Name == "true" {
			repl = "false"
		} else {
			repl = "true"
		}

		pos := fset.Position(ident.Pos())
		mutations = append(mutations, Mutation{
			Type:        MutBoolLiteral,
			File:        pos.Filename,
			Line:        pos.Line,
			Column:      pos.Column,
			Original:    ident.Name,
			Replacement: repl,
			Description: fmt.Sprintf("replaced %s with %s", ident.Name, repl),
		})

		return true
	})

	return mutations
}
