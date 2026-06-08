package engine

import (
	"go/ast"
	"go/printer"
	"go/token"
	"os"
	"strings"

	"github.com/matej/mutagen/pkg/mutator"
)

func applyMutationToAST(fset *token.FileSet, file *ast.File, mut mutator.Mutation) bool {
	switch mut.Type {
	case mutator.MutBinaryOp:
		return applyBinaryOpMutation(fset, file, mut)
	case mutator.MutBoolLiteral:
		return applyBoolLiteralMutation(fset, file, mut)
	case mutator.MutNilCheck:
		return applyNilCheckMutation(fset, file, mut)
	case mutator.MutReturnValue:
		return applyReturnValueMutation(fset, file, mut)
	case mutator.MutAssignOp:
		return applyAssignOpMutation(fset, file, mut)
	case mutator.MutRemoveElse:
		return applyRemoveElseMutation(fset, file, mut)
	case mutator.MutRemoveCase:
		return applyRemoveCaseMutation(fset, file, mut)
	default:
		return false
	}
}

func applyBinaryOpMutation(fset *token.FileSet, file *ast.File, mut mutator.Mutation) bool {
	found := false
	ast.Inspect(file, func(n ast.Node) bool {
		if found {
			return false
		}
		expr, ok := n.(*ast.BinaryExpr)
		if !ok {
			return true
		}
		pos := fset.Position(expr.OpPos)
		if pos.Line == mut.Line && pos.Column == mut.Column && expr.Op.String() == mut.Original {
			expr.Op = parseOperatorToken(mut.Replacement)
			found = expr.Op != token.ILLEGAL
			return false
		}
		return true
	})
	return found
}

func applyBoolLiteralMutation(fset *token.FileSet, file *ast.File, mut mutator.Mutation) bool {
	found := false
	ast.Inspect(file, func(n ast.Node) bool {
		if found {
			return false
		}
		ident, ok := n.(*ast.Ident)
		if !ok {
			return true
		}
		pos := fset.Position(ident.Pos())
		if pos.Line == mut.Line && pos.Column == mut.Column && ident.Name == mut.Original {
			ident.Name = mut.Replacement
			found = true
			return false
		}
		return true
	})
	return found
}

func applyNilCheckMutation(fset *token.FileSet, file *ast.File, mut mutator.Mutation) bool {
	found := false
	ast.Inspect(file, func(n ast.Node) bool {
		if found {
			return false
		}
		ifStmt, ok := n.(*ast.IfStmt)
		if !ok {
			return true
		}
		pos := fset.Position(ifStmt.Pos())
		if pos.Line == mut.Line && pos.Column == mut.Column {
			ifStmt.Body.List = nil
			found = true
			return false
		}
		return true
	})
	return found
}

func applyReturnValueMutation(fset *token.FileSet, file *ast.File, mut mutator.Mutation) bool {
	found := false
	ast.Inspect(file, func(n ast.Node) bool {
		if found {
			return false
		}
		ret, ok := n.(*ast.ReturnStmt)
		if !ok {
			return true
		}
		for i, expr := range ret.Results {
			pos := fset.Position(expr.Pos())
			if pos.Line == mut.Line && pos.Column == mut.Column {
				if zeroExpr := zeroExprFromString(mut.Replacement); zeroExpr != nil {
					ret.Results[i] = zeroExpr
					found = true
				}
				return false
			}
		}
		return true
	})
	return found
}

func applyAssignOpMutation(fset *token.FileSet, file *ast.File, mut mutator.Mutation) bool {
	found := false
	ast.Inspect(file, func(n ast.Node) bool {
		if found {
			return false
		}
		stmt, ok := n.(*ast.AssignStmt)
		if !ok {
			return true
		}
		pos := fset.Position(stmt.TokPos)
		if pos.Line == mut.Line && pos.Column == mut.Column && stmt.Tok.String() == mut.Original {
			stmt.Tok = parseOperatorToken(mut.Replacement)
			found = stmt.Tok != token.ILLEGAL
			return false
		}
		return true
	})
	return found
}

func applyRemoveElseMutation(fset *token.FileSet, file *ast.File, mut mutator.Mutation) bool {
	found := false
	ast.Inspect(file, func(n ast.Node) bool {
		if found {
			return false
		}
		ifStmt, ok := n.(*ast.IfStmt)
		if !ok || ifStmt.Else == nil {
			return true
		}
		pos := fset.Position(ifStmt.Else.Pos())
		if pos.Line == mut.Line && pos.Column == mut.Column {
			ifStmt.Else = nil
			found = true
			return false
		}
		return true
	})
	return found
}

func applyRemoveCaseMutation(fset *token.FileSet, file *ast.File, mut mutator.Mutation) bool {
	found := false
	ast.Inspect(file, func(n ast.Node) bool {
		if found {
			return false
		}
		cc, ok := n.(*ast.CaseClause)
		if !ok {
			return true
		}
		pos := fset.Position(cc.Pos())
		if pos.Line == mut.Line && pos.Column == mut.Column {
			cc.Body = nil
			found = true
			return false
		}
		return true
	})
	return found
}

func zeroExprFromString(s string) ast.Expr {
	switch s {
	case "false":
		return &ast.Ident{Name: "false"}
	case "0":
		return &ast.BasicLit{Kind: token.INT, Value: "0"}
	case "0.0":
		return &ast.BasicLit{Kind: token.FLOAT, Value: "0.0"}
	case `""`:
		return &ast.BasicLit{Kind: token.STRING, Value: `""`}
	case "nil":
		return &ast.Ident{Name: "nil"}
	default:
		return nil
	}
}

var operatorTokens = map[string]token.Token{
	"+": token.ADD, "-": token.SUB, "*": token.MUL, "/": token.QUO, "%": token.REM,
	">": token.GTR, ">=": token.GEQ, "<": token.LSS, "<=": token.LEQ,
	"==": token.EQL, "!=": token.NEQ,
	"&&": token.LAND, "||": token.LOR,
	"&": token.AND, "|": token.OR, "^": token.XOR, "<<": token.SHL, ">>": token.SHR,
	"+=": token.ADD_ASSIGN, "-=": token.SUB_ASSIGN,
	"*=": token.MUL_ASSIGN, "/=": token.QUO_ASSIGN, "%=": token.REM_ASSIGN,
}

func parseOperatorToken(s string) token.Token {
	if t, ok := operatorTokens[s]; ok {
		return t
	}
	return token.ILLEGAL
}

// isBuildError distinguishes compilation failures from test failures.
// `go test` outputs "[build failed]" in its FAIL line when compilation fails.
func isBuildError(output string) bool {
	if strings.Contains(output, "[build failed]") {
		return true
	}
	if strings.Contains(output, "[setup failed]") {
		return true
	}
	if strings.Contains(output, "# ") && strings.Contains(output, ".go:") &&
		!strings.Contains(output, "--- FAIL:") {
		return true
	}
	return false
}

func writeMutatedFile(fset *token.FileSet, file *ast.File, dest string) error {
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	cfg := printer.Config{Tabwidth: 8}
	return cfg.Fprint(f, fset, file)
}
