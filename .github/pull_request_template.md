<!-- The org squash-merges: branch off main, never stack on another PR's
branch. Title and commits use Conventional Commits (`type(scope): summary`)
— the squash commit is what `svu next` versions and the release changelog
groups by. Reviews apply docs/specs/pr-review-rubric.md. -->

## Summary

<!-- What changes and why, in a few sentences. Link the issue(s) this
closes. -->

## Checks

<!-- The build gate from AGENTS.md — run before opening the PR. -->

- [ ] `make check` — gofmt leaves no diff, `golangci-lint run`
      (`.golangci.yml`) clean, `go test -v ./...` green (unit + `tests/e2e`)
- [ ] `go test -race -short ./...` green
- [ ] `go.mod` still declares no dependencies (stdlib only)
- [ ] New or changed behavior has focused tests, including failure paths;
      exact output changes update `docs/specs/reporter-package.md`

## Docs housekeeping

<!-- Delete rows that don't apply (no docs touched). -->

- [ ] `AGENTS.md`, `docs/design/overview.md`, `docs/specs/*` updated for
      behavior or convention changes
- [ ] New docs started from their category's `TEMPLATE.md` and indexed in
      `docs/README.md`
- [ ] New significant decision recorded as an ADR *first*, in this PR
- [ ] Conformance aliases (ADR-0001) untouched — canonical targets edited
      instead

## Protected boundaries

<!-- Delete if the PR touches none of these. -->

- [ ] Changes under `.github/workflows/**`, `.goreleaser.yaml`, `.svu.yaml`,
      or `policies/agent-governance.json` are called out above as high risk
      (new actions SHA-pinned, least-privilege permissions)

## Verification

<!-- Paste evidence the gates ran locally. -->

- [ ] `node scripts/check-docs.mjs` green
- [ ] Checked against the
      [PR review rubric](https://github.com/frostyard/std/blob/main/docs/specs/pr-review-rubric.md)
