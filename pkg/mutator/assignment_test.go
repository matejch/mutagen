package mutator_test

import (
	"testing"

	"github.com/matej/mutagen/pkg/mutator"
)

func TestAssignmentMutator(t *testing.T) {
	m := &mutator.AssignmentMutator{}

	t.Run("add assign to sub assign", func(t *testing.T) {
		src := `package p
func f() { x := 0; x += 1 }
`
		muts := findMutations(t, m, src)
		assertMutationCount(t, muts, 1)
		assertHasMutation(t, muts, "replaced += with -=")
	})

	t.Run("all compound assignments", func(t *testing.T) {
		src := `package p
func f() {
	x := 0
	x += 1
	x -= 1
	x *= 2
	x /= 2
	x %= 3
}
`
		muts := findMutations(t, m, src)
		assertMutationCount(t, muts, 5)
		assertHasMutation(t, muts, "replaced += with -=")
		assertHasMutation(t, muts, "replaced -= with +=")
		assertHasMutation(t, muts, "replaced *= with /=")
		assertHasMutation(t, muts, "replaced /= with *=")
		assertHasMutation(t, muts, "replaced %= with *=")
	})

	t.Run("ignores plain assign", func(t *testing.T) {
		src := `package p
func f() { x := 0; x = 1; _ = x }
`
		muts := findMutations(t, m, src)
		assertMutationCount(t, muts, 0)
	})

	t.Run("nested in for loop", func(t *testing.T) {
		src := `package p
func f() int {
	x := 0
	for i := 0; i < 10; i++ {
		x += i
	}
	return x
}
`
		muts := findMutations(t, m, src)
		assertMutationCount(t, muts, 1)
		assertHasMutation(t, muts, "replaced += with -=")
	})

	t.Run("ignores short var decl", func(t *testing.T) {
		src := `package p
func f() { x := 1; _ = x }
`
		muts := findMutations(t, m, src)
		assertMutationCount(t, muts, 0)
	})
}
