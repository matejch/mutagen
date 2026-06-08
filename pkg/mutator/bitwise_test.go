package mutator_test

import (
	"testing"

	"github.com/matej/mutagen/pkg/mutator"
)

func TestBitwiseMutator(t *testing.T) {
	m := &mutator.BitwiseMutator{}

	t.Run("all bitwise ops", func(t *testing.T) {
		src := `package p
func f(a, b int) {
	_ = a & b
	_ = a | b
	_ = a ^ b
	_ = a << uint(b)
	_ = a >> uint(b)
}
`
		muts := findMutations(t, m, src)
		assertMutationCount(t, muts, 5)
		assertHasMutation(t, muts, "replaced & with |")
		assertHasMutation(t, muts, "replaced | with &")
		assertHasMutation(t, muts, "replaced ^ with &")
		assertHasMutation(t, muts, "replaced << with >>")
		assertHasMutation(t, muts, "replaced >> with <<")
	})

	t.Run("nested in function literal", func(t *testing.T) {
		src := `package p
func f() func(int, int) int {
	return func(a, b int) int { return a & b }
}
`
		muts := findMutations(t, m, src)
		assertMutationCount(t, muts, 1)
		assertHasMutation(t, muts, "replaced & with |")
	})

	t.Run("ignores arithmetic", func(t *testing.T) {
		src := `package p
func f(a, b int) int { return a + b }
`
		muts := findMutations(t, m, src)
		assertMutationCount(t, muts, 0)
	})

	t.Run("ignores logical", func(t *testing.T) {
		src := `package p
func f(a, b bool) bool { return a && b }
`
		muts := findMutations(t, m, src)
		assertMutationCount(t, muts, 0)
	})
}
