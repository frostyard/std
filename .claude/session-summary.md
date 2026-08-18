# Session summary

Ephemeral session state — agents replace the block below at session end
(session state lives in `.claude/`). Durable learnings go to
[.memory/](../.memory/README.md), never here; ongoing work is tracked in
GitHub issues (`gh --repo frostyard/std`). Never include credentials,
tokens, private user data, or raw command output; link issues, PRs, and
commits instead of copying logs.

## Current state

- ACMM conformance and fleet prerequisites landed (2026-08-18):
  [ADR-0001](../docs/adr/0001-acmm-conformance-via-canonical-aliases.md)'s
  alias lattice (`AGENTS.md` canonical; `docs/specs/pr-review-rubric.md`,
  `docs/specs/pr-acceptance-metric.md`, `docs/design/quality-loop.md`
  canonical with `docs/review-rubric.md`, `docs/metrics.md`,
  `docs/quality.md` aliases), the docs-integrity gate
  (`scripts/check-docs.mjs`, `docs-gate` job in `ci.yml`), the first CI
  workflow, the `tests/e2e/` example-program suite, `.claude/settings.json`
  tool-layer limits, `policies/agent-governance.json` (Fluent enrollment
  surface), and the changelog-only release path
  (`.goreleaser.yaml` + `release.yml`).

## Last landed

- #20 skills sync from frostyard/core; #19 concurrent JSON reporter test;
  #16 dependabot github-actions ecosystem dropped (restored by the
  conformance PR now that workflows exist).

## Next

- #17 (CI test and coverage reporting): `ci.yml` now runs the tests; a
  coverage artifact or baseline is still open — separate work item.
- First `make bump` after the release workflow merges needs the org
  `GORELEASER_KEY` secret visible to this repository.
