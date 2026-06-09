package engine

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/matej/mutagen/pkg/mutator"
)

const resumeFileName = "resume.json"

type resumeState struct {
	Mutations []mutator.Mutation `json:"mutations"`
	Results   []mutator.Result   `json:"results"`
	Completed int                `json:"completed"`
}

func resumePath(cacheDir string) string {
	return filepath.Join(cacheDir, resumeFileName)
}

func LoadResume(cacheDir string) (*resumeState, error) {
	data, err := os.ReadFile(resumePath(cacheDir))
	if err != nil {
		return nil, err
	}
	var state resumeState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func SaveResume(cacheDir string, mutations []mutator.Mutation, results []mutator.Result, completed int) error {
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return err
	}
	state := resumeState{
		Mutations: mutations,
		Results:   results,
		Completed: completed,
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(resumePath(cacheDir), data, 0o644)
}

func ClearResume(cacheDir string) {
	os.Remove(resumePath(cacheDir))
}

func HasResume(cacheDir string) bool {
	_, err := os.Stat(resumePath(cacheDir))
	return err == nil
}

// PromptResume asks the user whether to continue from saved state.
// Returns true to resume, false to start fresh.
func PromptResume(cacheDir string, verbose bool) bool {
	state, err := LoadResume(cacheDir)
	if err != nil {
		return false
	}

	fmt.Fprintf(os.Stderr, "Found interrupted run: %d/%d mutations completed.\n", state.Completed, len(state.Mutations))
	fmt.Fprintf(os.Stderr, "Continue from where you left off? [Y/n] ")

	var answer string
	fmt.Scanln(&answer)

	if answer == "" || answer == "y" || answer == "Y" || answer == "yes" {
		return true
	}

	ClearResume(cacheDir)
	return false
}
