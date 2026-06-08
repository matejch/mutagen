package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/matej/mutagen/pkg/engine"
	_ "github.com/matej/mutagen/pkg/mutator"
	"github.com/matej/mutagen/pkg/report"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(2)
	}
}

func run() error {
	cfg, exitCode := parseConfig()
	if exitCode >= 0 {
		os.Exit(exitCode)
	}

	eng := engine.New(cfg)
	results, err := eng.Run()
	if err != nil {
		return err
	}
	if results == nil {
		return nil
	}

	r := report.Build(results)

	if err := writeOutput(cfg, r); err != nil {
		return err
	}

	if cfg.Threshold > 0 && r.Summary.KillRate < cfg.Threshold {
		fmt.Fprintf(os.Stderr, "Kill rate %.1f%% is below threshold %.1f%%\n", r.Summary.KillRate, cfg.Threshold)
		os.Exit(1)
	}

	return nil
}

func writeOutput(cfg engine.Config, r report.Report) error {
	switch strings.ToLower(cfg.Output) {
	case "json":
		if err := report.WriteJSON(os.Stdout, r); err != nil {
			return fmt.Errorf("writing JSON: %w", err)
		}
	default:
		if err := report.WriteText(os.Stdout, r); err != nil {
			return fmt.Errorf("writing report: %w", err)
		}
	}

	if cfg.HTMLOutput != "" {
		if err := report.WriteHTMLFile(cfg.HTMLOutput, r); err != nil {
			return fmt.Errorf("writing HTML report: %w", err)
		}
		if cfg.Verbose {
			fmt.Fprintf(os.Stderr, "HTML report written to %s\n", cfg.HTMLOutput)
		}
	}

	return nil
}

// parseConfig returns the engine config. exitCode is -1 if the caller should
// continue normally, or >= 0 if the caller should exit with that code.
func parseConfig() (engine.Config, int) {
	var (
		workers      int
		timeout      time.Duration
		threshold    float64
		output       string
		htmlOutput   string
		cacheDir     string
		noCache      bool
		verbose      bool
		exclude      stringSlice
		enableMut    stringSlice
		coverprofile string
		diffBase     string
		perTest      bool
		configFile   string
	)

	flag.IntVar(&workers, "workers", 0, "number of parallel workers (default: NumCPU)")
	flag.DurationVar(&timeout, "timeout", 30*time.Second, "per-mutant test timeout")
	flag.Float64Var(&threshold, "threshold", 0, "minimum kill rate percentage to pass (e.g., 80.0)")
	flag.StringVar(&output, "output", "text", "output format: text, json, or html")
	flag.StringVar(&htmlOutput, "html", "", "write HTML report to this file")
	flag.StringVar(&cacheDir, "cache-dir", ".mutagen-cache", "cache directory for incremental mode")
	flag.BoolVar(&noCache, "no-cache", false, "disable incremental cache")
	flag.BoolVar(&verbose, "v", false, "verbose output")
	flag.Var(&exclude, "exclude", "glob pattern to exclude (repeatable)")
	flag.Var(&enableMut, "mutator", "enable specific mutator (repeatable; default: all)")
	flag.StringVar(&coverprofile, "coverprofile", "", "path to existing coverage profile")
	flag.StringVar(&diffBase, "diff", "", "only mutate lines changed since this git ref (e.g., main)")
	flag.BoolVar(&perTest, "per-test", false, "build per-test coverage map for targeted test execution")
	flag.StringVar(&configFile, "config", "", "path to config file (default: .mutagen.yaml)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: mutagen [flags] [packages]\n\n")
		fmt.Fprintf(os.Stderr, "Mutation testing engine for Go.\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  mutagen ./...                       Test all packages\n")
		fmt.Fprintf(os.Stderr, "  mutagen -workers 4 ./pkg/...        Test with 4 workers\n")
		fmt.Fprintf(os.Stderr, "  mutagen -threshold 80 ./...         Fail if kill rate < 80%%\n")
		fmt.Fprintf(os.Stderr, "  mutagen -diff main ./...            Only test changed lines\n")
		fmt.Fprintf(os.Stderr, "  mutagen -html report.html ./...     Generate HTML report\n")
		fmt.Fprintf(os.Stderr, "  mutagen -output json ./...          JSON output for CI\n")
	}

	flag.Parse()

	fileCfg := loadConfigFile(configFile)

	packages := flag.Args()
	if len(packages) == 0 {
		if len(fileCfg.Packages) > 0 {
			packages = fileCfg.Packages
		} else {
			packages = []string{"./..."}
		}
	}

	cfg := engine.Config{
		Packages:        packages,
		Workers:         coalesceInt("workers", workers, fileCfg.Workers),
		Timeout:         coalesceDuration("timeout", timeout, fileCfg.Timeout),
		Verbose:         verbose || fileCfg.Verbose,
		Threshold:       coalesceFloat("threshold", threshold, fileCfg.Threshold),
		Output:          coalesceString(flagSet("output"), output, fileCfg.Output, "text"),
		HTMLOutput:      coalesceString(flagSet("html"), htmlOutput, fileCfg.HTMLOutput, ""),
		CacheDir:        coalesceString(flagSet("cache-dir"), cacheDir, fileCfg.CacheDir, ".mutagen-cache"),
		NoCache:         noCache || fileCfg.NoCache,
		Exclude:         mergeSlices([]string(exclude), fileCfg.Exclude),
		Coverprofile:    coalesceString(flagSet("coverprofile"), coverprofile, fileCfg.Coverprofile, ""),
		DiffBase:        coalesceString(flagSet("diff"), diffBase, fileCfg.DiffBase, ""),
		PerTestCoverage: perTest,
		Mutators:        mergeSlices([]string(enableMut), fileCfg.Mutators),
	}

	return cfg, -1
}

type stringSlice []string

func (s *stringSlice) String() string         { return strings.Join(*s, ", ") }
func (s *stringSlice) Set(value string) error { *s = append(*s, value); return nil }

func flagSet(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

func coalesceInt(name string, flagVal, fileVal int) int {
	if flagSet(name) {
		return flagVal
	}
	return fileVal
}

func coalesceFloat(name string, flagVal, fileVal float64) float64 {
	if flagSet(name) {
		return flagVal
	}
	return fileVal
}

func coalesceDuration(name string, flagVal, fileVal time.Duration) time.Duration {
	if flagSet(name) {
		return flagVal
	}
	if fileVal != 0 {
		return fileVal
	}
	return flagVal
}

func coalesceString(explicit bool, flagVal, fileVal, defaultVal string) string {
	if explicit {
		return flagVal
	}
	if fileVal != "" {
		return fileVal
	}
	return defaultVal
}

func mergeSlices(a, b []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, s := range a {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	for _, s := range b {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}
