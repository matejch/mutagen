package mutator_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"github.com/matej/mutagen/pkg/mutator"
)

// parseAndTypeCheck parses Go source and returns the AST, FileSet, and type info.
func parseAndTypeCheck(t *testing.T, src string) (*token.FileSet, *ast.File, *types.Info) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Uses:  make(map[*ast.Ident]types.Object),
		Defs:  make(map[*ast.Ident]types.Object),
	}
	conf := types.Config{Importer: nil}
	// Type errors are expected for some snippets (missing imports).
	// We still get partial type info which is enough for testing.
	conf.Check("test", fset, []*ast.File{file}, info) //nolint:errcheck

	return fset, file, info
}

func findMutations(t *testing.T, m mutator.Mutator, src string) []mutator.Mutation {
	t.Helper()
	fset, file, info := parseAndTypeCheck(t, src)
	return m.Mutate(fset, file, info)
}

func assertMutationCount(t *testing.T, mutations []mutator.Mutation, want int) {
	t.Helper()
	if got := len(mutations); got != want {
		t.Errorf("got %d mutations, want %d", got, want)
		for i, m := range mutations {
			t.Logf("  [%d] line %d: %s", i, m.Line, m.Description)
		}
	}
}

func assertHasMutation(t *testing.T, mutations []mutator.Mutation, desc string) {
	t.Helper()
	for _, m := range mutations {
		if m.Description == desc {
			return
		}
	}
	t.Errorf("missing mutation %q", desc)
	for _, m := range mutations {
		t.Logf("  got: %s", m.Description)
	}
}
