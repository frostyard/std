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

*(none yet — org-wide decisions binding this repo are listed in
[org-adrs.md](org-adrs.md))*

### Design

- [Overview](design/overview.md) — purpose, architecture, key patterns,
  configuration, downstream consumers (the entry-point doc)

### Specs

- [Reporter package](specs/reporter-package.md) — the `Reporter` interface,
  `ProgressEvent` JSON Lines format, and exact per-implementation output
  formatting

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
- Adding a doc means adding it to the index above.
