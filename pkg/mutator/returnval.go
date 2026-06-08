package mutator

import (
	"go/ast"
	"go/token"
	"go/types"
)

func init() {
	Register(&ReturnValueMutator{})
}

// ReturnValueMutator replaces return values with their zero values.
// Only targets return expressions whose types have a known zero value.
type ReturnValueMutator struct{}

func (m *ReturnValueMutator) Name() string { return "returnval" }

func (m *ReturnValueMutator) Mutate(fset *token.FileSet, file *ast.File, info *types.Info) []Mutation {
	if info == nil {
		return nil
	}

	var mutations []Mutation

	ast.Inspect(file, func(n ast.Node) bool {
		ret, ok := n.(*ast.ReturnStmt)
		if !ok || len(ret.Results) == 0 {
			return true
		}

		for _, expr := range ret.Results {
			t := info.TypeOf(expr)
			if t == nil {
				continue
			}

			zeroStr := zeroValueString(t)
			if zeroStr == "" {
				continue
			}

			if isZeroLiteral(expr, t) {
				continue
			}

			pos := fset.Position(expr.Pos())
			mutations = append(mutations, Mutation{
				Type:        MutReturnValue,
				File:        pos.Filename,
				Line:        pos.Line,
				Column:      pos.Column,
				Original:    "return value",
				Replacement: zeroStr,
				Description: "replaced return value with " + zeroStr,
			})
		}

		return true
	})

	return mutations
}

func zeroValueString(t types.Type) string {
	switch u := t.Underlying().(type) {
	case *types.Basic:
		switch {
		case u.Info()&types.IsBoolean != 0:
			return "false"
		case u.Info()&types.IsInteger != 0:
			return "0"
		case u.Info()&types.IsFloat != 0:
			return "0.0"
		case u.Info()&types.IsString != 0:
			return `""`
		}
	case *types.Pointer, *types.Slice, *types.Map, *types.Chan, *types.Interface, *types.Signature:
		return "nil"
	}
	return ""
}

func isZeroLiteral(expr ast.Expr, t types.Type) bool {
	switch e := expr.(type) {
	case *ast.BasicLit:
		switch e.Kind {
		case token.INT:
			return e.Value == "0"
		case token.FLOAT:
			return e.Value == "0.0" || e.Value == "0"
		case token.STRING:
			return e.Value == `""`
		}
	case *ast.Ident:
		return e.Name == "nil" || e.Name == "false"
	}
	return false
}
