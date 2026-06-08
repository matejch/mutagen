# mutagen

Mutation testing engine for Go. Systematically mutates your source code (flips operators, removes nil checks, swaps boolean values, empties branches) and runs tests against each mutant. Mutations that survive — where tests still pass despite the change — reveal where your test suite is weak. Supports 9 mutation operators, parallel execution with `-overlay` isolation, coverage-guided filtering, incremental caching, per-test coverage mapping, diff-only mode, and arid-line detection.

Install with `go install ./cmd/mutagen/` and run `mutagen ./...` on any Go project. Use `-diff main` to only test changed lines in CI, `-per-test` to target only the tests that cover each mutated line, `-html report.html` for a source-annotated report, and `-threshold 80` to fail the build if the kill rate drops below 80%. Results are cached between runs — unchanged code is skipped automatically.

## Reports

Run `mutagen -html report.html ./...` to generate a source-annotated HTML report. Files are sorted by surviving mutation count (worst first), with inline badges showing each mutation and its status. Use `-output json` for machine-readable output or `-output text` (default) for terminal.

## Config

Copy `mutagen.example.yaml` to `.mutagen.yaml` in your project root. All fields are optional — CLI flags override the config file. Uncomment `mutators` to restrict which operators run, or `diff` to always run in diff-only mode.
