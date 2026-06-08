package engine

import (
	"testing"
)

func TestParseDiffOutput(t *testing.T) {
	diff := `diff --git a/pkg/foo.go b/pkg/foo.go
index abc..def 100644
--- a/pkg/foo.go
+++ b/pkg/foo.go
@@ -10,3 +10,5 @@ func f() {
+	new line 1
+	new line 2
@@ -20,0 +22,1 @@ func g() {
+	another new line
diff --git a/pkg/bar_test.go b/pkg/bar_test.go
--- a/pkg/bar_test.go
+++ b/pkg/bar_test.go
@@ -5,1 +5,2 @@ func TestBar(t *testing.T) {
+	test change
`
	changes := parseDiffOutput(diff, "/repo")

	// bar_test.go should be excluded (test file)
	if _, ok := changes["/repo/pkg/bar_test.go"]; ok {
		t.Error("should exclude test files from diff")
	}

	// foo.go should have changes
	fooChanges, ok := changes["/repo/pkg/foo.go"]
	if !ok {
		t.Fatal("foo.go should have changes")
	}

	// First hunk: +10,5 means lines 10-14
	for _, line := range []int{10, 11, 12, 13, 14} {
		if !fooChanges[line] {
			t.Errorf("foo.go:%d should be changed", line)
		}
	}

	// Second hunk: +22,1 means line 22
	if !fooChanges[22] {
		t.Error("foo.go:22 should be changed")
	}

	// Line 20 should NOT be changed
	if fooChanges[20] {
		t.Error("foo.go:20 should NOT be changed")
	}
}

func TestParseDiffOutputDeletionOnly(t *testing.T) {
	diff := `diff --git a/pkg/foo.go b/pkg/foo.go
--- a/pkg/foo.go
+++ b/pkg/foo.go
@@ -10,3 +10,0 @@ func f() {
`
	changes := parseDiffOutput(diff, "/repo")

	// Deletion-only hunk (count=0) should produce no changed lines
	if fooChanges, ok := changes["/repo/pkg/foo.go"]; ok && len(fooChanges) > 0 {
		t.Error("deletion-only hunk should not produce changed lines")
	}
}

func TestParseDiffOutputNonGoFiles(t *testing.T) {
	diff := `diff --git a/README.md b/README.md
--- a/README.md
+++ b/README.md
@@ -1,1 +1,2 @@
+new line
`
	changes := parseDiffOutput(diff, "/repo")
	if len(changes) != 0 {
		t.Error("should ignore non-Go files")
	}
}

func TestDiffFilterNil(t *testing.T) {
	var d *DiffFilter
	if !d.IsChanged("any.go", 1) {
		t.Error("nil filter should pass everything")
	}
	if !d.HasChanges("any.go") {
		t.Error("nil filter should pass everything")
	}
}
