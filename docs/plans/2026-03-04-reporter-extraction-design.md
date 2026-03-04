# Reporter Extraction Design

Extract the Reporter interface and implementations from `github.com/frostyard/nbc/pkg` into `github.com/frostyard/std/reporter` as the first shared package in the frostyard standard library.

## Source

- `github.com/frostyard/nbc/pkg/reporter.go` — Reporter interface, TextReporter, JSONReporter, NoopReporter
- `github.com/frostyard/nbc/pkg/types/types.go` — ProgressEvent, EventType

## Package

`github.com/frostyard/std/reporter` (Go 1.26, no external dependencies)

## File Layout

```
reporter/
├── reporter.go      # Reporter interface
├── event.go         # ProgressEvent struct, EventType constants
├── text.go          # TextReporter (human-readable output to io.Writer)
├── json.go          # JSONReporter (JSON Lines output, thread-safe with mutex)
├── noop.go          # NoopReporter (silent no-op)
├── text_test.go     # TextReporter tests
├── json_test.go     # JSONReporter tests
└── noop_test.go     # NoopReporter tests
```

## Interface

```go
type Reporter interface {
    Step(step, total int, name string)
    Progress(percent int, message string)
    Message(format string, args ...any)
    MessagePlain(format string, args ...any)
    Warning(format string, args ...any)
    Error(err error, message string)
    Complete(message string, details any)
    IsJSON() bool
}
```

## Key Decisions

- **ProgressEvent and EventType move into the reporter package** — they're tightly coupled to JSONReporter and don't warrant a separate subpackage.
- **Direct port** — implementations are copied as-is with only package/import changes. No redesign.
- **Tests ported and split** — nbc's reporter_test.go split into three files by implementation.
- **Scope limited to std** — nbc will be updated to import from std in a separate change.

## Dependencies

Standard library only: `encoding/json`, `fmt`, `io`, `strings`, `sync`, `time`.
