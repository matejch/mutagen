package engine

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseCoverageProfile(t *testing.T) {
	// Write a minimal coverage profile
	tmp := t.TempDir()
	profile := filepath.Join(tmp, "cover.out")

	content := `mode: set
example.com/pkg/foo.go:10.2,12.10 1 1
example.com/pkg/foo.go:15.5,20.2 3 0
example.com/pkg/bar.go:5.1,8.3 2 1
`
	if err := os.WriteFile(profile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	covered, err := parseCoverageProfile(profile)
	if err != nil {
		t.Fatal(err)
	}

	// foo.go lines 10-12 should be covered (count=1)
	// foo.go lines 15-20 should NOT be covered (count=0)
	// bar.go lines 5-8 should be covered (count=1)

	// Since paths won't resolve to real files, they stay as-is
	for _, line := range []int{10, 11, 12} {
		if !covered[fileLine{File: "example.com/pkg/foo.go", Line: line}] {
			t.Errorf("foo.go:%d should be covered", line)
		}
	}

	for _, line := range []int{15, 16, 17, 18, 19, 20} {
		if covered[fileLine{File: "example.com/pkg/foo.go", Line: line}] {
			t.Errorf("foo.go:%d should NOT be covered", line)
		}
	}

	for _, line := range []int{5, 6, 7, 8} {
		if !covered[fileLine{File: "example.com/pkg/bar.go", Line: line}] {
			t.Errorf("bar.go:%d should be covered", line)
		}
	}
}

func TestCoverageAdjacentRanges(t *testing.T) {
	tmp := t.TempDir()
	profile := filepath.Join(tmp, "cover.out")

	// Lines 10-12 covered, 13-15 NOT covered, 16-18 covered
	content := `mode: set
example.com/pkg/foo.go:10.1,12.10 1 1
example.com/pkg/foo.go:13.1,15.10 1 0
example.com/pkg/foo.go:16.1,18.10 1 1
`
	if err := os.WriteFile(profile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	covered, err := parseCoverageProfile(profile)
	if err != nil {
		t.Fatal(err)
	}

	// Boundary: line 12 covered, line 13 NOT covered
	if !covered[fileLine{File: "example.com/pkg/foo.go", Line: 12}] {
		t.Error("line 12 should be covered (boundary)")
	}
	if covered[fileLine{File: "example.com/pkg/foo.go", Line: 13}] {
		t.Error("line 13 should NOT be covered (boundary)")
	}
	// Boundary: line 15 NOT covered, line 16 covered
	if covered[fileLine{File: "example.com/pkg/foo.go", Line: 15}] {
		t.Error("line 15 should NOT be covered (boundary)")
	}
	if !covered[fileLine{File: "example.com/pkg/foo.go", Line: 16}] {
		t.Error("line 16 should be covered (boundary)")
	}
}

func TestCoverageMalformedLines(t *testing.T) {
	tmp := t.TempDir()
	profile := filepath.Join(tmp, "cover.out")

	content := `mode: set
example.com/pkg/foo.go:10.1,12.10 1 1
this is garbage
no-colon-line 1 1
example.com/pkg/bar.go:5.1,8.3 2 1
`
	if err := os.WriteFile(profile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	covered, err := parseCoverageProfile(profile)
	if err != nil {
		t.Fatal(err)
	}

	// Valid lines should still parse
	if !covered[fileLine{File: "example.com/pkg/foo.go", Line: 10}] {
		t.Error("valid line should still be parsed despite garbage")
	}
	if !covered[fileLine{File: "example.com/pkg/bar.go", Line: 5}] {
		t.Error("valid line after garbage should still be parsed")
	}
}

func TestCoverageEmptyProfile(t *testing.T) {
	tmp := t.TempDir()
	profile := filepath.Join(tmp, "cover.out")

	if err := os.WriteFile(profile, []byte("mode: set\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	covered, err := parseCoverageProfile(profile)
	if err != nil {
		t.Fatal(err)
	}
	if len(covered) != 0 {
		t.Errorf("empty profile should produce 0 covered lines, got %d", len(covered))
	}
}

func TestCoverageNonexistentFile(t *testing.T) {
	_, err := parseCoverageProfile("/nonexistent/cover.out")
	if err == nil {
		t.Error("expected error for nonexistent profile")
	}
}

func TestResolveFilePathAbsolute(t *testing.T) {
	abs := resolveFilePath("/absolute/path/foo.go", "/some/root")
	if abs != "/absolute/path/foo.go" {
		t.Errorf("absolute path should be returned as-is, got %s", abs)
	}
}

func TestResolveFilePathNoModRoot(t *testing.T) {
	result := resolveFilePath("example.com/pkg/foo.go", "")
	if result != "example.com/pkg/foo.go" {
		t.Errorf("no mod root should return path as-is, got %s", result)
	}
}

func TestFindModuleRoot(t *testing.T) {
	root := findModuleRoot()
	if root == "" {
		t.Skip("not running inside a Go module")
	}
	gomod := filepath.Join(root, "go.mod")
	if _, err := os.Stat(gomod); err != nil {
		t.Errorf("findModuleRoot returned %s but go.mod not found there", root)
	}
}

func TestReadModulePath(t *testing.T) {
	root := findModuleRoot()
	if root == "" {
		t.Skip("not running inside a Go module")
	}
	modPath := readModulePath(root)
	if modPath == "" {
		t.Error("readModulePath should return non-empty module path")
	}
	if modPath != "github.com/matej/mutagen" {
		t.Errorf("unexpected module path: %s", modPath)
	}
}

func TestReadModulePathInvalid(t *testing.T) {
	result := readModulePath("/nonexistent/dir")
	if result != "" {
		t.Errorf("nonexistent dir should return empty, got %s", result)
	}
}

func TestParseLineNum(t *testing.T) {
	tests := []struct {
		input   string
		want    int
		wantErr bool
	}{
		{"10.2", 10, false},
		{"1.1", 1, false},
		{"100.50", 100, false},
		{"5", 5, false},
		{"", 0, true},
		{"abc", 0, true},
	}

	for _, tt := range tests {
		got, err := parseLineNum(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("parseLineNum(%q): err=%v, wantErr=%v", tt.input, err, tt.wantErr)
		}
		if got != tt.want {
			t.Errorf("parseLineNum(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}
