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
├── tests/e2e/             # E2E suite: builds and runs every _examples/* program in every mode
├── docs/                  # All docs, core's four-category shape (formerly yeti/)
│   ├── README.md          # Category table + index of every doc
│   ├── org-adrs.md        # Core ADRs binding this repo
│   ├── adr/               # Repo-local decisions — 0001 (conformance aliases)
│   ├── design/            # Living architecture docs — this file, quality-loop.md
│   ├── specs/             # Exact contracts — reporter-package.md, PR rubric, PR metric
│   ├── plans/             # Design documents and implementation plans
│   └── metrics.md, review-rubric.md, quality.md   # conformance aliases (ADR-0001)
├── .github/
│   ├── workflows/ci.yml   # Lint, Security Scan, Unit Tests, Race Detection, Verify, Docs integrity, Release config
│   ├── workflows/release.yml  # On tag push: changelog-only GoReleaser release
│   ├── pull_request_template.md, ISSUE_TEMPLATE/, prompts/
│   ├── copilot-instructions.md -> ../AGENTS.md
│   └── dependabot.yml     # Dependabot config (Go modules + GitHub Actions, weekly)
├── policies/agent-governance.json  # Fluent enrollment surface (core repository-surfaces v1)
├── scripts/check-docs.mjs # Docs-integrity gate (index, links, aliases; .coverage-thresholds.json)
├── .agents/skills/        # Skills synced from frostyard/core (.claude/skills -> here)
├── .memory/               # Corrections inbox (append-only corrections.jsonl)
├── .claude/               # settings.json (tool-layer limits), session-summary.md
├── .goreleaser.yaml       # builds skipped; changelog grouped by commit type
├── .svu.yaml              # svu (semantic version utility) config for `make bump`
├── .golangci.yml          # What `make lint` and CI run (v2, standard + gofmt)
├── .editorconfig
├── go.mod                 # Module: github.com/frostyard/std, Go 1.26
├── Makefile               # Build/test/lint targets
└── AGENTS.md              # Canonical agent instructions + contributing guide
                           # (CLAUDE.md, GEMINI.md, CONTRIBUTING.md, .cursorrules,
                           #  .github/copilot-instructions.md symlink to it)
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

- `NewTextReporter(w io.Writer) *TextReporter` — a nil writer is normalized to `io.Discard`, keeping reporting silent and non-panicking
- `NewJSONReporter(w io.Writer) *JSONReporter` — a nil writer is normalized to `io.Discard`, keeping reporting silent and non-panicking
- `NoopReporter` — zero-value struct, no constructor needed (`NoopReporter{}`)

### Zero external dependencies

The module imports nothing outside the Go standard library. This is a hard constraint — all packages must remain stdlib-only.

### Nil/zero-value handling in JSON output

Three important design decisions for correct JSON serialization:
- `ProgressEvent.Percent` is `*int` (not `int`) so that 0% is distinguishable from "not reported" — `nil` omits the field, `&0` emits `"percent": 0`
- `JSONReporter.Error()` with `nil` error emits `{"error": null}` (not `{"error": "<nil>"}`) — the details map is always present for consistent downstream parsing
- `JSONReporter` never drops an event because its `Details` cannot be encoded: when `Complete` (the only method that forwards arbitrary caller data) receives a value `encoding/json` rejects (func, channel, NaN/Inf, failing `Marshaler`), the event is re-emitted with `Details` set to `{"encoding_error": "<reason>"}` so consumers still see the terminal `complete` line. Writer (I/O) errors stay silent to callers but make that reporter fail-stop: every later event is dropped so no valid-looking record can follow a partial JSON fragment — see [reporter-package.md](../specs/reporter-package.md#jsonreporter-reporterjsongo)

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
- End-to-end: `tests/e2e/examples_test.go` builds each `_examples/*` program with Go coverage instrumentation and runs it as a subprocess in `json`, `text`, `noop`, and an invalid format under a project-local `GOCOVERDIR`, decoding the JSON Lines stream with `reporter.ProgressEvent` (unknown fields rejected; last event `complete`). A final `go tool covdata percent` assertion fails unless the subprocesses covered reporter statements; an empty-directory regression test pins that failure path. This subprocess signal is separate from the in-process `coverage.out` profile and its 95% floor. `go test ./...` skips underscore directories, so this is the only thing that compiles the examples ([tests/e2e/README.md](../../tests/e2e/README.md)).

## Configuration

This module has no configuration files, environment variables, or runtime configuration. Behavior is determined entirely by which `Reporter` implementation is instantiated and what `io.Writer` is passed to its constructor.

Examples use a `-format` command-line flag to select between `text`, `json`, and `noop` output modes.

## Build & Test

```bash
make check           # Pre-commit: fmt + lint + vet + test + coverage self-test + 95% floor
make test            # Run all tests (unit + tests/e2e)
make lint            # Run golangci-lint (.golangci.yml)
make test-cover      # Tests with coverage + HTML report
make bump            # Tag next semver with svu and push (triggers release.yml)
node scripts/check-docs.mjs   # Docs-integrity gate
```

CI (`.github/workflows/ci.yml`) runs Lint, Security Scan, Unit Tests, Race
Detection, Verify (tidy/vet/gofmt), Docs integrity, and Release config on every
PR, every push to `main`, and every merge-queue branch (`merge_group`);
the loop around it — template, rubric, gates, corrections, metric, release
path — is described in [quality-loop.md](quality-loop.md).

## Release

`make bump` tags the next semver (svu, `.svu.yaml`) and pushes the tag;
`.github/workflows/release.yml` then runs GoReleaser Pro with
`.goreleaser.yaml` — builds skipped, changelog grouped by Conventional Commit
type, GitHub release under `frostyard/std` (`prerelease: auto`). Consumers
fetch tags through the Go module proxy; nothing else is published.

## Agent governance

`policies/agent-governance.json` is std's agent-governance surface under
frostyard/core's repository-surfaces contract v1 — with `AGENTS.md`,
`.agents/skills/`, and `docs/README.md` it is what Fluent reads to enroll the
repository. Deny by default; read/write/run-tests allowed; issues, PRs, and
follow-ups review-required; `.github/workflows/**` and the release surface
are review-required at high risk. `.claude/settings.json` enforces the same
limits at the tool layer. Rationale:
[ADR-0001](../adr/0001-acmm-conformance-via-canonical-aliases.md).

## Related Plans

Documents in [../plans/](../plans/) provide context for planned and past work:

- **clix module** ([2026-03-04-clix-design.md](../plans/2026-03-04-clix-design.md), [2026-03-04-clix-implementation.md](../plans/2026-03-04-clix-implementation.md)) — A planned separate module (`github.com/frostyard/clix`) that provides CLI convenience functions (version strings, common flags, JSON output helpers, reporter factory) built on fang/cobra. Separate from `std` because it has external dependencies. Three Frostyard CLI tools (nbc, updex, intuneme) are intended consumers. Shipped as [frostyard/clix](https://github.com/frostyard/clix).
- **reporter extraction** ([2026-03-04-reporter-extraction-design.md](../plans/2026-03-04-reporter-extraction-design.md), [2026-03-04-reporter-extraction.md](../plans/2026-03-04-reporter-extraction.md)) — Design for extracting the reporter package into this standalone module.
- **reporter examples** ([2026-03-04-reporter-examples-design.md](../plans/2026-03-04-reporter-examples-design.md), [2026-03-04-reporter-examples.md](../plans/2026-03-04-reporter-examples.md)) — Design for the `_examples/` directory demonstrating reporter usage patterns.

## Downstream Consumers

The `reporter` package is used by Frostyard CLI tools including `nbc`, `updex`, and `intuneme`. These tools use the `Reporter` interface for progress output during operations like disk management, package updates, and Intune management. The [clix](https://github.com/frostyard/clix) module provides a `NewReporter()` factory that selects the implementation based on `--json`/`--silent` flags.
