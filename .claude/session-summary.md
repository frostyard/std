# Session summary

Ephemeral session state — agents replace the block below at session end
(session state lives in `.claude/`). Durable learnings go to
[.memory/](../.memory/README.md), never here; ongoing work is tracked in
GitHub issues (`gh --repo frostyard/std`). Never include credentials,
tokens, private user data, or raw command output; link issues, PRs, and
commits instead of copying logs.

## Current state

- The repository baseline PR (core ADR-0071, rows B1 to B17): `make ci` runs
  `make verify` first and the CI `Verify` job runs `make ci`
  ([ADR-0073](../docs/org-adrs.md) form 1), `.github/risk-paths.json` with its
  caller workflow adds the label `risk:high`, the PR template has a risk tier
  section, every checkout sets `persist-credentials: false`, `SECURITY.md` and
  a kind-`library` `README.md` exist, `tests/baseline/` guards the rows, and
  the shipped 2026-03 plans are gone from `docs/plans/` (git history keeps
  them).

## Last landed

- #95 the Rust crate `frostyard-problems` in `rust/`; #94 the `problems`
  package; #93 the agent-skills sync from frostyard/core.

## Next

- The B11 follow-up: move `go.mod`'s toolchain to Go 1.27 with the
  golangci-lint release the frostyard-go-repo skill's "Fleet toolchain"
  section names.
