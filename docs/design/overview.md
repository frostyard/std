# frostyard/std — Overview

## Purpose

`github.com/frostyard/std` is a Go standard library module for the Frostyard project. It provides shared, reusable packages with **zero external dependencies** (stdlib only). Currently the module contains a single package — `reporter` — which defines a progress-reporting interface and three implementations for human-readable, machine-readable, and silent output.

## Architecture

```
├── reporter/              # Core package — progress reporting interface + implementations
│   ├── reporter.go        # Reporter interface definition
│   ├── event.go           # EventType constants and ProgressEvent struct
│   ├── text.go            # TextReporter — human-readable output
│   ├── json.go            # JSONReporter — JSON Lines output (thread-safe)
│   ├── noop.go            # NoopReporter — silent discard (zero-value)
│   ├── *_test.go          # One test file per implementation
├── _examples/             # Runnable example programs
│   ├── deploy/            # Multi-step deployment pipeline
│   ├── fileprocess/       # Batch file processing with progress
│   ├── healthcheck/       # Service health checks with errors/warnings
│   └── migration/         # Data migration with batches
├── docs/                  # All docs, core's four-category shape (formerly yeti/)
│   ├── README.md          # Category table + index of every doc
│   ├── org-adrs.md        # Core ADRs binding this repo
│   ├── adr/               # Repo-local decisions (none yet)
│   ├── design/            # Living architecture docs — this file
│   ├── specs/             # Exact contracts — reporter-package.md
│   └── plans/             # Design documents and implementation plans
├── .github/
│   └── dependabot.yml     # Dependabot config (Go modules + GitHub Actions, weekly)
├── .svu.yaml              # svu (semantic version utility) config for `make bump`
├── go.mod                 # Module: github.com/frostyard/std, Go 1.26
├── Makefile               # Build/test/lint targets
└── AGENTS.md              # Canonical agent instructions (CLAUDE.md/GEMINI.md/.cursorrules symlink to it)
```

For the reporter package's exact interface and output contract, see
[../specs/reporter-package.md](../specs/reporter-package.md).

## Key Patterns

### Interface-driven design

All consumers depend on the `Reporter` interface, never on concrete types. This allows callers to swap between text, JSON, and noop output without changing application logic.

### Runtime format discrimination via `IsJSON()`

`IsJSON()` is a method on the `Reporter` interface that returns `true` only for `JSONReporter`. Callers use it to decide whether to emit additional human-readable output (tips, decorative separators) alongside the reporter — content that would corrupt a JSON Lines stream.

```go
if !r.IsJSON() {
    fmt.Println("Tip: use --format json for machine-readable output")
}
```

### Thread safety varies by implementation

| Implementation | Thread-safe | Mechanism |
|----------------|-------------|-----------|
| TextReporter   | No          | —         |
| JSONReporter   | Yes         | `sync.Mutex` on every `emit()` call |
| NoopReporter   | Yes         | No shared state |

Callers using concurrent goroutines must use `JSONReporter` or `NoopReporter`.

### Constructor conventions

- `NewTextReporter(w io.Writer) *TextReporter` — requires an `io.Writer`
- `NewJSONReporter(w io.Writer) *JSONReporter` — requires an `io.Writer`
- `NoopReporter` — zero-value struct, no constructor needed (`NoopReporter{}`)

### Zero external dependencies

The module imports nothing outside the Go standard library. This is a hard constraint — all packages must remain stdlib-only.

### Nil/zero-value handling in JSON output

Two important design decisions for correct JSON serialization:
- `ProgressEvent.Percent` is `*int` (not `int`) so that 0% is distinguishable from "not reported" — `nil` omits the field, `&0` emits `"percent": 0`
- `JSONReporter.Error()` with `nil` error emits `{"error": null}` (not `{"error": "<nil>"}`) — the details map is always present for consistent downstream parsing

### Modern Go (1.26)

The codebase uses modern Go features:
- `omitzero` struct tags (omit zero-value fields in JSON); `*int` with `omitempty` where zero is a valid value
- `range over int` in examples (e.g., `for batch := range batches` in migration example)
- Standard variadic patterns for formatted messages

### Testing patterns

- One test file per implementation (`text_test.go`, `json_test.go`, `noop_test.go`)
- Standard `testing` package only — no test frameworks
- Output captured via `bytes.Buffer`
- JSON tests unmarshal and validate individual fields (type, message, timestamp presence)
- Tests verify exact output formatting for text reporter

## Configuration

This module has no configuration files, environment variables, or runtime configuration. Behavior is determined entirely by which `Reporter` implementation is instantiated and what `io.Writer` is passed to its constructor.

Examples use a `-format` command-line flag to select between `text`, `json`, and `noop` output modes.

## Build & Test

```bash
make check           # Pre-commit gate: fmt + lint + test
make test            # Run all tests
make lint            # Run golangci-lint
make test-cover      # Tests with coverage + HTML report
make bump            # Tag next semver with svu and push
```

## Related Plans

Documents in [../plans/](../plans/) provide context for planned and past work:

- **clix module** ([2026-03-04-clix-design.md](../plans/2026-03-04-clix-design.md), [2026-03-04-clix-implementation.md](../plans/2026-03-04-clix-implementation.md)) — A planned separate module (`github.com/frostyard/clix`) that provides CLI convenience functions (version strings, common flags, JSON output helpers, reporter factory) built on fang/cobra. Separate from `std` because it has external dependencies. Three Frostyard CLI tools (nbc, updex, intuneme) are intended consumers. Shipped as [frostyard/clix](https://github.com/frostyard/clix).
- **reporter extraction** ([2026-03-04-reporter-extraction-design.md](../plans/2026-03-04-reporter-extraction-design.md), [2026-03-04-reporter-extraction.md](../plans/2026-03-04-reporter-extraction.md)) — Design for extracting the reporter package into this standalone module.
- **reporter examples** ([2026-03-04-reporter-examples-design.md](../plans/2026-03-04-reporter-examples-design.md), [2026-03-04-reporter-examples.md](../plans/2026-03-04-reporter-examples.md)) — Design for the `_examples/` directory demonstrating reporter usage patterns.

## Downstream Consumers

The `reporter` package is used by Frostyard CLI tools including `nbc`, `updex`, and `intuneme`. These tools use the `Reporter` interface for progress output during operations like disk management, package updates, and Intune management. The [clix](https://github.com/frostyard/clix) module provides a `NewReporter()` factory that selects the implementation based on `--json`/`--silent` flags.
