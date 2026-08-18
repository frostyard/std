# 0001 — ACMM conformance via canonical aliases

- **Status:** Accepted
- **Date:** 2026-08-18

## Context

The Hive ACMM evaluation grades repositories by checking that fixed paths
exist — an e2e test tree, PR and issue templates, a code-style config, a
coverage gate, a prompts catalog, an EditorConfig, a corrections inbox, a PR
acceptance metric, a review rubric, a quality dashboard, agent-safety
settings, a session summary — and states for each that "the content can
follow your project's conventions." Hive itself is retired and std never
received `acmm` issues, but the same paths are the prerequisites for
agentic-fleet management: frostyard/core's repository-surfaces contract v1
requires `AGENTS.md`, `policies/agent-governance.json`, `.agents/skills/`,
and `docs/README.md` as real Git blobs and trees before Fluent will enroll a
repository, and the fleet's review and quality loop assumes the rest.

std already held canonical equivalents for part of the list at paths fixed
by org decisions listed in [org-adrs.md](../org-adrs.md): one canonical
`AGENTS.md` with `CLAUDE.md`, `GEMINI.md`, and `.cursorrules` symlinks
(core ADR-0018), synced `.agents/skills/` (core ADR-0026), the four-category
`docs/` tree (core ADR-0025), and `.svu.yaml` for `make bump` (core
ADR-0012). It had no CI workflow, no release workflow, no linter config, no
templates, and no repo-local ADR. Duplicating content into ACMM's paths would
guarantee drift — exactly what core ADR-0002 rejected. frostyard/core solved
the identical list with
[core ADR-0029](https://github.com/frostyard/core/blob/main/docs/adr/0029-acmm-conformance-via-canonical-aliases.md);
repogen and updex followed with their own ADR-0012s.

## Decision

ACMM's required paths are satisfied by **committed relative symlinks to
canonical content** wherever a canonical equivalent exists, and by genuinely
new artifacts only where none does. Canonical content lives where org
conventions put it — the four-category `docs/` tree and `AGENTS.md` — never
at the ACMM path.

The alias table (edit the targets, never the aliases):

| Alias | Target | Criterion |
| --- | --- | --- |
| `CLAUDE.md` | `AGENTS.md` | agent surface (core ADR-0002/0018) |
| `GEMINI.md` | `AGENTS.md` | agent surface (core ADR-0002/0018) |
| `CONTRIBUTING.md` | `AGENTS.md` | contributing guide |
| `.cursorrules` | `AGENTS.md` | cursor rules (core ADR-0018) |
| `.github/copilot-instructions.md` | `../AGENTS.md` | agent surface (core ADR-0002) |
| `.claude/skills` | `../.agents/skills` | simple skills |
| `docs/metrics.md` | `specs/pr-acceptance-metric.md` | PR acceptance metric |
| `docs/review-rubric.md` | `specs/pr-review-rubric.md` | PR review rubric |
| `docs/quality.md` | `design/quality-loop.md` | quality dashboard |

Rules:

- **Directory criteria always get real git trees** (`tests/e2e/`,
  `.github/ISSUE_TEMPLATE/`, `.github/prompts/`, `.memory/`, `.agents/skills/`,
  `policies/`) — an evaluator reading the git tree via API sees a symlink as a
  blob, not a tree, and core's surface contract requires canonical
  directories to be real trees.
- **Aliases are not docs**: they get no `docs/README.md` index entries and
  carry no cross-link obligations; the canonical target does.
- **Enrollment surfaces are real blobs at their canonical paths**:
  `AGENTS.md`, `policies/agent-governance.json`, and `docs/README.md` are
  regular files, and `.agents/skills/` a real tree, because Fluent's importer
  never follows aliases.
- Genuinely new artifacts, each doing real work: the merged `AGENTS.md`
  (now also the contributing guide, with `## Getting started`, `## Testing`,
  `## Releases`, `## Commits and pull requests`, and `## Agent limits and
  governance`); [`tests/e2e/`](../../tests/e2e/README.md) — a real
  black-box suite that builds every `_examples/*` program and runs it in
  every output mode, decoding the JSON Lines stream with
  `reporter.ProgressEvent` (the examples were previously compiled by
  nothing, since `go test ./...` skips underscore directories);
  `.github/pull_request_template.md` mirroring `make check`, the docs gate,
  docs housekeeping, and the aliases rule; `.github/ISSUE_TEMPLATE/`
  (`config.yml` with `blank_issues_enabled: true`, bug and feature
  templates); `.golangci.yml` (v2 schema, `standard` linters plus the
  `gofmt` formatter — the config `make lint` and CI both run);
  `.coverage-thresholds.json` enforced by `scripts/check-docs.mjs` in the
  new `docs-gate` CI job (docs-index coverage, link integrity, symlink
  resolution, all `1.0`, `never_relax: true`); `.github/prompts/README.md`
  and `review.prompt.md`; `.editorconfig`; the `.memory/` inbox with core
  ADR-0018's append-only five-field schema;
  [specs/pr-acceptance-metric.md](../specs/pr-acceptance-metric.md),
  [specs/pr-review-rubric.md](../specs/pr-review-rubric.md), and
  [design/quality-loop.md](../design/quality-loop.md); `.claude/settings.json`
  denying merge-own-PR, approve-own-work, release-publishing, and pushes to
  `main` at the tool layer; and `.claude/session-summary.md`.
- The same PR adds the two things std lacked that the fleet contract and the
  release convention require, recorded here because they share the
  rationale: `.github/workflows/ci.yml` (Lint, Unit Tests, Race Detection,
  Verify, Docs integrity — SHA-pinned, least-privilege, per core ADR-0021,
  which now binds this repo) and the release surface — `.goreleaser.yaml`
  (`builds: [{ skip: true }]`, changelog grouped by Conventional Commit type,
  GitHub release only) plus `.github/workflows/release.yml` on tag push, so
  the tag `make bump` pushes becomes a GitHub release with a changelog and
  nothing else (core ADR-0012's library half); and
  `policies/agent-governance.json`, std's agent-governance surface under
  core's repository-surfaces contract v1 (deny by default, six actions,
  `workflow-and-permissions` and `release-and-publication` boundaries
  review-required at high risk).

## Consequences

- One canonical body of content per criterion; conformance paths cannot
  drift from it.
- GitHub's web renderer shows a symlinked `.md` as its target path rather
  than its content; checkouts on Windows need `core.symlinks=true` or WSL.
- The alias table above is the registry; adding or removing an alias means
  amending it here (a new ADR if the mechanism itself changes).
- `scripts/check-docs.mjs` fails CI on any broken alias, unindexed doc, or
  dead relative link, making the lattice self-guarding.
- `make test` now includes `tests/e2e`, which builds and runs the example
  programs (a few seconds); the examples can no longer rot silently.
- std has workflows now, so core ADR-0021 (SHA-pinned, least-privilege CI)
  binds it; `.github/workflows/**` is a review-required high-risk boundary
  in `policies/agent-governance.json`.
- Contingency: if the ACMM evaluator rejects a symlink for one of the file
  criteria (contributing guide, cursor rules, metric, rubric, dashboard),
  that alias is replaced by a real stub file pointing at the canonical doc —
  a one-commit change that does not reverse this decision.

## Alternatives considered

- **Real duplicate files at the ACMM paths:** guaranteed drift; rejected for
  the same reason core ADR-0002 rejected per-tool instruction copies.
- **Content-free stub files:** a second class of "doc" that the index and
  cross-link rules would nominally govern; symlinks are aliases, not docs.
- **A `tests/e2e/README.md` that only points at the unit tests:** a
  content-free tree; the example programs are a genuine black-box surface
  and were untested, so a real suite was the honest answer.
- **Skip the release workflow because std ships no binary:** leaves `make
  bump` producing tags that nothing turns into a release; a changelog-only
  GoReleaser run is the library-shaped half of core ADR-0012.

## References

- Shapes: [design/quality-loop.md](../design/quality-loop.md),
  [specs/pr-review-rubric.md](../specs/pr-review-rubric.md),
  [specs/pr-acceptance-metric.md](../specs/pr-acceptance-metric.md),
  [design/overview.md](../design/overview.md)
- Pattern source:
  [core ADR-0029](https://github.com/frostyard/core/blob/main/docs/adr/0029-acmm-conformance-via-canonical-aliases.md),
  [repogen ADR-0012](https://github.com/frostyard/repogen/blob/main/docs/adr/0012-acmm-conformance-via-canonical-aliases.md),
  [updex ADR-0012](https://github.com/frostyard/updex/blob/main/docs/adr/0012-acmm-conformance-via-canonical-aliases.md)
- Builds on the org decisions in [org-adrs.md](../org-adrs.md): core
  ADR-0002, 0012, 0018, 0019, 0021, 0025, 0026, and the
  [repository-surfaces contract v1](https://github.com/frostyard/core/blob/main/docs/specs/organization-repository-enrollment.md)
