package mutator_test

import (
	"testing"

	"github.com/matej/mutagen/pkg/mutator"
)

func TestNilCheckMutator(t *testing.T) {
	m := &mutator.NilCheckMutator{}

	t.Run("standard error check", func(t *testing.T) {
		src := `package p
import "errors"
func f() error {
	err := errors.New("x")
	if err != nil {
		return err
	}
	return nil
}
`
		muts := findMutations(t, m, src)
		assertMutationCount(t, muts, 1)
		assertHasMutation(t, muts, "removed nil check guard")
	})

	t.Run("skips multi-statement body", func(t *testing.T) {
		src := `package p
import "fmt"
func f(err error) {
	if err != nil {
		fmt.Println(err)
		return
	}
}
`
		muts := findMutations(t, m, src)
		assertMutationCount(t, muts, 0)
	})

	t.Run("skips else branch", func(t *testing.T) {
		src := `package p
func f(err error) int {
	if err != nil {
		return 1
	} else {
		return 0
	}
}
`
		muts := findMutations(t, m, src)
		assertMutationCount(t, muts, 0)
	})

	t.Run("skips init statement", func(t *testing.T) {
		src := `package p
import "errors"
func f() error {
	if err := errors.New("x"); err != nil {
		return err
	}
	return nil
}
`
		muts := findMutations(t, m, src)
		assertMutationCount(t, muts, 0)
	})

	t.Run("nested in function literal", func(t *testing.T) {
		src := `package p
import "errors"
func f() func() error {
	return func() error {
		err := errors.New("x")
		if err != nil {
			return err
		}
		return nil
	}
}
`
		muts := findMutations(t, m, src)
		assertMutationCount(t, muts, 1)
		assertHasMutation(t, muts, "removed nil check guard")
	})

	t.Run("skips equality check", func(t *testing.T) {
		src := `package p
func f(err error) error {
	if err == nil {
		return err
	}
	return nil
}
`
		muts := findMutations(t, m, src)
		assertMutationCount(t, muts, 0)
	})
}
