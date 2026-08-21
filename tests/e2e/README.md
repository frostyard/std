# End-to-end tests

std is a library and ships no binary, so its end-to-end surface is the set of
runnable programs under [`_examples/`](../../_examples/) (`deploy`,
`fileprocess`, `healthcheck`, `migration`), which collectively exercise every
`reporter.Reporter` method, each run behind a `-format text|json|noop` flag. The
suite in [`examples_test.go`](examples_test.go) builds every example with Go
coverage instrumentation and runs it as a real subprocess:

- `-format json` — exit 0, nothing on stderr, every stdout line decodes into
  `reporter.ProgressEvent` with unknown fields rejected, every `type` is a
  known `EventType`, every `timestamp` is RFC3339, and the last event is
  `complete`;
- `-format text` — exit 0, non-empty output whose first line is not JSON;
- `-format noop` — exit 0, nothing on stderr, no JSON event lines, and
  strictly less output than text mode (the reporter's contribution is gone;
  an example may still print its own human-only tip because `IsJSON()` is
  false);
- `-format bogus` — exit 1 with an "unknown format" message on stderr.

Every execution receives a project-local, test-temporary `GOCOVERDIR`. After
all modes finish, the suite runs `go tool covdata percent` and fails unless
the subprocess counters report covered
`github.com/frostyard/std/reporter` statements. A focused empty-directory
regression test proves that removing the executions or their `GOCOVERDIR`
wiring cannot leave this signal green. This covdata signal is separate from
the in-process `coverage.out` profile and its 95% floor.

`TestExamplesCollectivelyExerciseReporter` pins the collective-coverage claim
above: it reflects the `reporter.Reporter` method set and parses every
`_examples/*/main.go`, failing unless the four programs together call all of
the interface's methods — so a new `Reporter` method fails the suite until an
example exercises it.

`go test ./...` skips underscore directories, so without this suite nothing
compiled the examples. Run it with:

```bash
go test -v ./tests/e2e/...   # the e2e suite alone
make test                    # unit tests + e2e (what CI's Unit Tests job runs)
```

CI runs it inside the `Unit Tests` and `Race Detection` jobs of
[`.github/workflows/ci.yml`](../../.github/workflows/ci.yml). The suite
imports only the standard library and `github.com/frostyard/std/reporter`,
keeping the module stdlib-only. This README is the discoverable e2e entry
point named by
[ADR-0001](../../docs/adr/0001-acmm-conformance-via-canonical-aliases.md).
