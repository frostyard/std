# Org-wide decisions (frostyard/core ADRs)

Conventions this repository follows that are decided at the org level are
recorded as ADRs in
[frostyard/core](https://github.com/frostyard/core/tree/main/docs/adr).
The ones that bind std:

- [ADR-0002 — Agent-portable instruction surface](https://github.com/frostyard/core/blob/main/docs/adr/0002-agent-portable-instruction-surface.md) — one canonical instruction file with tool-path symlinks; std's alias set is registered in [ADR-0001](adr/0001-acmm-conformance-via-canonical-aliases.md)
- [ADR-0012 — svu-derived versions, make bump, and the rolling dev prerelease](https://github.com/frostyard/core/blob/main/docs/adr/0012-svu-versioning-and-rolling-dev-prerelease.md) — `.svu.yaml` and `make bump` tag releases; the tag triggers `.github/workflows/release.yml`, a changelog-only GoReleaser run (`.goreleaser.yaml`, `builds: [{ skip: true }]`); as a library with no released binary, the rolling dev-prerelease half does not apply
- [ADR-0018 — Org-wide agent instruction and knowledge surfaces](https://github.com/frostyard/core/blob/main/docs/adr/0018-org-wide-agent-instruction-and-knowledge-surfaces.md) — `AGENTS.md` is canonical; `CLAUDE.md`, `GEMINI.md`, `CONTRIBUTING.md`, `.cursorrules`, and `.github/copilot-instructions.md` are symlinks to it; `.memory/corrections.jsonl` and `.github/prompts/*.prompt.md` follow its shapes
- [ADR-0019 — Governance as code and risk tiers](https://github.com/frostyard/core/blob/main/docs/adr/0019-governance-as-code-and-risk-tiers.md) — deny-by-default agent limits (`.claude/settings.json`, `policies/agent-governance.json`) and the `never_relax` guardrail in `.coverage-thresholds.json`
- [ADR-0021 — SHA-pinned actions and least-privilege CI](https://github.com/frostyard/core/blob/main/docs/adr/0021-sha-pinned-actions-and-least-privilege-ci.md) — binds `.github/workflows/ci.yml` and `release.yml`; Dependabot's `github-actions` ecosystem keeps the pins current
- [ADR-0025 — One docs/ tree per repository, in core's four-category shape](https://github.com/frostyard/core/blob/main/docs/adr/0025-consolidate-repository-docs-into-docs.md) — this `docs/` tree (formerly `yeti/`); indexed in [README.md](README.md)
- [ADR-0026 — Distribute core agent skills to repos via sync PRs from core](https://github.com/frostyard/core/blob/main/docs/adr/0026-distribute-core-skills-via-sync-prs.md) — std receives `.agents/skills/` via sync PRs (listed in core's `.github/skills-sync.json`); edit skills in core, not here
- [ADR-0029 — ACMM conformance via canonical aliases](https://github.com/frostyard/core/blob/main/docs/adr/0029-acmm-conformance-via-canonical-aliases.md) — the pattern std's [ADR-0001](adr/0001-acmm-conformance-via-canonical-aliases.md) applies
- [ADR-0038 — make ci stays canonical; the TestI/Integration name filter is chairlift-only](https://github.com/frostyard/core/blob/main/docs/adr/0038-scope-the-test-name-filter-to-chairlift.md) — `make ci` is the credential-free gate mirroring CI's jobs in CI's fail-fast order; std is not chairlift, so no test-name filter applies and every hermetic test runs in the gate and in pull-request CI
- [Repository enrollment spec (repository-surfaces contract v1)](https://github.com/frostyard/core/blob/main/docs/specs/organization-repository-enrollment.md) — `AGENTS.md`, `policies/agent-governance.json`, `.agents/skills/`, and `docs/README.md` are the canonical surfaces Fluent reads to enroll std; they must be real blobs/trees, never aliases

Not applicable:
[ADR-0011](https://github.com/frostyard/core/blob/main/docs/adr/0011-frostyard-prefixed-package-names.md)
(frostyard-prefixed package names) — std is a library and ships no distro
packages.

When changing behavior covered by one of these, update or supersede the ADR
in frostyard/core first, then change this repo in the same effort.
