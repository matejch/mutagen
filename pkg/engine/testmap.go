package engine

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// TestMap maps source file lines to the test functions that cover them.
type TestMap struct {
	mapping map[fileLine][]string // file:line -> test function names
}

// BuildTestMap profiles each test function individually to build a
// line-to-test mapping. This is expensive (one `go test -run` per test)
// but the result is cached.
func BuildTestMap(packages []string, testRunFilter string, verbose bool) (*TestMap, error) {
	if verbose {
		fmt.Fprintf(os.Stderr, "Building per-test coverage map...\n")
	}

	tm := &TestMap{mapping: make(map[fileLine][]string)}
	tmpDir, err := os.MkdirTemp("", "mutagen-testmap-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	var runFilter *regexp.Regexp
	if testRunFilter != "" {
		var err error
		runFilter, err = regexp.Compile(testRunFilter)
		if err != nil {
			return nil, fmt.Errorf("invalid -test-run pattern: %w", err)
		}
	}

	for _, pkg := range packages {
		tests, err := listTestFunctions(pkg)
		if err != nil {
			if verbose {
				fmt.Fprintf(os.Stderr, "Warning: could not list tests for %s: %v\n", pkg, err)
			}
			continue
		}

		if runFilter != nil {
			var filtered []string
			for _, t := range tests {
				if runFilter.MatchString(t) {
					filtered = append(filtered, t)
				}
			}
			tests = filtered
		}

		if verbose {
			fmt.Fprintf(os.Stderr, "  %s: %d test functions\n", pkg, len(tests))
		}

		for _, testName := range tests {
			coverFile := filepath.Join(tmpDir, sanitizeFilename(pkg+"_"+testName)+".cover")
			if err := runSingleTestCoverage(pkg, testName, coverFile); err != nil {
				if verbose {
					fmt.Fprintf(os.Stderr, "  Warning: %s/%s failed: %v\n", pkg, testName, err)
				}
				continue
			}

			covered, err := parseCoverageProfile(coverFile)
			if err != nil {
				continue
			}

			for fl := range covered {
				tm.mapping[fl] = append(tm.mapping[fl], testName)
			}
		}
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Test map: %d covered lines mapped\n", len(tm.mapping))
	}

	return tm, nil
}

// TestsForLine returns the test functions that cover the given file:line.
func (tm *TestMap) TestsForLine(file string, line int) []string {
	if tm == nil {
		return nil
	}
	return tm.mapping[fileLine{File: file, Line: line}]
}

// RunPattern returns a `-run` regex that matches all tests covering file:line.
// Returns empty string if no tests cover the line (caller should skip the mutation).
func (tm *TestMap) RunPattern(file string, line int) string {
	tests := tm.TestsForLine(file, line)
	if len(tests) == 0 {
		return ""
	}

	// Deduplicate and build regex
	seen := make(map[string]bool)
	var unique []string
	for _, t := range tests {
		if !seen[t] {
			seen[t] = true
			unique = append(unique, "^"+regexp.QuoteMeta(t)+"$")
		}
	}

	return strings.Join(unique, "|")
}

// listTestFunctions discovers top-level Test* functions in a package
// using `go test -list`.
func listTestFunctions(pkg string) ([]string, error) {
	cmd := exec.Command("go", "test", "-list", ".*", pkg)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("go test -list: %w\n%s", err, output)
	}

	var tests []string
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Test") || strings.HasPrefix(line, "Example") {
			tests = append(tests, line)
		}
	}
	return tests, nil
}

func runSingleTestCoverage(pkg, testName, coverFile string) error {
	cmd := exec.Command("go", "test",
		"-run", "^"+regexp.QuoteMeta(testName)+"$",
		"-coverprofile="+coverFile,
		"-count=1",
		"-short",
		pkg,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %v\n%s", testName, err, output)
	}
	return nil
}

func sanitizeFilename(s string) string {
	r := strings.NewReplacer("/", "_", "\\", "_", ":", "_", " ", "_")
	return r.Replace(s)
}

const testMapCacheFile = "testmap.json"

type testMapEntry struct {
	File  string   `json:"file"`
	Line  int      `json:"line"`
	Tests []string `json:"tests"`
}

// SaveTestMap writes the test map to the cache directory.
func SaveTestMap(dir string, tm *TestMap) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	var entries []testMapEntry
	for fl, tests := range tm.mapping {
		entries = append(entries, testMapEntry{File: fl.File, Line: fl.Line, Tests: tests})
	}

	data, err := json.Marshal(entries)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(dir, testMapCacheFile), data, 0o644)
}

// LoadTestMap loads a cached test map. Returns nil if not found or invalid.
func LoadTestMap(dir string) *TestMap {
	data, err := os.ReadFile(filepath.Join(dir, testMapCacheFile))
	if err != nil {
		return nil
	}

	var entries []testMapEntry
	if json.Unmarshal(data, &entries) != nil {
		return nil
	}

	tm := &TestMap{mapping: make(map[fileLine][]string, len(entries))}
	for _, e := range entries {
		tm.mapping[fileLine{File: e.File, Line: e.Line}] = e.Tests
	}
	return tm
}
