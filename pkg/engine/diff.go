package engine

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// ChangedLines represents lines changed in a file.
type ChangedLines struct {
	File  string       // Absolute path
	Lines map[int]bool // Set of changed line numbers
}

// DiffFilter restricts mutations to lines changed relative to a base ref.
type DiffFilter struct {
	changes map[string]map[int]bool // file -> set of changed lines
}

// NewDiffFilter creates a filter from git diff output.
// baseRef can be a branch name ("main"), commit hash, or range ("main...HEAD").
func NewDiffFilter(baseRef string, verbose bool) (*DiffFilter, error) {
	repoRoot, err := gitRepoRoot()
	if err != nil {
		return nil, fmt.Errorf("not a git repository: %w", err)
	}

	if err := gitRefExists(baseRef); err != nil {
		return nil, fmt.Errorf("base ref %q not found: %w", baseRef, err)
	}

	diffRef := baseRef
	if !strings.Contains(baseRef, "...") && !strings.Contains(baseRef, "..") {
		diffRef = baseRef + "...HEAD"
	}

	cmd := exec.Command("git", "diff", "--unified=0", diffRef)
	cmd.Dir = repoRoot
	output, err := cmd.Output()
	if err != nil {
		// Try without ...HEAD (might be comparing working tree)
		cmd = exec.Command("git", "diff", "--unified=0", baseRef)
		cmd.Dir = repoRoot
		output, err = cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("git diff failed: %w", err)
		}
	}

	cmdUnstaged := exec.Command("git", "diff", "--unified=0")
	cmdUnstaged.Dir = repoRoot
	unstaged, err := cmdUnstaged.Output()
	if err != nil {
		unstaged = nil
	}

	cmdStaged := exec.Command("git", "diff", "--unified=0", "--cached")
	cmdStaged.Dir = repoRoot
	staged, err := cmdStaged.Output()
	if err != nil {
		staged = nil
	}

	changes := parseDiffOutput(string(output), repoRoot)
	mergeChanges(changes, parseDiffOutput(string(unstaged), repoRoot))
	mergeChanges(changes, parseDiffOutput(string(staged), repoRoot))

	if verbose {
		totalLines := 0
		for _, lines := range changes {
			totalLines += len(lines)
		}
		fmt.Fprintf(os.Stderr, "Diff filter: %d changed files, %d changed lines\n", len(changes), totalLines)
	}

	return &DiffFilter{changes: changes}, nil
}

// IsChanged returns true if the given file:line was changed in the diff.
func (d *DiffFilter) IsChanged(file string, line int) bool {
	if d == nil {
		return true // No diff filter = everything passes
	}
	lines, ok := d.changes[file]
	if !ok {
		return false
	}
	return lines[line]
}

// HasChanges returns true if the given file has any changes.
func (d *DiffFilter) HasChanges(file string) bool {
	if d == nil {
		return true
	}
	_, ok := d.changes[file]
	return ok
}

// ChangedFiles returns the list of files with changes.
func (d *DiffFilter) ChangedFiles() []string {
	if d == nil {
		return nil
	}
	files := make([]string, 0, len(d.changes))
	for f := range d.changes {
		files = append(files, f)
	}
	return files
}

var (
	diffFileRegex = regexp.MustCompile(`^\+\+\+ b/(.+)$`)
	hunkRegex     = regexp.MustCompile(`^@@ -\d+(?:,\d+)? \+(\d+)(?:,(\d+))? @@`)
)

func parseDiffOutput(diff string, repoRoot string) map[string]map[int]bool {
	changes := make(map[string]map[int]bool)

	var currentFile string
	scanner := bufio.NewScanner(strings.NewReader(diff))

	for scanner.Scan() {
		line := scanner.Text()

		// Match file header: +++ b/path/to/file.go
		if matches := diffFileRegex.FindStringSubmatch(line); matches != nil {
			relPath := matches[1]
			if strings.HasSuffix(relPath, ".go") && !strings.HasSuffix(relPath, "_test.go") {
				currentFile = filepath.Join(repoRoot, relPath)
			} else {
				currentFile = ""
			}
			continue
		}

		if currentFile != "" {
			if matches := hunkRegex.FindStringSubmatch(line); matches != nil {
				startLine, err := strconv.Atoi(matches[1])
				if err != nil {
					continue
				}
				count := 1
				if matches[2] != "" {
					count, err = strconv.Atoi(matches[2])
					if err != nil {
						continue
					}
				}
				if count == 0 {
					continue
				}

				if changes[currentFile] == nil {
					changes[currentFile] = make(map[int]bool)
				}
				for l := startLine; l < startLine+count; l++ {
					changes[currentFile][l] = true
				}
			}
		}
	}

	return changes
}

func mergeChanges(dst, src map[string]map[int]bool) {
	for file, lines := range src {
		if dst[file] == nil {
			dst[file] = make(map[int]bool)
		}
		for line := range lines {
			dst[file][line] = true
		}
	}
}

func gitRepoRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func gitRefExists(ref string) error {
	// Strip ...HEAD or ..HEAD suffix for validation
	base := ref
	if idx := strings.Index(ref, "..."); idx >= 0 {
		base = ref[:idx]
	} else if idx := strings.Index(ref, ".."); idx >= 0 {
		base = ref[:idx]
	}

	cmd := exec.Command("git", "rev-parse", "--verify", base)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ref %q does not exist", base)
	}
	return nil
}
