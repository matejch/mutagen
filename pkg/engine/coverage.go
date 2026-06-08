package engine

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func runCoverageProfile(patterns []string, verbose bool) (string, error) {
	tmpFile, err := os.CreateTemp("", "mutagen-cover-*.out")
	if err != nil {
		return "", err
	}
	tmpFile.Close()

	args := []string{"test", "-coverprofile=" + tmpFile.Name(), "-count=1"}
	args = append(args, patterns...)

	if verbose {
		fmt.Fprintf(os.Stderr, "Running coverage: go %s\n", strings.Join(args, " "))
	}

	cmd := exec.Command("go", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("go test -coverprofile failed: %v\n%s", err, string(output))
	}

	return tmpFile.Name(), nil
}

// parseCoverageProfile parses a Go coverage profile into covered file:line pairs.
// Format: name.go:startLine.startCol,endLine.endCol numStatements count
func parseCoverageProfile(path string) (map[fileLine]bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	covered := make(map[fileLine]bool)
	scanner := bufio.NewScanner(f)
	modRoot := findModuleRoot()

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "mode:") {
			continue
		}

		colonIdx := strings.LastIndex(line, ":")
		if colonIdx < 0 {
			continue
		}

		fileName := line[:colonIdx]
		rest := line[colonIdx+1:]

		parts := strings.Fields(rest)
		if len(parts) < 3 {
			continue
		}

		count, err := strconv.Atoi(parts[2])
		if err != nil || count == 0 {
			continue
		}

		rangeParts := strings.Split(parts[0], ",")
		if len(rangeParts) != 2 {
			continue
		}

		startLine, startErr := parseLineNum(rangeParts[0])
		endLine, endErr := parseLineNum(rangeParts[1])
		if startErr != nil || endErr != nil || startLine == 0 || endLine == 0 {
			continue
		}

		absFile := resolveFilePath(fileName, modRoot)

		for l := startLine; l <= endLine; l++ {
			covered[fileLine{File: absFile, Line: l}] = true
		}
	}

	return covered, scanner.Err()
}

func parseLineNum(s string) (int, error) {
	dotIdx := strings.Index(s, ".")
	if dotIdx >= 0 {
		s = s[:dotIdx]
	}
	return strconv.Atoi(s)
}

// resolveFilePath maps a coverage-profile path (e.g. "github.com/foo/bar/file.go")
// to an absolute filesystem path by stripping the module prefix.
func resolveFilePath(coveragePath string, modRoot string) string {
	if filepath.IsAbs(coveragePath) {
		return coveragePath
	}

	if modRoot != "" {
		modPath := readModulePath(modRoot)
		if modPath != "" && strings.HasPrefix(coveragePath, modPath) {
			rel := strings.TrimPrefix(coveragePath, modPath)
			rel = strings.TrimPrefix(rel, "/")
			abs := filepath.Join(modRoot, rel)
			if _, err := os.Stat(abs); err == nil {
				return abs
			}
		}

		abs := filepath.Join(modRoot, coveragePath)
		if _, err := os.Stat(abs); err == nil {
			return abs
		}
	}

	return coveragePath
}

func findModuleRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func readModulePath(modRoot string) string {
	data, err := os.ReadFile(filepath.Join(modRoot, "go.mod"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module"))
		}
	}
	return ""
}
