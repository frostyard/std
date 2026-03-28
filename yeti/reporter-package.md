# Reporter Package — Detailed Reference

## Overview

Package `reporter` (`github.com/frostyard/std/reporter`) defines a `Reporter` interface for progress reporting and provides three implementations. It is designed for CLI tools and services that need to communicate step-by-step progress to users or downstream systems.

## Reporter Interface

Defined in `reporter/reporter.go`:

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

### Method semantics

| Method | Purpose |
|--------|---------|
| `Step` | Announce a discrete step in a multi-step process (e.g., "Step 2/5: Building") |
| `Progress` | Report percentage progress within the current step |
| `Message` | Formatted informational message (indented in text mode) |
| `MessagePlain` | Formatted message without indentation |
| `Warning` | Formatted warning message |
| `Error` | Report an error with context message |
| `Complete` | Signal completion with a summary message and optional structured `details` |
| `IsJSON` | Runtime discriminator — `true` only for JSONReporter |

## ProgressEvent (JSON serialization type)

Defined in `reporter/event.go`. This is the struct that `JSONReporter` encodes as JSON Lines:

```go
type ProgressEvent struct {
    Type       EventType `json:"type"`
    Timestamp  string    `json:"timestamp"`
    Step       int       `json:"step,omitzero"`
    TotalSteps int       `json:"total_steps,omitzero"`
    StepName   string    `json:"step_name,omitempty"`
    Message    string    `json:"message,omitempty"`
    Percent    *int      `json:"percent,omitempty"`
    Details    any       `json:"details,omitempty"`
}
```

- `Timestamp` is set at emit time in RFC3339 format (UTC)
- `omitzero` suppresses zero-value int fields; `omitempty` suppresses empty strings and nil pointers
- `Percent` is `*int` so that `nil` means "not reported" (omitted from JSON) while `0` means "zero percent" (included)
- `Details` is `any` — callers can pass structs, maps, or nil

### EventType constants

```go
EventTypeStep     = "step"
EventTypeProgress = "progress"
EventTypeMessage  = "message"
EventTypeWarning  = "warning"
EventTypeError    = "error"
EventTypeComplete = "complete"
```

## Implementations

### TextReporter (`reporter/text.go`)

Human-readable formatted output to an `io.Writer`.

- **Constructor:** `NewTextReporter(w io.Writer) *TextReporter`
- **Thread-safe:** No
- **IsJSON():** `false`

**Output formatting:**
- `Step`: `"Step 1/3: name...\n"` — blank line inserted before steps after the first
- `Progress`: prints non-empty messages only (ignores percent)
- `Message`: `"  formatted message\n"` (two-space indent)
- `MessagePlain`: `"formatted message\n"` (no indent)
- `Warning`: `"Warning: formatted message\n"`
- `Error`: `"Error: message: err.Error()\n"` (if `err` is nil: `"Error: message\n"`)
- `Complete`: message surrounded by 65-char `=` separator lines

**Internal state:** `stepped bool` tracks whether `Step()` has been called, to insert blank lines between steps for readability.

### JSONReporter (`reporter/json.go`)

JSON Lines output (one `ProgressEvent` per line) to an `io.Writer`.

- **Constructor:** `NewJSONReporter(w io.Writer) *JSONReporter`
- **Thread-safe:** Yes — `sync.Mutex` protects all `emit()` calls
- **IsJSON():** `true`

**Key behaviors:**
- Every method constructs a `ProgressEvent` and calls the private `emit()` method
- `emit()` sets the `Timestamp` field to `time.Now().UTC().Format(time.RFC3339)` and encodes via `json.Encoder`
- `Message` and `MessagePlain` produce identical output (both use `EventTypeMessage`)
- `Error` stores error details as `map[string]string{"error": err.Error()}` (nil errors stored as `"<nil>"`)
- The mutex lock covers both timestamp generation and encoding

### NoopReporter (`reporter/noop.go`)

Silent discard — all methods are empty no-ops.

- **Constructor:** None needed — use `NoopReporter{}` (zero-value struct)
- **Thread-safe:** Yes (no state)
- **IsJSON():** `false`

Useful for tests, benchmarks, or contexts where progress output should be suppressed.

## Adding a new Reporter implementation

1. Create a new file `reporter/<name>.go` with a struct implementing all `Reporter` interface methods
2. Create `reporter/<name>_test.go` with tests using `bytes.Buffer` and standard `testing` package
3. Document thread-safety characteristics
4. Keep stdlib-only — no external dependencies
