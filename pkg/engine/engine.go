package engine

import (
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"os"
	"strings"
	"time"

	"github.com/matej/mutagen/pkg/mutator"
	"golang.org/x/tools/go/packages"
)

type Config struct {
	Packages        []string
	Workers         int
	Timeout         time.Duration
	Verbose         bool
	Threshold       float64
	Output          string // "text" or "json" or "html"
	HTMLOutput      string
	CacheDir        string
	NoCache         bool
	Exclude         []string
	Coverprofile    string   // Path to existing coverage profile
	DiffBase        string   // Git base ref for diff-only mode (e.g., "main")
	Mutators        []string // Enabled mutator names (empty = all)
	PerTestCoverage bool     // Build per-test coverage map for targeted test execution
}

type Engine struct {
	cfg       Config
	fset      *token.FileSet
	coverage  map[fileLine]bool
	diff      *DiffFilter
	exclusion *ExclusionChecker
	testMap   *TestMap
}

type fileLine struct {
	File string
	Line int
}

func New(cfg Config) *Engine {
	return &Engine{
		cfg:       cfg,
		fset:      token.NewFileSet(),
		exclusion: NewExclusionChecker(cfg.Exclude),
	}
}

func (e *Engine) Run() ([]mutator.Result, error) {
	if e.cfg.Verbose {
		fmt.Fprintf(os.Stderr, "Loading packages: %v\n", e.cfg.Packages)
	}

	pkgs, err := e.loadPackages()
	if err != nil {
		return nil, fmt.Errorf("loading packages: %w", err)
	}

	if e.cfg.DiffBase != "" {
		d, err := NewDiffFilter(e.cfg.DiffBase, e.cfg.Verbose)
		if err != nil {
			return nil, fmt.Errorf("diff filter: %w", err)
		}
		e.diff = d
	}

	if err := e.collectCoverage(); err != nil {
		return nil, fmt.Errorf("collecting coverage: %w", err)
	}

	if e.cfg.PerTestCoverage {
		if err := e.buildTestMap(); err != nil {
			return nil, fmt.Errorf("building test map: %w", err)
		}
	}

	allMutations := e.collectMutations(pkgs)

	if e.cfg.Verbose {
		fmt.Fprintf(os.Stderr, "Found %d mutations across %d packages\n", len(allMutations), len(pkgs))
	}

	if len(allMutations) == 0 {
		fmt.Fprintf(os.Stderr, "No mutations found.\n")
		return nil, nil
	}

	return e.runWithCache(allMutations)
}

func (e *Engine) collectMutations(pkgs []*packages.Package) []mutator.Mutation {
	mutators := e.enabledMutators()
	fileLines := make(map[string][]string)
	var all []mutator.Mutation

	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			filename := e.fset.Position(file.Pos()).Filename
			muts := e.collectFileMutations(pkg, file, filename, mutators, fileLines)
			all = append(all, muts...)
		}
	}

	return all
}

func (e *Engine) collectFileMutations(
	pkg *packages.Package,
	file *ast.File,
	filename string,
	mutators []mutator.Mutator,
	fileLines map[string][]string,
) []mutator.Mutation {
	if e.exclusion.ShouldExcludeFile(filename) {
		return nil
	}
	if e.diff != nil && !e.diff.HasChanges(filename) {
		return nil
	}

	if _, ok := fileLines[filename]; !ok {
		if data, err := os.ReadFile(filename); err == nil {
			fileLines[filename] = strings.Split(string(data), "\n")
		}
	}

	var result []mutator.Mutation
	for _, m := range mutators {
		mutations := m.Mutate(e.fset, file, pkg.TypesInfo)
		for i := range mutations {
			mutations[i].Package = pkg.PkgPath
			if mutations[i].File == "" {
				mutations[i].File = filename
			}
		}
		for _, mut := range mutations {
			if !e.shouldIncludeMutation(mut, fileLines) {
				continue
			}
			result = append(result, mut)
		}
	}
	return result
}

func (e *Engine) shouldIncludeMutation(mut mutator.Mutation, fileLines map[string][]string) bool {
	if !e.isCovered(mut.File, mut.Line) {
		return false
	}
	if e.diff != nil && !e.diff.IsChanged(mut.File, mut.Line) {
		return false
	}
	if lines, ok := fileLines[mut.File]; ok && mut.Line > 0 && mut.Line <= len(lines) {
		if e.exclusion.IsAridLine(mut.File, mut.Line, lines[mut.Line-1]) {
			return false
		}
	}
	return true
}

func (e *Engine) runWithCache(allMutations []mutator.Mutation) ([]mutator.Result, error) {
	var cache *Cache
	if !e.cfg.NoCache {
		cache = NewCache(e.cfg.CacheDir)
	}

	var toRun []mutator.Mutation
	var results []mutator.Result

	for _, mut := range allMutations {
		if cache != nil {
			testFiles := FindTestFiles(mut.File)
			if status, ok := cache.Lookup(mut, testFiles); ok {
				results = append(results, mutator.Result{Mutation: mut, Status: status})
				continue
			}
		}
		toRun = append(toRun, mut)
	}

	if e.cfg.Verbose && len(allMutations) != len(toRun) {
		fmt.Fprintf(os.Stderr, "Cache: %d cached, %d to test\n", len(allMutations)-len(toRun), len(toRun))
	}

	if len(toRun) > 0 {
		runner := NewRunner(e.cfg, e.fset, e.testMap)
		runResults, err := runner.RunAll(toRun)
		if err != nil {
			return nil, fmt.Errorf("running mutations: %w", err)
		}

		for _, res := range runResults {
			if cache != nil && (res.Status == mutator.StatusKilled || res.Status == mutator.StatusSurvived) {
				testFiles := FindTestFiles(res.Mutation.File)
				cache.Store(res.Mutation, res.Status, testFiles)
			}
		}
		results = append(results, runResults...)
	}

	if cache != nil {
		if err := cache.Save(); err != nil && e.cfg.Verbose {
			fmt.Fprintf(os.Stderr, "Warning: failed to save cache: %v\n", err)
		}
	}

	return results, nil
}

func (e *Engine) loadPackages() ([]*packages.Package, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedSyntax |
			packages.NeedTypes |
			packages.NeedTypesInfo |
			packages.NeedCompiledGoFiles,
		Fset:  e.fset,
		Tests: false,
	}

	pkgs, err := packages.Load(cfg, e.cfg.Packages...)
	if err != nil {
		return nil, err
	}

	var errs []string
	var valid []*packages.Package
	for _, pkg := range pkgs {
		if len(pkg.Errors) > 0 {
			for _, e := range pkg.Errors {
				errs = append(errs, e.Error())
			}
			continue
		}
		valid = append(valid, pkg)
	}

	if len(valid) == 0 && len(errs) > 0 {
		return nil, fmt.Errorf("all packages had errors:\n%s", strings.Join(errs, "\n"))
	}
	if e.cfg.Verbose && len(errs) > 0 {
		fmt.Fprintf(os.Stderr, "Warning: some packages had errors:\n%s\n", strings.Join(errs, "\n"))
	}

	return valid, nil
}

func (e *Engine) collectCoverage() error {
	profile := e.cfg.Coverprofile
	if profile == "" {
		var err error
		profile, err = runCoverageProfile(e.cfg.Packages, e.cfg.Verbose)
		if err != nil {
			if e.cfg.Verbose {
				fmt.Fprintf(os.Stderr, "Warning: coverage collection failed: %v\nProceeding without coverage filtering.\n", err)
			}
			return nil
		}
		defer os.Remove(profile)
	}

	covered, err := parseCoverageProfile(profile)
	if err != nil {
		if e.cfg.Verbose {
			fmt.Fprintf(os.Stderr, "Warning: coverage parsing failed: %v\nProceeding without coverage filtering.\n", err)
		}
		return nil
	}

	e.coverage = covered
	if e.cfg.Verbose {
		fmt.Fprintf(os.Stderr, "Coverage data: %d covered lines\n", len(e.coverage))
	}
	return nil
}

func (e *Engine) isCovered(file string, line int) bool {
	if e.coverage == nil {
		return true
	}
	return e.coverage[fileLine{File: file, Line: line}]
}

func (e *Engine) enabledMutators() []mutator.Mutator {
	all := mutator.All()
	if len(e.cfg.Mutators) == 0 {
		return all
	}

	enabled := make(map[string]bool, len(e.cfg.Mutators))
	for _, name := range e.cfg.Mutators {
		enabled[name] = true
	}

	var filtered []mutator.Mutator
	for _, m := range all {
		if enabled[m.Name()] {
			filtered = append(filtered, m)
		}
	}
	return filtered
}

func (e *Engine) buildTestMap() error {
	if !e.cfg.NoCache {
		if cached := LoadTestMap(e.cfg.CacheDir); cached != nil {
			if e.cfg.Verbose {
				fmt.Fprintf(os.Stderr, "Loaded cached test map (%d lines)\n", len(cached.mapping))
			}
			e.testMap = cached
			return nil
		}
	}

	tm, err := BuildTestMap(e.cfg.Packages, e.cfg.Verbose)
	if err != nil {
		return err
	}
	e.testMap = tm

	if !e.cfg.NoCache {
		if err := SaveTestMap(e.cfg.CacheDir, tm); err != nil && e.cfg.Verbose {
			fmt.Fprintf(os.Stderr, "Warning: failed to cache test map: %v\n", err)
		}
	}
	return nil
}

func PrintMutatedFile(fset *token.FileSet, file *ast.File, dest string) error {
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	cfg := printer.Config{
		Mode:     printer.SourcePos,
		Tabwidth: 8,
	}
	return cfg.Fprint(f, fset, file)
}
