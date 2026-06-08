package mutator_test

import (
	"testing"

	"github.com/matej/mutagen/pkg/mutator"
)

func TestBooleanMutator(t *testing.T) {
	m := &mutator.BooleanMutator{}

	t.Run("true to false", func(t *testing.T) {
		src := `package p
func f() bool { return true }
`
		muts := findMutations(t, m, src)
		assertMutationCount(t, muts, 1)
		assertHasMutation(t, muts, "replaced true with false")
	})

	t.Run("false to true", func(t *testing.T) {
		src := `package p
func f() bool { return false }
`
		muts := findMutations(t, m, src)
		assertMutationCount(t, muts, 1)
		assertHasMutation(t, muts, "replaced false with true")
	})

	t.Run("multiple booleans", func(t *testing.T) {
		src := `package p
func f() (bool, bool) { return true, false }
`
		muts := findMutations(t, m, src)
		assertMutationCount(t, muts, 2)
	})

	t.Run("nested in struct literal", func(t *testing.T) {
		src := `package p
type Config struct{ Enabled bool }
func f() Config { return Config{Enabled: true} }
`
		muts := findMutations(t, m, src)
		assertMutationCount(t, muts, 1)
		assertHasMutation(t, muts, "replaced true with false")
	})

	t.Run("no booleans", func(t *testing.T) {
		src := `package p
func f() int { return 42 }
`
		muts := findMutations(t, m, src)
		assertMutationCount(t, muts, 0)
	})
}
