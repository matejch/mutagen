# mutagen

Mutation testing engine for Go. Mutates your code, runs your tests, and reports which changes your tests failed to catch. ([What is mutation testing?](#what-is-mutation-testing))

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

## What is mutation testing?

Code coverage tells you which lines your tests execute. It does not tell you whether your tests actually check anything. **A test that calls a function and ignores the result gets 100% coverage and catches zero bugs.**

Mutation testing answers a harder question: if I change this code, does any test break? The tool makes small, systematic changes to your source — replacing `+` with `-`, swapping `==` for `!=`, removing an `if err != nil` guard — and runs your test suite against each change. If every test still passes after a change, that mutation "survived," meaning your tests don't verify that behavior. If a test fails, the mutation was "killed," meaning your tests caught it.

The kill rate (killed / total tested) is a more honest measure of test quality than coverage. A codebase with 90% line coverage and a 60% kill rate has tests that run the code but don't verify what it does. A codebase with 70% coverage and an 85% kill rate has fewer tests, but the tests it has actually work.

Mutagen supports 9 mutation operators: arithmetic (`+`/`-`), comparison (`>`/`>=`, `==`/`!=`), logical (`&&`/`||`), boolean (`true`/`false`), nil check removal, return value replacement, compound assignment (`+=`/`-=`), branch removal (else blocks, switch cases), and bitwise (`&`/`|`, `<<`/`>>`). It uses coverage-guided filtering to skip uncovered lines, parallel execution with Go's `-overlay` flag for filesystem isolation, incremental caching to skip unchanged code, per-test coverage mapping to run only relevant tests per mutation, diff-only mode for CI, and arid-line detection to skip logging and boilerplate.
