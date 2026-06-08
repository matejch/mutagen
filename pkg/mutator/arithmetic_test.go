package mutator_test

import (
	"testing"

	"github.com/matej/mutagen/pkg/mutator"
)

func TestArithmeticMutator(t *testing.T) {
	m := &mutator.ArithmeticMutator{}

	t.Run("basic operators", func(t *testing.T) {
		src := `package p
func f(a, b int) int { return a + b }
`
		muts := findMutations(t, m, src)
		assertMutationCount(t, muts, 1)
		assertHasMutation(t, muts, "replaced + with -")
	})

	t.Run("all arithmetic ops", func(t *testing.T) {
		src := `package p
func f(a, b int) int {
	_ = a + b
	_ = a - b
	_ = a * b
	_ = a / b
	_ = a % b
	return 0
}
`
		muts := findMutations(t, m, src)
		assertMutationCount(t, muts, 5)
		assertHasMutation(t, muts, "replaced + with -")
		assertHasMutation(t, muts, "replaced - with +")
		assertHasMutation(t, muts, "replaced * with /")
		assertHasMutation(t, muts, "replaced / with *")
		assertHasMutation(t, muts, "replaced % with *")
	})

	t.Run("skips string concatenation", func(t *testing.T) {
		src := `package p
func f(a, b string) string { return a + b }
`
		muts := findMutations(t, m, src)
		assertMutationCount(t, muts, 0)
	})

	t.Run("nested in function literal", func(t *testing.T) {
		src := `package p
func f() int {
	fn := func() int { return 1 + 2 }
	return fn()
}
`
		muts := findMutations(t, m, src)
		assertMutationCount(t, muts, 1)
		assertHasMutation(t, muts, "replaced + with -")
	})

	t.Run("nested in if condition", func(t *testing.T) {
		src := `package p
func f(a, b int) bool {
	if a + b > 0 {
		return true
	}
	return false
}
`
		muts := findMutations(t, m, src)
		assertMutationCount(t, muts, 1)
		assertHasMutation(t, muts, "replaced + with -")
	})

	t.Run("deeply nested", func(t *testing.T) {
		src := `package p
func f() int {
	type S struct{ Val int }
	s := S{Val: 1 + 2}
	return s.Val
}
`
		muts := findMutations(t, m, src)
		assertMutationCount(t, muts, 1)
	})

	t.Run("no mutations in non-arithmetic", func(t *testing.T) {
		src := `package p
func f(a, b bool) bool { return a && b }
`
		muts := findMutations(t, m, src)
		assertMutationCount(t, muts, 0)
	})
}
