# Org-wide decisions (frostyard/core ADRs)

Conventions this repository follows that are decided at the org level are
recorded as ADRs in
[frostyard/core](https://github.com/frostyard/core/tree/main/docs/adr).
The ones that bind std:

- [ADR-0012 — svu-derived versions, make bump, and the rolling dev prerelease](https://github.com/frostyard/core/blob/main/docs/adr/0012-svu-versioning-and-rolling-dev-prerelease.md) — `.svu.yaml` and `make bump` tag releases; as a library with no released binary, the goreleaser dev-prerelease half does not apply
- [ADR-0018 — Org-wide agent instruction and knowledge surfaces](https://github.com/frostyard/core/blob/main/docs/adr/0018-org-wide-agent-instruction-and-knowledge-surfaces.md) — `AGENTS.md` is canonical; `CLAUDE.md`, `GEMINI.md`, and `.cursorrules` are symlinks to it
- [ADR-0022 — make ci is the canonical gate; TestI* is reserved](https://github.com/frostyard/core/blob/main/docs/adr/0022-make-ci-gate-and-test-naming-filter.md) — the local gate here is `make check` (fmt → lint → test); the `TestI` prefix stays reserved for environment-requiring integration tests
- [ADR-0025 — One docs/ tree per repository, in core's four-category shape](https://github.com/frostyard/core/blob/main/docs/adr/0025-consolidate-repository-docs-into-docs.md) — this `docs/` tree (formerly `yeti/`); indexed in [README.md](README.md)
- [ADR-0026 — Distribute core agent skills to repos via sync PRs from core](https://github.com/frostyard/core/blob/main/docs/adr/0026-distribute-core-skills-via-sync-prs.md) — std receives `.agents/skills/` via sync PRs (listed in core's `.github/skills-sync.json`); edit skills in core, not here

Not yet applicable: [ADR-0021](https://github.com/frostyard/core/blob/main/docs/adr/0021-sha-pinned-actions-and-least-privilege-ci.md)
(SHA-pinned, least-privilege CI) — this repo has no workflows yet (only
`.github/dependabot.yml`); it binds the first workflow added.
[ADR-0011](https://github.com/frostyard/core/blob/main/docs/adr/0011-frostyard-prefixed-package-names.md)
(frostyard-prefixed package names) — std is a library and ships no distro
packages.

When changing behavior covered by one of these, update or supersede the ADR
in frostyard/core first, then change this repo in the same effort.
