# mutagen

Mutation testing engine for Go. Systematically mutates your source code (flips operators, removes nil checks, swaps boolean values, empties branches) and runs tests against each mutant. Mutations that survive — where tests still pass despite the change — reveal where your test suite is weak. Supports 9 mutation operators, parallel execution with `-overlay` isolation, coverage-guided filtering, incremental caching, per-test coverage mapping, diff-only mode, and arid-line detection.

## Install

```
make install
```

Or directly: `go install ./cmd/mutagen/`

## Usage

```
mutagen ./...                       # test all packages
mutagen -diff main ./...            # only test changed lines (CI mode)
mutagen -per-test ./...             # target only relevant tests per mutation
mutagen -threshold 80 ./...         # fail if kill rate < 80%
mutagen -html report.html ./...     # generate HTML report
mutagen -output json ./...          # JSON output for CI
```

Results are cached between runs — unchanged code is skipped automatically.

## Make targets

```
make test          # run unit tests
make mutate        # run mutation testing on this repo
make mutate-html   # same, with HTML report
```

## Reports

Use `-html report.html` to generate a source-annotated HTML report. Files are sorted by surviving mutation count (worst first), with inline badges showing each mutation and its status. Use `-output json` for machine-readable output or `-output text` (default) for terminal.

## Config

Copy `mutagen.example.yaml` to `.mutagen.yaml` in your project root. All fields are optional — CLI flags override the config file. Uncomment `mutators` to restrict which operators run, or `diff` to always run in diff-only mode.
