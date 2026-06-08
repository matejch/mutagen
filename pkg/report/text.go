package report

import (
	"fmt"
	"io"
	"strings"

	"github.com/matej/mutagen/pkg/mutator"
)

// WriteText writes a human-readable report to w.
func WriteText(w io.Writer, r Report) error {
	var err error
	p := func(format string, args ...interface{}) {
		if err != nil {
			return
		}
		_, err = fmt.Fprintf(w, format, args...)
	}

	p("\n")
	p("Mutation Testing Report\n")
	p("%s\n\n", strings.Repeat("=", 50))

	p("Total mutations:  %d\n", r.Summary.Total)
	p("Killed:           %d\n", r.Summary.Killed)
	p("Survived:         %d\n", r.Summary.Survived)
	p("Build errors:     %d\n", r.Summary.BuildError)
	p("Timeouts:         %d\n", r.Summary.Timeout)
	p("Skipped:          %d\n", r.Summary.Skipped)
	p("Kill rate:        %.1f%%\n", r.Summary.KillRate)
	p("Duration:         %.1fs\n", r.Summary.Duration)

	if len(r.Survivors) == 0 {
		p("\nAll mutations were killed. Your tests are strong.\n")
		return err
	}

	p("\n%s\n", strings.Repeat("-", 50))
	p("Surviving Mutations (%d)\n", len(r.Survivors))
	p("%s\n\n", strings.Repeat("-", 50))

	for i, res := range r.Survivors {
		p("%d. %s:%d\n", i+1, res.Mutation.File, res.Mutation.Line)
		p("   %s\n", res.Mutation.Description)
		p("   Package: %s\n\n", res.Mutation.Package)
	}

	return err
}

// WriteSurvivorSummary writes a compact one-line-per-survivor list.
func WriteSurvivorSummary(w io.Writer, results []mutator.Result) {
	for _, res := range results {
		if res.Status == mutator.StatusSurvived {
			fmt.Fprintf(w, "SURVIVED: %s\n", res.Mutation)
		}
	}
}
