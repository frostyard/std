# Review a pull request

Review the given frostyard/std PR against the repo rubric. You are reviewing,
not merging: never approve-and-merge in one act, and never merge a PR you
authored (mechanically backed by `.claude/settings.json`, declared in
`policies/agent-governance.json`,
[ADR-0001](../../docs/adr/0001-acmm-conformance-via-canonical-aliases.md)).
Automated feedback is advisory: do not approve changes, weaken required
checks, or claim verification passed without evidence from the pull request.

1. Read [AGENTS.md](../../AGENTS.md) — the architecture, conventions, and
   documentation rules the diff must satisfy. In particular:
   - **Stdlib only** — no external dependencies anywhere in the module;
     `go.mod` stays `require`-free.
   - **Interface-driven** — consumers depend on `reporter.Reporter`, never on
     concrete types; `IsJSON()` is the only runtime discriminator; exact
     output formats are a contract pinned by
     [docs/specs/reporter-package.md](../../docs/specs/reporter-package.md).
   - **Thread-safety table** — `JSONReporter` is mutex-guarded,
     `TextReporter` is not, `NoopReporter` has no state; a change must not
     silently move a row.
   - **Go 1.26 idioms** — range-over-int, `omitzero`, `any`, standard
     `testing` only.
2. Apply every row of the
   [PR review rubric](../../docs/specs/pr-review-rubric.md)
   (`docs/review-rubric.md` resolves to the same file). Check each row
   independently; cite file and line for every failure.
3. Run the gates the rubric names:
   - `make check` (gofmt, golangci-lint with `.golangci.yml`, `go test -v
     ./...` including `tests/e2e`)
   - `go test -race -short ./...`
   - `node scripts/check-docs.mjs`
4. If the diff changes the `Reporter` interface, `ProgressEvent`, or any
   implementation's output, verify
   [docs/specs/reporter-package.md](../../docs/specs/reporter-package.md) and
   [docs/design/overview.md](../../docs/design/overview.md) changed alongside
   the code, and that the `_examples/` programs still pass the e2e suite.
5. If the diff touches `.github/workflows/**`, `.goreleaser.yaml`,
   `.svu.yaml`, or `policies/agent-governance.json`, treat it as high risk:
   confirm every action is SHA-pinned with least-privilege permissions and
   that a human reviews it.
6. Report findings as review comments ordered by severity, labelled
   blocking / non-blocking / question / nit per the rubric; state plainly
   when a row passes. A PR with any failing rubric row gets "request
   changes", not silence.
