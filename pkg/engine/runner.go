package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
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

	// Set up cancellation on Ctrl+C
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	go func() {
		<-sigCh
		if r.cfg.Verbose {
			fmt.Fprintf(os.Stderr, "\n\033[2KInterrupted. Saving progress...\n")
		}
		cancel()
	}()
	defer signal.Stop(sigCh)

	results := make([]mutator.Result, len(mutations))
	var completed int64
	total := len(mutations)

	if workers == 1 {
		for i, mut := range mutations {
			if ctx.Err() != nil {
				break
			}
			if r.cfg.Verbose {
				fmt.Fprintf(os.Stderr, "\r\033[2K[%d/%d] Testing: %s", i+1, total, mut.Description)
			}
			results[i] = r.testMutation(mut, 0)
			atomic.AddInt64(&completed, 1)
		}
		if r.cfg.Verbose {
			fmt.Fprintln(os.Stderr)
		}
	} else {
		r.runParallel(ctx, mutations, results, &completed, workers, total)
	}

	done := int(atomic.LoadInt64(&completed))
	if ctx.Err() != nil && done < total {
		if r.cfg.Verbose {
			fmt.Fprintf(os.Stderr, "Completed %d/%d mutations before interrupt.\n", done, total)
		}
	}

	// Return only completed results
	return results[:done], nil
}

func (r *Runner) runParallel(ctx context.Context, mutations []mutator.Mutation, results []mutator.Result, completed *int64, workers, total int) {
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

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for item := range ch {
				if ctx.Err() != nil {
					return
				}
				result := r.testMutation(item.mutation, workerID)
				results[item.index] = result

				n := int(atomic.AddInt64(completed, 1))
				if r.cfg.Verbose {
					mu.Lock()
					fmt.Fprintf(os.Stderr, "\r\033[2K[%d/%d] %s — %s", n, total, item.mutation.Description, result.Status)
					mu.Unlock()
				}
			}
		}(w)
	}

	wg.Wait()
	if r.cfg.Verbose {
		fmt.Fprintln(os.Stderr)
	}
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
	if r.cfg.MemLimit != "" {
		cmd.Env = append(os.Environ(), "GOMEMLIMIT="+r.cfg.MemLimit)
	}

	// Start memory watchdog if limit is set
	var memKilled atomic.Bool
	if r.cfg.MemLimit != "" {
		if limitBytes, err := parseMemLimit(r.cfg.MemLimit); err == nil && limitBytes > 0 {
			var buf bytes.Buffer
			cmd.Stdout = &buf
			cmd.Stderr = &buf

			if startErr := cmd.Start(); startErr != nil {
				return r.errResult(mut, start, "start error: %v", startErr)
			}
			done := make(chan struct{})
			go watchMemory(cmd.Process.Pid, limitBytes, done, func() {
				memKilled.Store(true)
			})
			waitErr := cmd.Wait()
			close(done)

			duration := time.Since(start).Seconds()
			outStr := buf.String()

			if memKilled.Load() {
				return mutator.Result{
					Mutation: mut,
					Status:   mutator.StatusTimeout,
					Duration: duration,
					Output:   fmt.Sprintf("killed: exceeded memory limit %s\n%s", r.cfg.MemLimit, outStr),
				}
			}
			if ctx.Err() == context.DeadlineExceeded {
				return mutator.Result{Mutation: mut, Status: mutator.StatusTimeout, Duration: duration, Output: outStr}
			}
			if waitErr != nil {
				if isBuildError(outStr) {
					return mutator.Result{Mutation: mut, Status: mutator.StatusBuildError, Duration: duration, Output: outStr}
				}
				return mutator.Result{Mutation: mut, Status: mutator.StatusKilled, Duration: duration, Output: outStr}
			}
			return mutator.Result{Mutation: mut, Status: mutator.StatusSurvived, Duration: duration, Output: outStr}
		}
	}

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
