# Spec: PR review rubric

One paragraph: the checklist every frostyard/std pull-request review applies,
kept consistent, actionable, and focused on the risks the pull request
introduces. Consumers: human reviewers, the
[review runbook](../../.github/prompts/review.prompt.md), and the
[PR template](../../.github/pull_request_template.md), whose sections mirror
these checks. `docs/review-rubric.md` is a conformance alias for this file
([ADR-0001](../adr/0001-acmm-conformance-via-canonical-aliases.md)).

## Interface

Every review verifies each row; a PR merges only when all applicable rows
pass.

| Check | How to verify |
| --- | --- |
| Correctness and scope | The change solves the linked problem and handles the relevant error cases; the diff is focused — no unrelated refactors or generated artifacts (`coverage.out`, `coverage.html`, `dist/`). |
| Stdlib only | No `require` in `go.mod` and no `go.sum` (the Verify job checks both); every import in `reporter/`, `_examples/`, and `tests/e2e/` is Go standard library or `github.com/frostyard/std/...`. |
| API and contract | Changes to the `Reporter` interface, `ProgressEvent`, `EventType`, or any implementation's exact output are intentional, backward-compatible unless the PR says otherwise, and reflected in [specs/reporter-package.md](reporter-package.md) in the same PR. |
| Build gate green | `make check` passes: `gofmt -w` leaves no diff, `golangci-lint run` (`.golangci.yml`) reports no issues, `go test -v ./...` passes — the same steps as the Lint, Unit Tests, and Verify jobs in [`.github/workflows/ci.yml`](../../.github/workflows/ci.yml); the Race Detection job (`go test -race -short ./...`) also passes. |
| Tests | New or changed behavior has a focused test including a failure path (`reporter/*_test.go`, one file per implementation, `bytes.Buffer` capture, JSON fields unmarshalled and checked); example programs still pass the [e2e suite](../../tests/e2e/README.md) (`go test ./tests/e2e/...`). |
| Docs housekeeping | `AGENTS.md`, `docs/design/overview.md`, and `docs/specs/*` reflect the behavior change; new docs start from their category `TEMPLATE.md`, are indexed in [docs/README.md](../README.md), and cross-link both ways; a new significant decision ⇒ ADR first, in the same change. |
| Docs-integrity gate green | `node scripts/check-docs.mjs` passes: every doc indexed, every relative link resolving, every symlink alias intact (thresholds in `.coverage-thresholds.json`). |
| Aliases untouched | Conformance aliases ([ADR-0001](../adr/0001-acmm-conformance-via-canonical-aliases.md)) are not edited directly; canonical targets are. |
| Protected boundaries | Changes under `.github/workflows/**`, `.goreleaser.yaml`, `.svu.yaml`, or `policies/agent-governance.json` are called out as high risk and reviewed by a human (`policies/agent-governance.json`); new actions are SHA-pinned with least-privilege permissions. |
| Conventional title | The PR title (or lone commit subject) is `type(scope): summary`, since the squash commit is what `svu` versions and the release changelog groups by. |
| Agent limits respected | The PR was not merged, approved, released, or tagged by the agent that authored it; mechanically backed by `.claude/settings.json` and declared in `policies/agent-governance.json`. |

## Rules

- Each check is independently verifiable from the PR diff plus the commands
  named in its row — a review MUST NOT rely on out-of-band context.
- Label findings by impact:
  - **Blocking:** a correctness, compatibility, dependency, or required
    test/documentation issue that must be resolved before approval.
  - **Non-blocking:** a worthwhile improvement that does not prevent merging.
  - **Question:** a request for context or clarification, not an assumed
    defect.
  - **Nit:** an optional minor style suggestion; avoid nits already enforced
    by `gofmt` or `.golangci.yml`.
- Comments identify the affected behavior, explain its impact, and suggest a
  concrete resolution. Reviewers re-check resolved blocking findings and
  confirm required CI checks pass before approval.
- Rubric changes ride with the artifact that enforces them (the gate script,
  the workflow, or the template) in the same PR.
- The org squash-merges: the review covers the squashed result, not
  intermediate commits.

## References

- Rationale: [ADR-0001](../adr/0001-acmm-conformance-via-canonical-aliases.md)
- Context: [design/quality-loop.md](../design/quality-loop.md)
- Related: [specs/reporter-package.md](reporter-package.md) (the contract
  the API row protects)
