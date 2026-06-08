package mutator_test

import (
	"testing"

	"github.com/matej/mutagen/pkg/mutator"
)

func TestReturnValueMutator(t *testing.T) {
	m := &mutator.ReturnValueMutator{}

	t.Run("int return", func(t *testing.T) {
		src := `package p
func f() int { return 42 }
`
		muts := findMutations(t, m, src)
		assertMutationCount(t, muts, 1)
		assertHasMutation(t, muts, "replaced return value with 0")
	})

	t.Run("string return", func(t *testing.T) {
		src := `package p
func f() string { return "hello" }
`
		muts := findMutations(t, m, src)
		assertMutationCount(t, muts, 1)
		assertHasMutation(t, muts, `replaced return value with ""`)
	})

	t.Run("bool return", func(t *testing.T) {
		src := `package p
func f() bool { return true }
`
		muts := findMutations(t, m, src)
		assertMutationCount(t, muts, 1)
		assertHasMutation(t, muts, "replaced return value with false")
	})

	t.Run("skips already zero", func(t *testing.T) {
		src := `package p
func f() int { return 0 }
`
		muts := findMutations(t, m, src)
		assertMutationCount(t, muts, 0)
	})

	t.Run("skips nil return", func(t *testing.T) {
		src := `package p
func f() error { return nil }
`
		muts := findMutations(t, m, src)
		assertMutationCount(t, muts, 0)
	})

	t.Run("multi-return", func(t *testing.T) {
		src := `package p
func f() (int, string) { return 42, "hello" }
`
		muts := findMutations(t, m, src)
		assertMutationCount(t, muts, 2)
		assertHasMutation(t, muts, "replaced return value with 0")
		assertHasMutation(t, muts, `replaced return value with ""`)
	})

	t.Run("naked return ignored", func(t *testing.T) {
		src := `package p
func f() (x int) { return }
`
		muts := findMutations(t, m, src)
		assertMutationCount(t, muts, 0)
	})

	t.Run("nested in function literal", func(t *testing.T) {
		src := `package p
func f() func() int {
	return func() int { return 42 }
}
`
		muts := findMutations(t, m, src)
		// outer return (func literal → nil) + inner return (42 → 0)
		assertMutationCount(t, muts, 2)
		assertHasMutation(t, muts, "replaced return value with nil")
		assertHasMutation(t, muts, "replaced return value with 0")
	})

	t.Run("skips false return", func(t *testing.T) {
		src := `package p
func f() bool { return false }
`
		muts := findMutations(t, m, src)
		assertMutationCount(t, muts, 0)
	})

	t.Run("skips empty string return", func(t *testing.T) {
		src := `package p
func f() string { return "" }
`
		muts := findMutations(t, m, src)
		assertMutationCount(t, muts, 0)
	})

	t.Run("skips zero float return", func(t *testing.T) {
		src := `package p
func f() float64 { return 0.0 }
`
		muts := findMutations(t, m, src)
		assertMutationCount(t, muts, 0)
	})

	t.Run("does not skip non-zero int", func(t *testing.T) {
		src := `package p
func f() int { return 1 }
`
		muts := findMutations(t, m, src)
		assertMutationCount(t, muts, 1)
	})

	t.Run("does not skip non-empty string", func(t *testing.T) {
		src := `package p
func f() string { return "hello" }
`
		muts := findMutations(t, m, src)
		assertMutationCount(t, muts, 1)
	})

	t.Run("does not skip true", func(t *testing.T) {
		src := `package p
func f() bool { return true }
`
		muts := findMutations(t, m, src)
		assertMutationCount(t, muts, 1)
	})

	t.Run("nil info returns nothing", func(t *testing.T) {
		muts := m.Mutate(nil, nil, nil)
		if len(muts) != 0 {
			t.Errorf("expected 0 mutations with nil info, got %d", len(muts))
		}
	})
}
