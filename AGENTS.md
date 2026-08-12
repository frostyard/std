# AGENTS.md

This file provides guidance to AI coding agents when working with code in this
repository. It is the canonical agent instructions file — `CLAUDE.md`,
`GEMINI.md`, and `.cursorrules` are symlinks to it (frostyard/core ADR-0018).

## Project

`github.com/frostyard/std` is a Go standard library module for the Frostyard project. It provides shared packages with zero external dependencies (stdlib only). Currently contains the `reporter` package.

## Commands

```bash
make test            # run all tests
make lint            # run golangci-lint
make check           # fmt + lint + test (pre-commit gate)
make bump            # tag next semver with svu and push
go test -v -run TestName ./reporter/  # run a single test
```

## Architecture

### reporter package

Defines a `Reporter` interface for progress reporting with three implementations:

- **TextReporter** — human-readable formatted output to an `io.Writer`. Not thread-safe.
- **JSONReporter** — JSON Lines output to an `io.Writer`. Thread-safe via mutex.
- **NoopReporter** — silent discard. Zero-value struct, no constructor needed.

`ProgressEvent` is the serialization type used by JSONReporter. `EventType` constants categorize events (step, progress, message, warning, error, complete).

`IsJSON()` is a runtime discriminator — callers use it to decide whether to emit structured or human output alongside the reporter.

## Conventions

- Go 1.26; use modern Go syntax (range-over-int, omitzero, etc.)
- One test file per implementation, standard `testing` package only
- Tests capture output via `bytes.Buffer`; JSON tests unmarshal and validate fields
- No external dependencies — stdlib only

## Documentation

**update documentation** After any change to source code, update relevant documentation in AGENTS.md, README.md and the `docs/` tree. A task is not complete without reviewing and updating relevant documentation.

**docs/ tree** All repository documentation lives in the single `docs/` tree, in frostyard/core's four-category shape per [frostyard/core ADR-0025](https://github.com/frostyard/core/blob/main/docs/adr/0025-consolidate-repository-docs-into-docs.md) (formerly the separate `yeti/` AI-docs directory): `docs/adr/` (why — repo-local decisions), `docs/design/` (how it fits together), `docs/specs/` (exact contracts), `docs/plans/` (order of work), indexed in [docs/README.md](docs/README.md). [docs/design/overview.md](docs/design/overview.md) is the entry point — read it for codebase context before performing tasks. New repo-local decisions get an ADR in `docs/adr/` (start from its `TEMPLATE.md`); org-wide decisions belong in frostyard/core — see [docs/org-adrs.md](docs/org-adrs.md). Write these docs to be maximally useful to an AI agent understanding the codebase — detailed architecture, patterns, and decision rationale rather than user-facing guides.
