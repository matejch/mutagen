package engine

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestTestMapRunPattern(t *testing.T) {
	tm := &TestMap{mapping: map[fileLine][]string{
		{File: "foo.go", Line: 10}: {"TestA", "TestB"},
		{File: "foo.go", Line: 20}: {"TestC"},
	}}

	pattern := tm.RunPattern("foo.go", 10)
	if pattern == "" {
		t.Fatal("expected non-empty pattern")
	}
	// Should contain both test names
	if !strings.Contains(pattern, "TestA") || !strings.Contains(pattern, "TestB") {
		t.Errorf("pattern should match TestA and TestB, got %s", pattern)
	}

	pattern = tm.RunPattern("foo.go", 20)
	if !strings.Contains(pattern, "TestC") {
		t.Errorf("expected pattern matching TestC, got %s", pattern)
	}

	pattern = tm.RunPattern("foo.go", 99)
	if pattern != "" {
		t.Errorf("expected empty pattern for uncovered line, got %s", pattern)
	}
}

func TestTestMapNil(t *testing.T) {
	var tm *TestMap
	tests := tm.TestsForLine("foo.go", 10)
	if tests != nil {
		t.Error("nil test map should return nil")
	}

	pattern := tm.RunPattern("foo.go", 10)
	if pattern != "" {
		t.Error("nil test map should return empty pattern")
	}
}

func TestTestMapCacheRoundTrip(t *testing.T) {
	dir := t.TempDir()

	tm := &TestMap{mapping: map[fileLine][]string{
		{File: "/src/foo.go", Line: 10}: {"TestA", "TestB"},
		{File: "/src/foo.go", Line: 20}: {"TestC"},
		{File: "/src/bar.go", Line: 5}:  {"TestD"},
	}}

	if err := SaveTestMap(dir, tm); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded := LoadTestMap(dir)
	if loaded == nil {
		t.Fatal("LoadTestMap returned nil")
	}

	for fl, expectedTests := range tm.mapping {
		gotTests := loaded.TestsForLine(fl.File, fl.Line)
		if len(gotTests) != len(expectedTests) {
			t.Errorf("line %s:%d: got %d tests, want %d", fl.File, fl.Line, len(gotTests), len(expectedTests))
		}
	}
}

func TestTestMapBuildIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// This test needs to run from the module root
	modRoot := findModuleRoot()
	if modRoot == "" {
		t.Skip("could not find module root")
	}

	// Use absolute package path
	tm, err := BuildTestMap([]string{modRoot + "/examples/sample"}, false)
	if err != nil {
		t.Skipf("skipping: %v", err)
	}

	if len(tm.mapping) == 0 {
		t.Fatal("expected non-empty test map")
	}

	found := false
	for fl, tests := range tm.mapping {
		if filepath.Base(fl.File) == "math.go" && len(tests) > 0 {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected math.go lines to be covered in test map")
	}
}
