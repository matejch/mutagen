package mutator_test

import (
	"testing"

	"github.com/matej/mutagen/pkg/mutator"
)

func TestBranchMutator(t *testing.T) {
	m := &mutator.BranchMutator{}

	t.Run("removes else block", func(t *testing.T) {
		src := `package p
func f(x int) int {
	if x > 0 {
		return 1
	} else {
		return -1
	}
}
`
		muts := findMutations(t, m, src)
		assertMutationCount(t, muts, 1)
		assertHasMutation(t, muts, "removed else block")
	})

	t.Run("no else no mutation", func(t *testing.T) {
		src := `package p
func f(x int) int {
	if x > 0 {
		return 1
	}
	return 0
}
`
		muts := findMutations(t, m, src)
		assertMutationCount(t, muts, 0)
	})

	t.Run("else-if chain", func(t *testing.T) {
		src := `package p
func f(x int) int {
	if x > 0 {
		return 1
	} else if x < 0 {
		return -1
	} else {
		return 0
	}
}
`
		muts := findMutations(t, m, src)
		// Two else blocks: "else if" and final "else"
		count := 0
		for _, mut := range muts {
			if mut.Description == "removed else block" {
				count++
			}
		}
		if count != 2 {
			t.Errorf("expected 2 else removals, got %d", count)
		}
	})

	t.Run("switch case removal", func(t *testing.T) {
		src := `package p
func f(x int) string {
	switch x {
	case 1:
		return "one"
	case 2:
		return "two"
	default:
		return "other"
	}
}
`
		muts := findMutations(t, m, src)
		caseCount := 0
		for _, mut := range muts {
			if mut.Description == "removed case body" || mut.Description == "removed default body" {
				caseCount++
			}
		}
		if caseCount != 3 {
			t.Errorf("expected 3 case body removals, got %d", caseCount)
			for _, mut := range muts {
				t.Logf("  %s", mut.Description)
			}
		}
	})

	t.Run("empty case no mutation", func(t *testing.T) {
		src := `package p
func f(x int) {
	switch x {
	case 1:
	case 2:
		return
	}
}
`
		muts := findMutations(t, m, src)
		// case 1 is empty (0 mutations), case 2 has body (1 mutation)
		caseCount := 0
		for _, mut := range muts {
			if mut.Description == "removed case body" {
				caseCount++
			}
		}
		if caseCount != 1 {
			t.Errorf("expected 1 case body removal, got %d", caseCount)
		}
	})

	t.Run("nested switch in function literal", func(t *testing.T) {
		src := `package p
func f() func(int) string {
	return func(x int) string {
		switch x {
		case 1:
			return "one"
		default:
			return "other"
		}
	}
}
`
		muts := findMutations(t, m, src)
		caseCount := 0
		for _, mut := range muts {
			if mut.Description == "removed case body" || mut.Description == "removed default body" {
				caseCount++
			}
		}
		if caseCount != 2 {
			t.Errorf("expected 2 case body removals in nested switch, got %d", caseCount)
		}
	})

	t.Run("skip fallthrough-only case", func(t *testing.T) {
		src := `package p
func f(x int) int {
	switch x {
	case 1:
		fallthrough
	case 2:
		return 2
	}
	return 0
}
`
		muts := findMutations(t, m, src)
		caseCount := 0
		for _, mut := range muts {
			if mut.Description == "removed case body" {
				caseCount++
			}
		}
		// case 1 is fallthrough-only (skipped), case 2 has real body (1 mutation)
		if caseCount != 1 {
			t.Errorf("expected 1 case body removal (skip fallthrough), got %d", caseCount)
		}
	})
}
