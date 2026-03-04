# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

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
