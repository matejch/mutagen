package engine

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/matej/mutagen/pkg/mutator"
)

// Cache provides incremental mutation testing by storing previous results.
// A cached result is valid when the source file and test files haven't changed.
type Cache struct {
	dir     string
	entries map[string]cacheEntry
	dirty   bool
}

type cacheEntry struct {
	MutationKey string         `json:"mutation_key"` // Unique mutation identifier
	FileHash    string         `json:"file_hash"`    // SHA-256 of the source file
	TestHash    string         `json:"test_hash"`    // SHA-256 of test files in the package
	Status      mutator.Status `json:"status"`
}

const cacheFileName = "results.json"

// NewCache creates or loads a cache from the given directory.
func NewCache(dir string) *Cache {
	c := &Cache{
		dir:     dir,
		entries: make(map[string]cacheEntry),
	}
	c.load()
	return c
}

func (c *Cache) load() {
	data, err := os.ReadFile(filepath.Join(c.dir, cacheFileName))
	if err != nil {
		return
	}
	var entries []cacheEntry
	if json.Unmarshal(data, &entries) != nil {
		return
	}
	for _, e := range entries {
		c.entries[e.MutationKey] = e
	}
}

func (c *Cache) Save() error {
	if !c.dirty {
		return nil
	}

	if err := os.MkdirAll(c.dir, 0o755); err != nil {
		return err
	}

	entries := make([]cacheEntry, 0, len(c.entries))
	for _, e := range c.entries {
		entries = append(entries, e)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].MutationKey < entries[j].MutationKey
	})

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(c.dir, cacheFileName), data, 0o644)
}

func (c *Cache) Lookup(mut mutator.Mutation, testFiles []string) (mutator.Status, bool) {
	key := mutationKey(mut)
	entry, ok := c.entries[key]
	if !ok {
		return 0, false
	}

	fileHash, err := hashFile(mut.File)
	if err != nil || fileHash != entry.FileHash {
		return 0, false
	}

	testHash := hashFiles(testFiles)
	if testHash != entry.TestHash {
		return 0, false
	}

	return entry.Status, true
}

func (c *Cache) Store(mut mutator.Mutation, status mutator.Status, testFiles []string) {
	fileHash, err := hashFile(mut.File)
	if err != nil {
		return
	}

	c.entries[mutationKey(mut)] = cacheEntry{
		MutationKey: mutationKey(mut),
		FileHash:    fileHash,
		TestHash:    hashFiles(testFiles),
		Status:      status,
	}
	c.dirty = true
}

func mutationKey(mut mutator.Mutation) string {
	return fmt.Sprintf("%s:%d:%d:%s->%s", mut.File, mut.Line, mut.Column, mut.Original, mut.Replacement)
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func hashFiles(paths []string) string {
	sorted := make([]string, len(paths))
	copy(sorted, paths)
	sort.Strings(sorted)

	h := sha256.New()
	for _, p := range sorted {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		h.Write([]byte(p))
		h.Write(data)
	}
	return hex.EncodeToString(h.Sum(nil))
}

// FindTestFiles returns test files for the package containing the given source file.
func FindTestFiles(sourceFile string) []string {
	dir := filepath.Dir(sourceFile)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var testFiles []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), "_test.go") {
			testFiles = append(testFiles, filepath.Join(dir, e.Name()))
		}
	}
	return testFiles
}
