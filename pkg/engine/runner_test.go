package engine

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/matej/mutagen/pkg/mutator"
)

func TestApplyBinaryOpMutation(t *testing.T) {
	src := `package p

func f(a, b int) int {
	return a + b
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}

	mut := mutator.Mutation{
		Type:        mutator.MutBinaryOp,
		Line:        4,
		Column:      11,
		Original:    "+",
		Replacement: "-",
	}

	if !applyMutationToAST(fset, file, mut) {
		t.Fatal("failed to apply mutation")
	}

	// Verify the operator was changed
	found := false
	ast.Inspect(file, func(n ast.Node) bool {
		if expr, ok := n.(*ast.BinaryExpr); ok {
			if expr.Op == token.SUB {
				found = true
			}
		}
		return true
	})
	if !found {
		t.Error("operator was not changed to SUB")
	}
}

func TestApplyBoolLiteralMutation(t *testing.T) {
	src := `package p

func f() bool {
	return true
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}

	mut := mutator.Mutation{
		Type:        mutator.MutBoolLiteral,
		Line:        4,
		Column:      9,
		Original:    "true",
		Replacement: "false",
	}

	if !applyMutationToAST(fset, file, mut) {
		t.Fatal("failed to apply mutation")
	}

	found := false
	ast.Inspect(file, func(n ast.Node) bool {
		if ident, ok := n.(*ast.Ident); ok && ident.Name == "false" {
			found = true
		}
		return true
	})
	if !found {
		t.Error("true was not changed to false")
	}
}

func TestApplyNilCheckMutation(t *testing.T) {
	src := `package p

func f(err error) error {
	if err != nil {
		return err
	}
	return nil
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}

	mut := mutator.Mutation{
		Type:   mutator.MutNilCheck,
		Line:   4,
		Column: 2,
	}

	if !applyMutationToAST(fset, file, mut) {
		t.Fatal("failed to apply mutation")
	}

	// The if-body should now be empty
	found := false
	ast.Inspect(file, func(n ast.Node) bool {
		if ifStmt, ok := n.(*ast.IfStmt); ok {
			if len(ifStmt.Body.List) == 0 {
				found = true
			}
		}
		return true
	})
	if !found {
		t.Error("if-body was not emptied")
	}
}

func TestParseOperatorToken(t *testing.T) {
	tests := map[string]token.Token{
		"+":  token.ADD,
		"-":  token.SUB,
		"*":  token.MUL,
		"/":  token.QUO,
		"==": token.EQL,
		"!=": token.NEQ,
		"&&": token.LAND,
		"||": token.LOR,
		">>": token.SHR,
		"+=": token.ADD_ASSIGN,
		"-=": token.SUB_ASSIGN,
		"*=": token.MUL_ASSIGN,
		"/=": token.QUO_ASSIGN,
		"%=": token.REM_ASSIGN,
	}
	for s, want := range tests {
		got := parseOperatorToken(s)
		if got != want {
			t.Errorf("parseOperatorToken(%q) = %v, want %v", s, got, want)
		}
	}

	if parseOperatorToken("bogus") != token.ILLEGAL {
		t.Error("unknown operator should return ILLEGAL")
	}
}

func TestIsBuildError(t *testing.T) {
	tests := []struct {
		output string
		want   bool
	}{
		{"FAIL\tpkg [build failed]", true},
		{"[setup failed]", true},
		{"# pkg/foo\nfoo.go:10: undefined: x", true},
		{"--- FAIL: TestFoo (0.00s)\n    foo_test.go:10: expected 5", false},
		{"ok  \tpkg\t0.100s", false},
	}
	for _, tt := range tests {
		if got := isBuildError(tt.output); got != tt.want {
			t.Errorf("isBuildError(%q) = %v, want %v", tt.output[:min(40, len(tt.output))], got, tt.want)
		}
	}
}

func TestZeroExprFromString(t *testing.T) {
	tests := map[string]bool{
		"false": true,
		"0":     true,
		"0.0":   true,
		`""`:    true,
		"nil":   true,
		"bogus": false,
	}
	for input, wantNonNil := range tests {
		got := zeroExprFromString(input)
		if (got != nil) != wantNonNil {
			t.Errorf("zeroExprFromString(%q): got nil=%v, want nil=%v", input, got == nil, !wantNonNil)
		}
	}
}
