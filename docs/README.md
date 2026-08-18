# Documentation

Docs are split by the question they answer (frostyard/core's four-category
shape, [core ADR-0025](https://github.com/frostyard/core/blob/main/docs/adr/0025-consolidate-repository-docs-into-docs.md)):

| Directory | Question | Contents |
| --- | --- | --- |
| [adr/](adr/) | **Why** did we choose this? | Repo-local Architecture Decision Records — immutable once accepted; superseded, never edited. Org-wide decisions live in frostyard/core — see [org-adrs.md](org-adrs.md) |
| [design/](design/) | **How** does it fit together? | Living documents describing the current architecture |
| [specs/](specs/) | **What exactly** is the contract? | Precise, testable interface definitions |
| [plans/](plans/) | **When/in what order** do we build? | Roadmaps and phase plans; updated as work lands |

## Index

### Decisions (ADRs)

Org-wide decisions binding this repo are listed in [org-adrs.md](org-adrs.md).

- [0001 — ACMM conformance via canonical aliases](adr/0001-acmm-conformance-via-canonical-aliases.md)
  — one canonical file per criterion behind committed relative symlinks
  (the alias registry), real trees for directory criteria, the docs-integrity
  gate, and the CI, release, and agent-governance surfaces added with it

### Design

- [Overview](design/overview.md) — purpose, architecture, key patterns,
  configuration, downstream consumers (the entry-point doc)
- [Quality loop](design/quality-loop.md) — declare → review → gate → learn →
  observe, wired to `ci.yml`, the docs gate, `.memory/`, and the release path
  (`docs/quality.md` is its alias)

### Specs

- [Reporter package](specs/reporter-package.md) — the `Reporter` interface,
  `ProgressEvent` JSON Lines format, and exact per-implementation output
  formatting
- [PR review rubric](specs/pr-review-rubric.md) — the checklist every review
  applies, rows = the repo's verifiable gates (`docs/review-rubric.md` is its
  alias)
- [PR acceptance metric](specs/pr-acceptance-metric.md) — the monthly
  acceptance-rate definition and rules (`docs/metrics.md` is its alias)

### Plans

Shipped work; kept for the decision context they carry:

- [clix design](plans/2026-03-04-clix-design.md) and
  [clix implementation](plans/2026-03-04-clix-implementation.md) — the CLI
  convenience module built on this package, shipped as
  [frostyard/clix](https://github.com/frostyard/clix)
- [Reporter extraction design](plans/2026-03-04-reporter-extraction-design.md)
  and [implementation plan](plans/2026-03-04-reporter-extraction.md) —
  extracting `reporter` from nbc into this module
- [Reporter examples design](plans/2026-03-04-reporter-examples-design.md)
  and [implementation plan](plans/2026-03-04-reporter-examples.md) — the
  `_examples/` programs

## Conventions

- **New docs start from their category's `TEMPLATE.md`** (in each directory).
- New decision → new ADR with the next number; if it reverses an old one, mark
  the old one `Superseded by NNNN` rather than editing it. Decisions that bind
  more than this repo become ADRs in
  [frostyard/core](https://github.com/frostyard/core/tree/main/docs/adr) plus
  a line in [org-adrs.md](org-adrs.md).
- Design docs are updated in place to always reflect reality.
- Specs change only alongside the code that implements them.
- Cross-links between categories are mandatory in both directions.
- Adding a doc means adding it to the index above; `node
  scripts/check-docs.mjs` (CI's `docs-gate` job) fails on an unindexed doc,
  a dead relative link, or a broken alias.
- Conformance aliases (`docs/metrics.md`, `docs/review-rubric.md`,
  `docs/quality.md` — [ADR-0001](adr/0001-acmm-conformance-via-canonical-aliases.md))
  are not docs: never index or edit them; edit their targets.
