package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/matej/mutagen/pkg/mutator"
)

type Runner struct {
	cfg     Config
	fset    *token.FileSet
	tmpBase string
	testMap *TestMap
}

func NewRunner(cfg Config, fset *token.FileSet, testMap *TestMap) *Runner {
	return &Runner{cfg: cfg, fset: fset, testMap: testMap}
}

func (r *Runner) RunAll(mutations []mutator.Mutation) ([]mutator.Result, error) {
	tmpBase, err := os.MkdirTemp("", "mutagen-*")
	if err != nil {
		return nil, fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpBase)
	r.tmpBase = tmpBase

	workers := r.cfg.Workers
	if workers <= 0 {
		workers = runtime.NumCPU()
	}

	results := make([]mutator.Result, len(mutations))
	total := len(mutations)

	if workers == 1 {
		for i, mut := range mutations {
			if r.cfg.Verbose {
				fmt.Fprintf(os.Stderr, "\r\033[2K[%d/%d] Testing: %s", i+1, total, mut.Description)
			}
			results[i] = r.testMutation(mut, 0)
		}
		if r.cfg.Verbose {
			fmt.Fprintln(os.Stderr)
		}
		return results, nil
	}

	return r.runParallel(mutations, results, workers, total)
}

func (r *Runner) runParallel(mutations []mutator.Mutation, results []mutator.Result, workers, total int) ([]mutator.Result, error) {
	type work struct {
		index    int
		mutation mutator.Mutation
	}

	ch := make(chan work, len(mutations))
	for i, m := range mutations {
		ch <- work{index: i, mutation: m}
	}
	close(ch)

	var wg sync.WaitGroup
	var mu sync.Mutex
	completed := 0

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for item := range ch {
				result := r.testMutation(item.mutation, workerID)
				results[item.index] = result

				if r.cfg.Verbose {
					mu.Lock()
					completed++
					fmt.Fprintf(os.Stderr, "\r\033[2K[%d/%d] %s — %s", completed, total, item.mutation.Description, result.Status)
					mu.Unlock()
				}
			}
		}(w)
	}

	wg.Wait()
	if r.cfg.Verbose {
		fmt.Fprintln(os.Stderr)
	}

	return results, nil
}

func (r *Runner) testMutation(mut mutator.Mutation, workerID int) mutator.Result {
	start := time.Now()

	workerDir := filepath.Join(r.tmpBase, fmt.Sprintf("worker-%d", workerID))
	if err := os.MkdirAll(workerDir, 0o755); err != nil {
		return r.errResult(mut, start, "mkdir error: %v", err)
	}

	overlayPath, err := r.prepareMutatedOverlay(mut, workerDir)
	if err != nil {
		return r.errResult(mut, start, "%v", err)
	}

	return r.executeTest(mut, overlayPath, start)
}

func (r *Runner) prepareMutatedOverlay(mut mutator.Mutation, workerDir string) (string, error) {
	fset := token.NewFileSet()
	astFile, err := parser.ParseFile(fset, mut.File, nil, parser.ParseComments)
	if err != nil {
		return "", fmt.Errorf("parse error: %v", err)
	}

	if !applyMutationToAST(fset, astFile, mut) {
		return "", fmt.Errorf("could not locate mutation target in AST")
	}

	mutatedPath := filepath.Join(workerDir, filepath.Base(mut.File))
	if err := writeMutatedFile(fset, astFile, mutatedPath); err != nil {
		return "", fmt.Errorf("write error: %v", err)
	}

	overlayPath := filepath.Join(workerDir, "overlay.json")
	overlay := map[string]interface{}{
		"Replace": map[string]string{mut.File: mutatedPath},
	}
	overlayData, err := json.Marshal(overlay)
	if err != nil {
		return "", fmt.Errorf("overlay marshal error: %v", err)
	}
	if err := os.WriteFile(overlayPath, overlayData, 0o644); err != nil {
		return "", fmt.Errorf("overlay write error: %v", err)
	}

	return overlayPath, nil
}

func (r *Runner) executeTest(mut mutator.Mutation, overlayPath string, start time.Time) mutator.Result {
	timeout := r.cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	args := []string{"test", "-overlay=" + overlayPath, "-count=1", "-failfast"}
	if r.testMap != nil {
		if pattern := r.testMap.RunPattern(mut.File, mut.Line); pattern != "" {
			args = append(args, "-run", pattern)
		}
	} else if r.cfg.TestRun != "" {
		args = append(args, "-run", r.cfg.TestRun)
	}
	if r.cfg.TestTags != "" {
		args = append(args, "-tags", r.cfg.TestTags)
	}
	if r.cfg.Short {
		args = append(args, "-short")
	}
	args = append(args, mut.Package)

	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = filepath.Dir(mut.File)
	output, err := cmd.CombinedOutput()
	duration := time.Since(start).Seconds()

	if ctx.Err() == context.DeadlineExceeded {
		return mutator.Result{Mutation: mut, Status: mutator.StatusTimeout, Duration: duration, Output: string(output)}
	}

	if err != nil {
		outStr := string(output)
		if isBuildError(outStr) {
			return mutator.Result{Mutation: mut, Status: mutator.StatusBuildError, Duration: duration, Output: outStr}
		}
		return mutator.Result{Mutation: mut, Status: mutator.StatusKilled, Duration: duration, Output: outStr}
	}

	return mutator.Result{Mutation: mut, Status: mutator.StatusSurvived, Duration: duration, Output: string(output)}
}

func (r *Runner) errResult(mut mutator.Mutation, start time.Time, format string, args ...interface{}) mutator.Result {
	return mutator.Result{
		Mutation: mut,
		Status:   mutator.StatusBuildError,
		Duration: time.Since(start).Seconds(),
		Output:   fmt.Sprintf(format, args...),
	}
}
