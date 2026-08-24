# Quality loop

Living document. Rationale:
[ADR-0001](../adr/0001-acmm-conformance-via-canonical-aliases.md).
Contracts: [specs/pr-review-rubric.md](../specs/pr-review-rubric.md),
[specs/pr-acceptance-metric.md](../specs/pr-acceptance-metric.md).
`docs/quality.md` is a conformance alias for this file (ADR-0001); this page
is also the quality dashboard.

[![CI](https://github.com/frostyard/std/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/frostyard/std/actions/workflows/ci.yml?query=branch%3Amain)

## Overview

How change quality is proposed, gated, observed, and learned from in this
repo. Changes produced with AI assistance follow the same loop as every
other contribution: they are never merged solely on the basis of generated
output. One loop, five stations:

```
PR template ──► review rubric ──► CI gates ──► corrections ──► promotion
(checklist)     (spec)            (ci.yml,     (.memory/)      (AGENTS.md,
     ▲                             docs-gate)                    docs, skills)
     └────────────── acceptance metric (spec) observes the stream ─────────┘
```

## Design

- **Declare** — [.github/pull_request_template.md](../../.github/pull_request_template.md)
  makes every PR walk the build-gate and docs-housekeeping checklists and
  affirm the aliases are untouched; PR titles are Conventional Commits
  because the squash commit is what `svu` versions.
- **Review** — the [PR review rubric](../specs/pr-review-rubric.md) is the
  contract a review applies; the
  [review runbook](../../.github/prompts/review.prompt.md) is its
  task-shaped form for agents. Maintainers remain accountable for the merge
  decision.
- **Gate** — [.github/workflows/ci.yml](../../.github/workflows/ci.yml)
  runs on every PR, every push to `main`, and every merge-queue branch
  (`merge_group`), SHA-pinned with least-privilege permissions:
  - *Lint* — golangci-lint v2 configured by
    [`.golangci.yml`](../../.golangci.yml) (`standard` linters, `gofmt`
    formatter); the same file `make lint` uses. The release is pinned in
    [`mise.toml`](../../mise.toml) with its checksums in
    [`mise.lock`](../../mise.lock) (core ADR-0043): the CI job installs
    exactly that version through `jdx/mise-action`; locally `mise install`
    provisions it, the [`Makefile`](../../Makefile) reads the same pin,
    and `make lint` fails with `mise install` when the binary is absent and
    warns when the version differs. A linter release therefore cannot
    change or disappear from the gate without a failure. It runs twice: once
    over the packages `./...` matches, once over the
    [`_examples/`](#example-programs-are-analyzed-explicitly) program
    directories.
  - *Security Scan* — `govulncheck ./...` with
    `golang.org/x/vuln/cmd/govulncheck@v1.7.0` pinned (the same job clix and
    updex run): with no dependencies the signal is a reachable Go
    standard-library advisory, which every consumer of `reporter` inherits. It
    runs twice: once over the packages `./...` matches, once over the
    [`_examples/`](#example-programs-are-analyzed-explicitly) program
    directories.
  - *Unit Tests* — `go test -v -coverprofile=coverage.out -covermode=atomic
    -coverpkg=github.com/frostyard/std/reporter ./...`: `reporter/` unit tests
    plus the [e2e suite](../../tests/e2e/README.md). `coverage.out` records
    reporter code executed inside those `go test` processes; its **95.0% total
    statement-coverage floor** is enforced with
    [`scripts/check-coverage.sh`](../../scripts/check-coverage.sh) (`make
    coverage-check`, ported from frostyard/clix and updex), after self-testing
    that script against fixture profiles (`make test-coverage-check`) so a
    broken checker cannot silently pass a regression. The profile is uploaded
    as the `coverage-profile` artifact. Separately, the e2e harness builds
    every `_examples/*` program with Go coverage instrumentation, runs every
    output mode with a project-local `GOCOVERDIR`, and rejects missing or zero
    reporter covdata; those subprocess counters are an asserted e2e signal,
    not input to the 95% profile. Observed in-process coverage is 97.8%. The
    floor may tighten, never loosen; there is no coverage service, and `make
    test-cover` still produces a local `coverage.html`.
  - *Race Detection* — `go test -race -short ./...`.
  - *Verify* — `go mod tidy` leaves no diff (the stdlib-only constraint is
    visible here: `go.mod` never gains a `require`), `go vet ./...` plus a
    second `go vet` over the
    [`_examples/`](#example-programs-are-analyzed-explicitly) program
    directories, `gofmt -l .` empty. `make verify` runs this same
    credential-free, non-mutating gate locally.
  - *Docs integrity* (`docs-gate`): `node scripts/check-docs.mjs` checks
    docs-index coverage, relative-link integrity, symlink resolution, and the
    metric's pinned headings against
    [.coverage-thresholds.json](../../.coverage-thresholds.json) — all 1.0,
    `never_relax: true` (the loop may tighten, never loosen).
  - *Release config* (`release-config`): the job reports on every trigger, but
    `goreleaser check` validates
    [`.goreleaser.yaml`](../../.goreleaser.yaml) only when
    `GORELEASER_KEY` is available. That guarded action runs the GoReleaser Pro
    distribution at the same action SHA `release.yml` uses because the config
    sets `pro: true` and OSS GoReleaser refuses to validate Pro fields. Without
    the key, the complementary guard emits `GORELEASER_KEY unavailable for this
    run; skipping goreleaser check.` and the job succeeds without validation.
  - `make check` (fmt → lint → vet → test) reproduces the Go half locally,
    including both passes over the example programs.

### Example programs are analyzed explicitly

Go package patterns ignore directories whose name begins with `_`, so
neither `./...` nor `./_examples/...` matches the
[`_examples/`](../../tests/e2e/README.md) programs — before this coverage
existed, `go vet ./...` and `golangci-lint run` reported them clean because
they never looked at them. The analyzers therefore receive the example
package directories by name. [`scripts/example-dirs.sh`](../../scripts/example-dirs.sh)
enumerates them at run time (every `_examples/*/` directory holding at least
one `.go` file) and exits non-zero when the list is empty, so a silent
regression to analyzing nothing fails the gate. The CI Lint, Security Scan,
and Verify jobs and the `lint` and `vet` Makefile targets all call that one
script, which is what keeps the [`.golangci.yml`](../../.golangci.yml) promise
that local and CI findings agree, and means a new `_examples/<program>/` is
analyzed without editing [ci.yml](../../.github/workflows/ci.yml) or the
[`Makefile`](../../Makefile).
- **Learn** — corrections land in
  [.memory/corrections.jsonl](../../.memory/README.md) (append-only,
  five-field schema) and are promoted into `AGENTS.md`, docs, or skills;
  promotion is the only sanctioned duplication. Session continuity lives in
  [.claude/session-summary.md](../../.claude/session-summary.md).
- **Enforce mechanically** — [.claude/settings.json](../../.claude/settings.json)
  denies the forbidden acts at the tool layer: merging PRs (`gh pr merge`),
  approving own work (`gh pr review --approve`), publishing releases
  (`gh release`), pushing to `main`, and force-pushing;
  [policies/agent-governance.json](../../policies/agent-governance.json)
  states the same limits as machine-readable policy under frostyard/core's
  repository-surfaces contract v1 (deny by default; workflows and the release
  surface review-required at high risk) — it is also the surface Fluent reads
  to enroll std.
- **Observe** — the [PR acceptance metric](../specs/pr-acceptance-metric.md)
  summarizes the stream; it informs, never gates.

## Release path

The loop ends at a tag: the operator's `make bump` runs `make check`, asks
`svu next` for the version, tags, and pushes; the tag triggers
[.github/workflows/release.yml](../../.github/workflows/release.yml), a
GoReleaser Pro run of [`.goreleaser.yaml`](../../.goreleaser.yaml) that skips
builds and publishes a changelog-only GitHub release. Both files are a
review-required boundary; agents never tag or release.

A tag is immutable, so trusted runs with `GORELEASER_KEY` use the *Release
config* gate to move a `.goreleaser.yaml` error from after the tag to before
the merge. Secretless runs do not validate `.goreleaser.yaml`; they still
report the stable job context through the explicit skip step, so maintainers
must rely on a trusted validation run before tagging.

## Operational notes

Re-run every gate locally before pushing:

```
make check
node scripts/check-docs.mjs
go test -race -short ./...        # what the Race Detection job runs
```

Failure modes: a broken alias or missing index line fails docs-gate (fix the
canonical target or the index, never the alias); a lint finding after a
golangci-lint upgrade means the gate was already red — fix the finding, never
loosen `.golangci.yml`; an e2e failure means an example program stopped
building or emitting a valid JSON Lines stream ending in `complete` — fix the
example or the reporter, and update
[specs/reporter-package.md](../specs/reporter-package.md) if the contract
moved; a `Verify` failure on `go.mod` almost always means a dependency crept
in — remove it, std is stdlib-only.

## References

- Rationale: [ADR-0001](../adr/0001-acmm-conformance-via-canonical-aliases.md)
- Contracts: [specs/pr-review-rubric.md](../specs/pr-review-rubric.md),
  [specs/pr-acceptance-metric.md](../specs/pr-acceptance-metric.md)
- Where it fits: [design/overview.md](overview.md)
- Built in: the conformance PR recorded by ADR-0001 (this repo keeps no
  roadmap; shipped plans live in [plans/](../plans/))
