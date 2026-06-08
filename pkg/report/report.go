package report

import (
	"github.com/matej/mutagen/pkg/mutator"
)

// Summary aggregates mutation testing results.
type Summary struct {
	Total      int     `json:"total"`
	Killed     int     `json:"killed"`
	Survived   int     `json:"survived"`
	BuildError int     `json:"build_errors"`
	Timeout    int     `json:"timeouts"`
	Skipped    int     `json:"skipped"`
	KillRate   float64 `json:"kill_rate"` // Killed / (Killed + Survived)
	Duration   float64 `json:"duration_seconds"`
}

// Report holds the full mutation testing output.
type Report struct {
	Summary   Summary          `json:"summary"`
	Results   []mutator.Result `json:"results"`
	Survivors []mutator.Result `json:"survivors"` // Convenience: only survived mutations
}

func Build(results []mutator.Result) Report {
	var r Report
	r.Results = results

	for _, res := range results {
		r.Summary.Duration += res.Duration
		switch res.Status {
		case mutator.StatusKilled:
			r.Summary.Killed++
		case mutator.StatusSurvived:
			r.Summary.Survived++
			r.Survivors = append(r.Survivors, res)
		case mutator.StatusBuildError:
			r.Summary.BuildError++
		case mutator.StatusTimeout:
			r.Summary.Timeout++
		case mutator.StatusSkipped:
			r.Summary.Skipped++
		}
	}

	r.Summary.Total = len(results)
	tested := r.Summary.Killed + r.Summary.Survived
	if tested > 0 {
		r.Summary.KillRate = float64(r.Summary.Killed) / float64(tested) * 100
	}

	return r
}
