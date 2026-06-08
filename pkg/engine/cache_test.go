package engine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/matej/mutagen/pkg/mutator"
)

func TestCacheRoundTrip(t *testing.T) {
	dir := t.TempDir()

	// Create a source file to hash
	srcFile := filepath.Join(dir, "src.go")
	if err := os.WriteFile(srcFile, []byte("package p\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a test file
	testFile := filepath.Join(dir, "src_test.go")
	if err := os.WriteFile(testFile, []byte("package p\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cacheDir := filepath.Join(dir, "cache")

	mut := mutator.Mutation{
		File:        srcFile,
		Package:     "test",
		Line:        1,
		Column:      1,
		Original:    "+",
		Replacement: "-",
		Description: "test mutation",
	}
	testFiles := []string{testFile}

	// Store a result
	c := NewCache(cacheDir)
	c.Store(mut, mutator.StatusKilled, testFiles)
	if err := c.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}

	// Load it back
	c2 := NewCache(cacheDir)
	status, ok := c2.Lookup(mut, testFiles)
	if !ok {
		t.Fatal("cache miss after save/load")
	}
	if status != mutator.StatusKilled {
		t.Errorf("status = %v, want killed", status)
	}
}

func TestCacheInvalidation(t *testing.T) {
	dir := t.TempDir()

	srcFile := filepath.Join(dir, "src.go")
	if err := os.WriteFile(srcFile, []byte("package p\nfunc f() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	testFile := filepath.Join(dir, "src_test.go")
	if err := os.WriteFile(testFile, []byte("package p\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cacheDir := filepath.Join(dir, "cache")
	mut := mutator.Mutation{
		File: srcFile, Line: 1, Column: 1,
		Original: "+", Replacement: "-",
	}
	testFiles := []string{testFile}

	c := NewCache(cacheDir)
	c.Store(mut, mutator.StatusSurvived, testFiles)
	if err := c.Save(); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(srcFile, []byte("package p\nfunc f() { changed }\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	c2 := NewCache(cacheDir)
	_, ok := c2.Lookup(mut, testFiles)
	if ok {
		t.Error("cache should have missed after source change")
	}
}

func TestCacheTestFileChange(t *testing.T) {
	dir := t.TempDir()

	srcFile := filepath.Join(dir, "src.go")
	if err := os.WriteFile(srcFile, []byte("package p\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	testFile := filepath.Join(dir, "src_test.go")
	if err := os.WriteFile(testFile, []byte("package p\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cacheDir := filepath.Join(dir, "cache")
	mut := mutator.Mutation{
		File: srcFile, Line: 1, Column: 1,
		Original: "+", Replacement: "-",
	}
	testFiles := []string{testFile}

	c := NewCache(cacheDir)
	c.Store(mut, mutator.StatusKilled, testFiles)
	if err := c.Save(); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(testFile, []byte("package p\nfunc TestNew(t *testing.T) {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	c2 := NewCache(cacheDir)
	_, ok := c2.Lookup(mut, testFiles)
	if ok {
		t.Error("cache should have missed after test file change")
	}
}

func TestCacheEmptyDir(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "nonexistent", "cache")

	c := NewCache(cacheDir)
	_, ok := c.Lookup(mutator.Mutation{File: "x.go", Line: 1, Column: 1, Original: "+", Replacement: "-"}, nil)
	if ok {
		t.Error("empty cache should miss")
	}
}

func TestCacheLookupNonexistentFile(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "cache")

	mut := mutator.Mutation{
		File: "/nonexistent/src.go", Line: 1, Column: 1,
		Original: "+", Replacement: "-",
	}

	c := NewCache(cacheDir)
	c.Store(mut, mutator.StatusKilled, nil)
	if err := c.Save(); err != nil {
		t.Fatal(err)
	}

	// Lookup should fail because the file can't be hashed
	c2 := NewCache(cacheDir)
	_, ok := c2.Lookup(mut, nil)
	if ok {
		t.Error("lookup should fail when source file doesn't exist")
	}
}

func TestCacheStoreNonexistentFile(t *testing.T) {
	dir := t.TempDir()
	c := NewCache(filepath.Join(dir, "cache"))

	mut := mutator.Mutation{File: "/nonexistent/file.go", Line: 1, Column: 1}
	c.Store(mut, mutator.StatusKilled, nil)

	// Should not crash, just silently skip
	if err := c.Save(); err != nil {
		t.Fatal(err)
	}
}

func TestCacheCorruptedFile(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "cache")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write garbage to cache file
	if err := os.WriteFile(filepath.Join(cacheDir, "results.json"), []byte("{invalid json"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Should not crash, just start empty
	c := NewCache(cacheDir)
	_, ok := c.Lookup(mutator.Mutation{File: "x.go", Line: 1, Column: 1, Original: "+", Replacement: "-"}, nil)
	if ok {
		t.Error("corrupted cache should miss")
	}
}

func TestCacheSaveNoChanges(t *testing.T) {
	dir := t.TempDir()
	c := NewCache(filepath.Join(dir, "cache"))

	// Save without storing anything — should be a no-op
	if err := c.Save(); err != nil {
		t.Errorf("save with no changes should succeed, got %v", err)
	}

	// Cache dir should not be created
	if _, err := os.Stat(filepath.Join(dir, "cache", "results.json")); err == nil {
		t.Error("cache file should not be created when nothing was stored")
	}
}

func TestFindTestFiles(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "foo.go"), []byte("package p\n"), 0o644)      //nolint:errcheck
	os.WriteFile(filepath.Join(dir, "foo_test.go"), []byte("package p\n"), 0o644) //nolint:errcheck
	os.WriteFile(filepath.Join(dir, "bar_test.go"), []byte("package p\n"), 0o644) //nolint:errcheck
	os.WriteFile(filepath.Join(dir, "helper.go"), []byte("package p\n"), 0o644)   //nolint:errcheck

	testFiles := FindTestFiles(filepath.Join(dir, "foo.go"))
	if len(testFiles) != 2 {
		t.Errorf("expected 2 test files, got %d", len(testFiles))
	}
}

func TestFindTestFilesNonexistentDir(t *testing.T) {
	testFiles := FindTestFiles("/nonexistent/dir/foo.go")
	if testFiles != nil {
		t.Error("nonexistent dir should return nil")
	}
}
