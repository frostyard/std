# frostyard/std

`github.com/frostyard/std` is a Go standard library module for the Frostyard
project. It provides shared packages with **zero external dependencies**
(stdlib only). It contains 2 packages: `reporter` (progress reporting) and
`problems` (problem records in the local systemd journal). The folder
[`rust/`](rust/README.md) holds the Rust crate `frostyard-problems`, the
same problem-record writer for Rust apps (Rust standard library only;
[ADR-0003](docs/adr/0003-rust-crate-frostyard-problems-in-rust-folder.md)).
Start at
[docs/README.md](docs/README.md); read
[docs/design/overview.md](docs/design/overview.md),
[docs/specs/reporter-package.md](docs/specs/reporter-package.md) and
[docs/specs/problems-package.md](docs/specs/problems-package.md) for codebase
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

- **Go 1.26.10** (`go.mod`'s `toolchain go1.26.10` line is the only Go pin;
  CI, `mise`, and Snowcat workers all read it)
- `make`
- [`mise`](https://mise.jdx.dev/) to provision every other pinned tool:
  `mise install` reads [`mise.toml`](mise.toml) and verifies each download
  against [`mise.lock`](mise.lock) (core ADR-0043). Today that is
  [`golangci-lint`](https://golangci-lint.run/) v2 for `make lint`
  (configured by [`.golangci.yml`](.golangci.yml)); the Makefile reads the
  pin from `mise.toml`, fails with `mise install` when the binary is absent,
  warns when the installed version differs, and fails on findings; CI
  installs the same lock through `jdx/mise-action`
- [`svu`](https://github.com/caarlos0/svu) only for `make bump`, pinned in
  `mise.toml` (3.4.1; `mise install` provides it). svu reads `.svu.yml` only
  from v3 on, so `make bump` runs `make svu-version-check` first and refuses
  any other release on `PATH` (an older `go install`ed svu ignores the config)
- **Rust 1.99.1** only for `make rust-check` (the crate in `rust/`), pinned
  in `mise.toml` with clippy, rustfmt and the `x86_64-pc-windows-gnu` and
  `x86_64-apple-darwin` targets; `mise install` provides it. `make
  rust-check` (and so `make ci`) fails with `mise install` when `cargo` is
  absent
- Node 24+ only for `node scripts/check-docs.mjs` (zero dependencies);
  the 20.x line reached end-of-life on 2026-04-30, and 24 is the Active LTS

### Commands

```bash
make check           # fmt + lint + vet + test + coverage floor — the local gate; run before every PR
make ci              # canonical local gate: runs make verify first (tidy diff and no go.sum, gofmt, lint at the pin, vet, tests), then unit+e2e tests with the coverage profile, race, cross-arch build of _examples/, the Rust crate (make rust-check) — not a mirror of the GitHub workflow; the CI Verify job runs this target (core ADR-0038, ADR-0044, ADR-0073 form 1)
make verify          # credential-free, non-mutating gate (what a read-only reviewer runs): tidy diff, gofmt -l, lint at the exact pin, vet, tests
make test            # unit + e2e; asserts subprocess coverage and writes the in-process coverage profile
make lint            # golangci-lint run (.golangci.yml), module + _examples/
make vet             # go vet, module + _examples/
make test-cover      # coverage profile + HTML report
make coverage-check  # enforce the 95.0% total statement-coverage floor on coverage.out
make test-coverage-check  # self-test scripts/check-coverage.sh with fixture profiles
make svu-version-check  # fail unless the svu on PATH is the mise.toml pin (make bump runs it first)
make bump            # tag next semver with svu and push the tag (see Releases)
make rust-check      # Rust crate in rust/: cargo fmt --check, clippy -D warnings, cargo test, check builds for Windows and macOS, no dependency in cargo tree
go test -v -run TestName ./reporter/   # run a single unit test
go test -v ./tests/e2e/...             # the example-program e2e suite alone
node scripts/check-docs.mjs            # docs-integrity gate (index, links, aliases)
```

`make ci` is the canonical local gate core ADR-0038 requires, not a
reproduction of [`.github/workflows/ci.yml`](.github/workflows/ci.yml). It runs
`make verify` first, then its own steps, in one shell and in the order listed
above, stopping at the first failure; the workflow's jobs declare no `needs:`
and run in parallel, so there is no CI ordering to mirror. The CI `Verify` job
runs `make ci` (core ADR-0073 form 1), so the local gate and the hosted one
are the same command (core ADR-0044). The two are deliberately not the same
set:

- **GitHub only** — the Security Scan job (`govulncheck` over the module and
  the `_examples/` programs), the Docs integrity job
  (`node scripts/check-docs.mjs`), and the Release config job (the goreleaser
  configuration check) have no step in `make ci`. Run
  `node scripts/check-docs.mjs` yourself after a docs change.
- **Both, deliberately identical** — the cross-architecture build of every
  `_examples/` program for `linux/amd64` and `linux/arm64` runs in `make ci`
  *and* in the workflow's Examples Cross-Arch Build job (`examples-build`),
  off the same [`scripts/example-dirs.sh`](scripts/example-dirs.sh)
  enumeration, so the local gate and the hosted one cannot drift apart.
  The same holds for the Rust crate: the last step of `make ci` is `make
  rust-check`, which the workflow's Rust crate job (`rust`) runs too.
- **Coverage floor** — the workflow's Unit Tests job runs
  `make test-coverage-check` and `make coverage-check` on top of the test run;
  `make ci` does not. Locally that floor belongs to `make check`.

`make check` runs `gofmt -w` first, so it may modify files — commit the
result. `go test ./...` deliberately excludes `_examples/` (underscore
directories are invisible to package patterns); the e2e suite in
[tests/e2e/](tests/e2e/README.md) builds those programs with Go coverage
instrumentation, runs them with a project-local `GOCOVERDIR`, and fails unless
the resulting covdata reports covered `reporter` statements. That subprocess
signal is separate from `coverage.out`, whose 95% floor covers `reporter` and
`problems` code executed inside the `go test` processes (one total over both
packages, the `-coverpkg` list of `make test` and the CI Unit Tests job).
The same package-pattern blind spot applies to the analyzers, so `make lint`
and `make vet` (and so `make verify` and the CI Verify job) each run a second
pass over the
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

### problems package

Writes problem records (panics and coded errors) of a frostyard app to the
local systemd journal and reads them back (chairlift uses the reader). It
implements the writer and reader side of
[core `docs/specs/problem-report.md`](https://github.com/frostyard/core/blob/main/docs/specs/problem-report.md)
(S8 `app_problem`, the journal record registry); the decision is
[ADR-0002](docs/adr/0002-problems-package-writes-to-the-journal-only.md) and
the contract is [docs/specs/problems-package.md](docs/specs/problems-package.md).

- **Recorder** — `New(appID, version, component, opts...)` never fails; a bad
  identity makes it inert (`Err()` says why). `Error(code, err)` records a
  coded error; `defer r.RecoverPanic()` records a panic of its goroutine and
  panics again; `CrashOutput()` covers other goroutines and fatal errors
  through `runtime/debug.SetCrashOutput` at the next start.
- **Transport** — 1 non-blocking `AF_UNIX` datagram per record, made with
  `syscall`; every send failure drops the record silently; at most 10
  records per process in any 60 seconds. The package never uses the network.
- **Reader** — `Read` runs `journalctl -o json --all`; `ParseJSON` and
  `Validate` (the one validation path) drop and count every record that
  fails schema v1. Only journald's `_` fields are trusted.
- The v1 patterns, allowlist and enums in `problems/problems.go` are copied
  from core's `v1.json`; change them only with a core spec change, together
  with the shared vectors in `problems/testdata/vectors/` (also read by the
  Rust crate).
- Linux only at run time: other OSes get stubs that compile and do nothing.

### Rust crate frostyard-problems (`rust/`)

The Rust writer and validator of the same records
([ADR-0003](docs/adr/0003-rust-crate-frostyard-problems-in-rust-folder.md);
contract: the section "Rust crate" of
[docs/specs/problems-package.md](docs/specs/problems-package.md)). `rust/`
is its own cargo workspace root; `rust/Cargo.lock` is committed and every
cargo command runs `--locked`.

- **Rust standard library only, also in tests.** No `[dependencies]` table
  of any kind; `make rust-check` fails unless `cargo tree` prints only the
  crate, and a test fails on network type names in the sources. The v1
  patterns are hand-written matchers in `rust/src/patterns.rs` that must
  give exactly the result of the Go regular expressions in
  `problems/problems.go`; change both together.
- **API** — `Recorder::new(app_id, version, component)` (inert on a bad
  identity, `err()` says why), `with_socket`, `error(code, Option<&dyn
  Display>)` (formats `err` only when it can write, on Linux, inside
  `catch_unwind`: a panicking `Display` gives the code as `MESSAGE`, and
  the panic payload is leaked, not dropped),
  `install_panic_hook(env!("CARGO_MANIFEST_DIR"))` (`#[track_caller]`;
  records the panic FIRST, THEN
  calls the previous hook; the crate's own work in the hook never panics,
  but a previous hook that panics or blocks is outside that guarantee),
  `map_location(crate_root, file, line)` (the "crate that installs the
  hook" rule: only files under `<crate_root>src/`; the root is the longest
  prefix of the caller file before a `src` folder that ends
  `CARGO_MANIFEST_DIR`, else `""` only for a caller file with `src/` as its
  only `src` folder, else none), `validate` (the rules of Go
  `Validate`, frames included), `decode_entry`.
- **Transport** — 1 `send_to` on a new non-blocking
  `UnixDatagram::unbound()` on Linux; every error drops the record; the
  rate cap is a process-wide static. Other OSes compile and write nothing.

## Conventions

- Go 1.26; use modern Go syntax (range-over-int, omitzero, etc.)
- **No external dependencies — stdlib only.** This is a hard constraint for
  every package in the module; the e2e suite may import only stdlib and
  `github.com/frostyard/std/...`.
- One test file per implementation (in `problems/`, per source file),
  standard `testing` package only
- Tests capture output via `bytes.Buffer`; JSON tests unmarshal and validate
  fields
- Formatting is `gofmt` (enforced by `make fmt` and the `gofmt` formatter in
  `.golangci.yml`); linting is golangci-lint's `standard` set —
  `.golangci.yml` is the single source of what `make lint` and CI run, and
  `mise.toml` is the single source of which golangci-lint release runs it
  (the Makefile and the CI Lint job both read that pin, `mise.lock` carries
  its checksums, and `make lint` warns on a mismatch), so a new finding
  after a deliberate linter bump is fixed in code, never hidden by loosening
  the config. Bump golangci-lint before bumping `go.mod`'s toolchain, as one
  change: `make verify` refuses a linter built with an older Go than the
  toolchain
- [`.editorconfig`](.editorconfig) carries the editor defaults (tabs in Go
  and Makefiles, two-space YAML/JSON/Markdown, LF, final newline)
- Error strings are lowercase without trailing punctuation

## Testing

- **Unit tests** — `reporter/*_test.go`, one file per implementation; run
  with `make test` or `go test ./reporter/`.
- **problems tests** — `problems/*_test.go`: a `unixgram` test socket in
  `t.TempDir()` (`WithSocket`), a replaced rate-cap clock (no sleeps),
  subprocess runs of the test binary (`TestMain` switches on
  `PROBLEMS_TEST_HELPER`) for real panics and crashes, a fake `journalctl`
  script for `Read`, every vector in `problems/testdata/vectors/v1.txt`, and
  a `GOOS=windows`/`GOOS=darwin` build (skipped with `-short`). Run with
  `go test ./problems/`.
- **Rust crate tests** — `rust/src/tests/` (inside the crate, so they reach
  the crate-private encoder, frame mapping and rate cap; standard test
  harness only): every vector in `problems/testdata/vectors/v1.txt` (Rust
  `write` vectors through a `UnixDatagram` test socket in a new folder under
  the temporary folder), the matchers, `map_location`, the decoder, the
  message cut, the rate cap with given instants, the hook order with a fake
  sender and a fake previous hook, and the test binary run again as a helper
  process (`--exact`, `FROSTYARD_PROBLEMS_TEST_HELPER`) for a real panic.
  Tests that send records hold a static lock and reset the shared rate cap.
  Run with `make rust-check` (or `cargo test --locked` in `rust/`).
- **End-to-end** — [`tests/e2e/examples_test.go`](tests/e2e/examples_test.go)
  builds every `_examples/*` program with coverage instrumentation and runs it
  as a subprocess in `json`, `text`, `noop`, and an invalid format under a
  project-local `GOCOVERDIR`; the JSON Lines stream is decoded with
  `reporter.ProgressEvent` (unknown fields rejected), timestamps must be
  RFC3339, and the last event must be `complete`. The suite rejects absent or
  zero reporter covdata. It is part of `go test ./...` and of CI's Unit Tests
  and Race Detection jobs.
- **Baseline guard** — [`tests/baseline/baseline_test.go`](tests/baseline/baseline_test.go)
  holds the repository to the frostyard repository baseline (core ADR-0071,
  `docs/specs/repository-baseline.md`): 1 test per applicable row, B1 to B17,
  reading the tracked tree with `git ls-files -s -z` and the contents from
  disk, and skipping when git is unusable. B9 is exempt (a library: pkg.go.dev
  documents it) and B11 is a follow-up. The pure checks (the `uses:` pin, the
  checkout credentials, the top-level permissions) have their own table tests.
  It runs in `go test ./...`, so in `make verify`, `make ci` and CI. A new
  baseline row needs its test here.
- New or changed behavior needs a focused test including a failure path;
  exact text formatting is pinned by the tests and by
  [docs/specs/reporter-package.md](docs/specs/reporter-package.md) — change
  them together.

## Continuous integration

[`.github/workflows/ci.yml`](.github/workflows/ci.yml) runs on every push to
`main`, every pull request, and every merge-queue branch (`merge_group`):
**Lint** (the golangci-lint release
`mise.toml` pins, installed from `mise.lock` by `jdx/mise-action`, with
`.golangci.yml`, over the module and the `_examples/` programs), **Security Scan** (`govulncheck ./...` with
`golang.org/x/vuln/cmd/govulncheck@v1.7.0` pinned — reachable Go
standard-library advisories, since the module has no dependencies), **Unit
Tests** (`go test -v ./...` with an in-process coverage profile, plus the e2e
harness's independently asserted instrumented-subprocess signal, then the
95.0% total statement-coverage floor on the in-process profile via
`make coverage-check`
([`scripts/check-coverage.sh`](scripts/check-coverage.sh)) after
`make test-coverage-check` self-tests it; the profile is uploaded as the
`coverage-profile` artifact),
**Race Detection** (`go test -race -short ./...`), **Examples Cross-Arch
Build** (every `_examples/` program directory `GOOS=linux GOARCH=amd64` and
`GOOS=linux GOARCH=arm64` built, the hosted half of what `make ci` mirrors —
core ADR-0038), **Verify** (`make ci`, the canonical gate, in 1 step: form 1
of ADR-0073, the CI entry of the ADR-0071 row B12; `make ci` runs `make
verify` first, which covers the tidy diff and the absent `go.sum`, `gofmt`,
the linter at the pin, `go vet` over the module and the `_examples/`
programs, and the tests — so the job repeats no step by hand), **Docs
integrity** (`node
scripts/check-docs.mjs` — every doc indexed in `docs/README.md`, every
relative link resolving, every symlink alias intact, thresholds in
[`.coverage-thresholds.json`](.coverage-thresholds.json), all `1.0`,
`never_relax: true`), **Rust crate** (`make rust-check` on the Rust
toolchain that `mise.toml` pins, installed by `jdx/mise-action`: cargo fmt,
clippy, tests, Windows and macOS check builds, no dependency), and
**Release config** (`goreleaser check` over
[`.goreleaser.yaml`](.goreleaser.yaml), so a broken release configuration
fails pre-merge instead of after an immutable tag is pushed; it runs the
GoReleaser Pro distribution at the same action SHA `release.yml` uses,
because the config sets `pro: true`, and needs the org secret
`GORELEASER_KEY`). Actions are pinned by commit SHA with least-privilege
permissions (core ADR-0021), every checkout sets `persist-credentials:
false` (baseline row B5), and a job's token scope sits on the job. `make
check` reproduces the Go half locally.

[`.github/workflows/risk-paths.yml`](.github/workflows/risk-paths.yml) runs
core's reusable risk-paths workflow (core ADR-0068) on every pull request,
through `pull_request_target`, so a pull request cannot edit itself out of
the check. It reads
[`.github/risk-paths.json`](.github/risk-paths.json), the list of the paths
this repository treats as high risk (the workflows, the release surface, the
governance policy, the `Makefile` and the `mise` pins, the gate's scripts),
and adds the label `risk:high` to a pull request that touches 1. The caller
is pinned to a full core commit SHA, and the job checks out nothing.

## Releases

std is a library: a release is a Git tag plus a GitHub release with a
changelog — no binaries, packages, or archives.

1. The operator runs `make bump` on a clean, checked `main`: it checks that
   `svu` is the `mise.toml` pin (`make svu-version-check`), runs `make
   check`, asks [`svu next`](.svu.yml) for the next semantic version from
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
  (`.goreleaser.yaml`, `.svu.yml`, `.github/workflows/release.yml`) are
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
- [`SECURITY.md`](SECURITY.md) is the repository's security policy (baseline
  row B14, core ADR-0040): the supported version, GitHub private
  vulnerability reporting, and the scope — the journal transport, the
  reader and record validation as untrusted input, and the crash files of
  `CrashOutput`.

## Documentation

**update documentation** After any change to source code, update relevant
documentation in `AGENTS.md`, [`README.md`](README.md), and the `docs/`
tree. A task is not complete
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
