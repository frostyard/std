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
  runs on every PR and push to `main`, SHA-pinned with least-privilege
  permissions:
  - *Lint* — golangci-lint v2 configured by
    [`.golangci.yml`](../../.golangci.yml) (`standard` linters, `gofmt`
    formatter); the same file `make lint` uses.
  - *Unit Tests* — `go test -v ./...`: `reporter/` unit tests plus the
    [e2e suite](../../tests/e2e/README.md), which builds and runs every
    `_examples/*` program in every output mode.
  - *Race Detection* — `go test -race -short ./...`.
  - *Verify* — `go mod tidy` leaves no diff (the stdlib-only constraint is
    visible here: `go.mod` never gains a `require`), `go vet ./...`,
    `gofmt -l .` empty.
  - *Docs integrity* (`docs-gate`): `node scripts/check-docs.mjs` checks
    docs-index coverage, relative-link integrity, symlink resolution, and the
    metric's pinned headings against
    [.coverage-thresholds.json](../../.coverage-thresholds.json) — all 1.0,
    `never_relax: true` (the loop may tighten, never loosen).
  - `make check` (fmt → lint → test) reproduces the Go half locally.
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
