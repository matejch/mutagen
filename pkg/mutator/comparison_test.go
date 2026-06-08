package mutator_test

import (
	"testing"

	"github.com/matej/mutagen/pkg/mutator"
)

func TestComparisonMutator(t *testing.T) {
	m := &mutator.ComparisonMutator{}

	t.Run("greater than produces two mutations", func(t *testing.T) {
		src := `package p
func f(a, b int) bool { return a > b }
`
		muts := findMutations(t, m, src)
		assertMutationCount(t, muts, 2)
		assertHasMutation(t, muts, "replaced > with >=")
		assertHasMutation(t, muts, "replaced > with <")
	})

	t.Run("equality", func(t *testing.T) {
		src := `package p
func f(a, b int) bool { return a == b }
`
		muts := findMutations(t, m, src)
		assertMutationCount(t, muts, 1)
		assertHasMutation(t, muts, "replaced == with !=")
	})

	t.Run("all comparison ops", func(t *testing.T) {
		src := `package p
func f(a, b int) {
	_ = a > b
	_ = a >= b
	_ = a < b
	_ = a <= b
	_ = a == b
	_ = a != b
}
`
		muts := findMutations(t, m, src)
		// > -> >=, <  (2)
		// >= -> >, <= (2)
		// < -> <=, > (2)
		// <= -> <, >= (2)
		// == -> !=   (1)
		// != -> ==   (1)
		assertMutationCount(t, muts, 10)
	})

	t.Run("nested in composite literal", func(t *testing.T) {
		src := `package p
func f(x int) []bool {
	return []bool{x > 0, x < 10}
}
`
		muts := findMutations(t, m, src)
		// > produces 2 mutations, < produces 2 mutations
		assertMutationCount(t, muts, 4)
	})

	t.Run("nested in function literal", func(t *testing.T) {
		src := `package p
func f() func() bool {
	return func() bool { return 1 == 2 }
}
`
		muts := findMutations(t, m, src)
		assertMutationCount(t, muts, 1)
		assertHasMutation(t, muts, "replaced == with !=")
	})

	t.Run("ignores arithmetic", func(t *testing.T) {
		src := `package p
func f(a, b int) int { return a + b }
`
		muts := findMutations(t, m, src)
		assertMutationCount(t, muts, 0)
	})
}
