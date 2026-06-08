package mutator_test

import (
	"testing"

	"github.com/matej/mutagen/pkg/mutator"
)

func TestLogicalMutator(t *testing.T) {
	m := &mutator.LogicalMutator{}

	t.Run("and to or", func(t *testing.T) {
		src := `package p
func f(a, b bool) bool { return a && b }
`
		muts := findMutations(t, m, src)
		assertMutationCount(t, muts, 1)
		assertHasMutation(t, muts, "replaced && with ||")
	})

	t.Run("or to and", func(t *testing.T) {
		src := `package p
func f(a, b bool) bool { return a || b }
`
		muts := findMutations(t, m, src)
		assertMutationCount(t, muts, 1)
		assertHasMutation(t, muts, "replaced || with &&")
	})

	t.Run("nested in if condition", func(t *testing.T) {
		src := `package p
func f(a, b, c bool) bool {
	if a && (b || c) {
		return true
	}
	return false
}
`
		muts := findMutations(t, m, src)
		assertMutationCount(t, muts, 2)
		assertHasMutation(t, muts, "replaced && with ||")
		assertHasMutation(t, muts, "replaced || with &&")
	})

	t.Run("ignores non-logical ops", func(t *testing.T) {
		src := `package p
func f(a, b int) bool { return a > b }
`
		muts := findMutations(t, m, src)
		assertMutationCount(t, muts, 0)
	})
}
