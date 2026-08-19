# frostyard/std

`github.com/frostyard/std` is a Go standard library module for the Frostyard
project. It provides shared packages with **zero external dependencies**
(stdlib only). Currently it contains the `reporter` package. Start at
[docs/README.md](docs/README.md); read
[docs/design/overview.md](docs/design/overview.md) and
[docs/specs/reporter-package.md](docs/specs/reporter-package.md) for codebase
context before performing tasks.

This file (`AGENTS.md`) is the CANONICAL agent instructions **and** the
contributing guide — `CLAUDE.md`, `GEMINI.md`, `CONTRIBUTING.md`,
`.cursorrules`, and `.github/copilot-instructions.md` are symlinks to it, and
`.claude/skills` symlinks to `.agents/skills/`
([ADR-0001](docs/adr/0001-acmm-conformance-via-canonical-aliases.md); pattern
from
[frostyard/core ADR-0002](https://github.com/frostyard/core/blob/main/docs/adr/0002-agent-portable-instruction-surface.md),
[ADR-0018](https://github.com/frostyard/core/blob/main/docs/adr/0018-org-wide-agent-instruction-and-knowledge-surfaces.md),
and
[ADR-0029](https://github.com/frostyard/core/blob/main/docs/adr/0029-acmm-conformance-via-canonical-aliases.md)).
Edit only the canonical paths; keep content tool-agnostic. Conformance alias
symlinks are listed in ADR-0001 — edit their canonical targets, never the
aliases.

## Skills (follow these for common tasks)

Step-by-step procedures live in [.agents/skills/](.agents/skills/) (synced
from frostyard/core by
[core ADR-0026](https://github.com/frostyard/core/blob/main/docs/adr/0026-distribute-core-skills-via-sync-prs.md)
— edit them in core, not here); follow them rather than improvising,
whichever agent you are:

- **Structuring, building, testing, or releasing this repo the frostyard Go
  way** →
  [.agents/skills/frostyard-go-repo/SKILL.md](.agents/skills/frostyard-go-repo/SKILL.md)
- **Maintaining the `docs/` tree (four-category shape, index, migrations)** →
  [.agents/skills/frostyard-repo-docs/SKILL.md](.agents/skills/frostyard-repo-docs/SKILL.md)
- **Hive ACMM conformance (fleet prerequisites, alias lattice)** →
  [.agents/skills/frostyard-acmm-conformance/SKILL.md](.agents/skills/frostyard-acmm-conformance/SKILL.md)
  — canonical aliases per ADR-0001, never duplicated content

## Getting started

### Prerequisites

- **Go 1.26.6 or newer** (`go.mod` targets `go 1.26.6`; CI uses the version
  `go.mod` names)
- `make`
- [`golangci-lint`](https://golangci-lint.run/) v2 for `make lint`
  (configured by [`.golangci.yml`](.golangci.yml); the release CI installs is
  pinned as `GOLANGCI_LINT_VERSION` in the `Makefile`, currently 2.12.2 —
  the Makefile prints a skip notice when the binary is absent, warns when the
  installed version differs from the pin, and fails on findings when it is
  present; CI always runs the pinned release)
- [`svu`](https://github.com/caarlos0/svu) only for `make bump`
- Node 20+ only for `node scripts/check-docs.mjs` (zero dependencies)

### Commands

```bash
make check           # fmt + lint + vet + test + coverage floor — the local gate; run before every PR
make test            # go test -v ./... with a coverage profile (unit tests + tests/e2e)
make lint            # golangci-lint run (.golangci.yml), module + _examples/
make vet             # go vet, module + _examples/
make test-cover      # coverage profile + HTML report
make coverage-check  # enforce the 95.0% total statement-coverage floor on coverage.out
make test-coverage-check  # self-test scripts/check-coverage.sh with fixture profiles
make bump            # tag next semver with svu and push the tag (see Releases)
go test -v -run TestName ./reporter/   # run a single unit test
go test -v ./tests/e2e/...             # the example-program e2e suite alone
node scripts/check-docs.mjs            # docs-integrity gate (index, links, aliases)
```

`make check` runs `gofmt -w` first, so it may modify files — commit the
result. `go test ./...` deliberately excludes `_examples/` (underscore
directories are invisible to package patterns); the e2e suite in
[tests/e2e/](tests/e2e/README.md) builds and runs those programs so they
cannot rot. The same blind spot applies to the analyzers, so `make lint`,
`make vet`, and the CI Lint and Verify jobs each run a second pass over the
example package directories named explicitly;
[`scripts/example-dirs.sh`](scripts/example-dirs.sh) enumerates them (and
fails when the list is empty), so adding `_examples/<program>/` needs no
change to the `Makefile` or to `ci.yml` — see
[docs/design/quality-loop.md](docs/design/quality-loop.md).

## Architecture

### reporter package

Defines a `Reporter` interface for progress reporting with three
implementations:

- **TextReporter** — human-readable formatted output to an `io.Writer`. Not
  thread-safe.
- **JSONReporter** — JSON Lines output to an `io.Writer`. Thread-safe via
  mutex.
- **NoopReporter** — silent discard. Zero-value struct, no constructor needed.

`ProgressEvent` is the serialization type used by JSONReporter. `EventType`
constants categorize events (step, progress, message, warning, error,
complete).

`IsJSON()` is a runtime discriminator — callers use it to decide whether to
emit structured or human output alongside the reporter.

The `_examples/` programs (`deploy`, `fileprocess`, `healthcheck`,
`migration`) each take `-format text|json|noop` and exercise the whole
interface; they double as the e2e fixtures.

## Conventions

- Go 1.26; use modern Go syntax (range-over-int, omitzero, etc.)
- **No external dependencies — stdlib only.** This is a hard constraint for
  every package in the module; the e2e suite may import only stdlib and
  `github.com/frostyard/std/...`.
- One test file per implementation, standard `testing` package only
- Tests capture output via `bytes.Buffer`; JSON tests unmarshal and validate
  fields
- Formatting is `gofmt` (enforced by `make fmt` and the `gofmt` formatter in
  `.golangci.yml`); linting is golangci-lint's `standard` set —
  `.golangci.yml` is the single source of what `make lint` and CI run, and
  `GOLANGCI_LINT_VERSION` in the `Makefile` is the single source of which
  golangci-lint release runs it (the CI Lint job reads that variable and
  `make lint` warns on a mismatch), so a new finding after a deliberate
  linter bump is fixed in code, never hidden by loosening the config
- [`.editorconfig`](.editorconfig) carries the editor defaults (tabs in Go
  and Makefiles, two-space YAML/JSON/Markdown, LF, final newline)
- Error strings are lowercase without trailing punctuation
- The `TestI` prefix stays reserved for environment-requiring integration
  tests (core ADR-0022); nothing here needs it today

## Testing

- **Unit tests** — `reporter/*_test.go`, one file per implementation; run
  with `make test` or `go test ./reporter/`.
- **End-to-end** — [`tests/e2e/examples_test.go`](tests/e2e/examples_test.go)
  builds every `_examples/*` program and runs it as a subprocess in `json`,
  `text`, `noop`, and an invalid format; the JSON Lines stream is decoded
  with `reporter.ProgressEvent` (unknown fields rejected), timestamps must be
  RFC3339, and the last event must be `complete`. It is part of `go test
  ./...` and of CI's Unit Tests and Race Detection jobs.
- New or changed behavior needs a focused test including a failure path;
  exact text formatting is pinned by the tests and by
  [docs/specs/reporter-package.md](docs/specs/reporter-package.md) — change
  them together.

## Continuous integration

[`.github/workflows/ci.yml`](.github/workflows/ci.yml) runs on every push to
`main` and every pull request: **Lint** (the `GOLANGCI_LINT_VERSION` release
of golangci-lint from the `Makefile`, with `.golangci.yml`, over the module
and the `_examples/` programs), **Security Scan** (`govulncheck ./...` with
`golang.org/x/vuln/cmd/govulncheck@v1.6.0` pinned — reachable Go
standard-library advisories, since the module has no dependencies), **Unit
Tests** (`go test -v ./...` with a coverage profile, then the 95.0% total
statement-coverage floor via `make coverage-check`
([`scripts/check-coverage.sh`](scripts/check-coverage.sh)) after
`make test-coverage-check` self-tests it; the profile is uploaded as the
`coverage-profile` artifact),
**Race Detection** (`go test -race -short ./...`), **Verify** (`go mod tidy`
leaves no diff, `go vet` over the module and the `_examples/` programs,
`gofmt -l` empty), **Docs integrity** (`node
scripts/check-docs.mjs` — every doc indexed in `docs/README.md`, every
relative link resolving, every symlink alias intact, thresholds in
[`.coverage-thresholds.json`](.coverage-thresholds.json), all `1.0`,
`never_relax: true`), and **Release config** (`goreleaser check` over
[`.goreleaser.yaml`](.goreleaser.yaml), so a broken release configuration
fails pre-merge instead of after an immutable tag is pushed; it runs the
GoReleaser Pro distribution at the same action SHA `release.yml` uses,
because the config sets `pro: true`, and needs the org secret
`GORELEASER_KEY`). Actions are pinned by commit SHA with least-privilege
permissions (core ADR-0021). `make check` reproduces the Go half locally.

## Releases

std is a library: a release is a Git tag plus a GitHub release with a
changelog — no binaries, packages, or archives.

1. The operator runs `make bump` on a clean, checked `main`: it runs `make
   check`, asks [`svu next`](.svu.yaml) for the next semantic version from
   the Conventional Commit history (`v0` tags, `always: true`), creates an
   annotated tag, and pushes it.
2. The pushed tag triggers
   [`.github/workflows/release.yml`](.github/workflows/release.yml), which
   runs GoReleaser Pro (`goreleaser release --clean`) with
   [`.goreleaser.yaml`](.goreleaser.yaml): builds are skipped (`builds: [{
   skip: true }]`), the changelog is grouped by Conventional Commit type, and
   the GitHub release is created under `frostyard/std` (`prerelease: auto`
   marks pre-release tags). It needs `GITHUB_TOKEN` (workflow-provided) and
   the org secret `GORELEASER_KEY`.
3. Consumers pick the tag up through the Go module proxy; nothing else is
   published.

Agents never run `make bump`, push tags, or edit a published release; those
are operator acts (see Agent limits below).

## Commits and pull requests

- Branch off `main`; the org squash-merges, so never stack on another PR's
  branch.
- PR titles and commit subjects are Conventional Commits
  (`type(scope): summary`); the squash commit is what `svu` versions and the
  GoReleaser changelog groups by.
- Fill in [`.github/pull_request_template.md`](.github/pull_request_template.md):
  gates run, docs housekeeping, aliases untouched. Reviews apply
  [docs/specs/pr-review-rubric.md](docs/specs/pr-review-rubric.md); the
  review procedure for agents is
  [.github/prompts/review.prompt.md](.github/prompts/review.prompt.md).
- Before opening a PR: `make check` green, `node scripts/check-docs.mjs`
  green, docs updated (below).
- Issues use the templates in `.github/ISSUE_TEMPLATE/`; blank issues stay
  enabled.

## Agent limits and governance

- [`policies/agent-governance.json`](policies/agent-governance.json) is this
  repository's agent-governance surface under frostyard/core's
  repository-surfaces contract v1 — the file Fluent reads (from GitHub, at
  the observed default-branch head) when enrolling std in its fleet,
  alongside `AGENTS.md`, `.agents/skills/`, and `docs/README.md`. Deny by
  default; read, write, and run-tests allowed; issues, pull requests, and
  follow-ups review-required; `.github/workflows/**` and the release surface
  (`.goreleaser.yaml`, `.svu.yaml`, `.github/workflows/release.yml`) are
  review-required at high risk. Change it only alongside the matching ADR
  or design change; it must validate against core's
  `organization/schemas/v1/repository-agent-governance.schema.json`.
- [`.claude/settings.json`](.claude/settings.json) enforces the same limits
  at the tool layer: never merge a PR, approve your own work, publish a
  release, push to `main`, or force-push; pushing a branch and dispatching a
  workflow ask first; `.env` and secrets are unreadable.
- Corrections that should outlive a session go in
  [.memory/corrections.jsonl](.memory/README.md) (append-only, five fields)
  and are promoted into this file, a doc, or a skill; session continuity
  goes in [.claude/session-summary.md](.claude/session-summary.md).

## Documentation

**update documentation** After any change to source code, update relevant
documentation in `AGENTS.md`, `README.md` (if one exists — std has none
today), and the `docs/` tree. A task is not complete
without reviewing and updating relevant documentation.

**docs/ tree** All repository documentation lives in the single `docs/`
tree, in frostyard/core's four-category shape per
[frostyard/core ADR-0025](https://github.com/frostyard/core/blob/main/docs/adr/0025-consolidate-repository-docs-into-docs.md)
(formerly the separate `yeti/` AI-docs directory): `docs/adr/` (why —
repo-local decisions), `docs/design/` (how it fits together), `docs/specs/`
(exact contracts), `docs/plans/` (order of work), indexed in
[docs/README.md](docs/README.md). [docs/design/overview.md](docs/design/overview.md)
is the entry point — read it for codebase context before performing tasks.
New repo-local decisions get an ADR in `docs/adr/` (start from its
`TEMPLATE.md`); org-wide decisions belong in frostyard/core — see
[docs/org-adrs.md](docs/org-adrs.md). Write these docs to be maximally useful
to an AI agent understanding the codebase — detailed architecture, patterns,
and decision rationale rather than user-facing guides.

**rules the gate enforces** Every new doc starts from its category's
`TEMPLATE.md`, gets a line in `docs/README.md`, and cross-links both ways
(design → ADR + spec; spec → ADR + design; ADR → what it shapes).
`docs/metrics.md`, `docs/review-rubric.md`, and `docs/quality.md` are
conformance aliases (ADR-0001) — not docs, never indexed; edit
`docs/specs/pr-acceptance-metric.md`, `docs/specs/pr-review-rubric.md`, and
`docs/design/quality-loop.md` instead. `node scripts/check-docs.mjs` fails on
any unindexed doc, dead relative link, or broken alias.
