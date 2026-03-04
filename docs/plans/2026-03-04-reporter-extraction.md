# Reporter Extraction Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Extract the Reporter interface and three implementations from nbc into `github.com/frostyard/std/reporter`.

**Architecture:** Flat package `reporter/` with one file per concern — interface, event types, and each implementation in separate files. Direct port from nbc with package/import adjustments only.

**Tech Stack:** Go 1.26, stdlib only (`encoding/json`, `fmt`, `io`, `strings`, `sync`, `time`)

---

### Task 1: Update go.mod and create reporter directory

**Files:**
- Modify: `go.mod`
- Create: `reporter/` directory

**Step 1: Update go.mod to Go 1.26**

Change `go.mod` to:

```
module github.com/frostyard/std

go 1.26
```

**Step 2: Create reporter directory**

```bash
mkdir -p reporter
```

**Step 3: Commit**

```bash
git add go.mod
git commit -m "chore: bump go version to 1.26"
```

---

### Task 2: Create event types

**Files:**
- Create: `reporter/event.go`

**Step 1: Write event.go**

```go
package reporter

// EventType represents the type of progress event.
type EventType string

const (
	EventTypeStep     EventType = "step"
	EventTypeProgress EventType = "progress"
	EventTypeMessage  EventType = "message"
	EventTypeWarning  EventType = "warning"
	EventTypeError    EventType = "error"
	EventTypeComplete EventType = "complete"
)

// ProgressEvent represents a single line of JSON Lines output for streaming progress.
type ProgressEvent struct {
	Type       EventType `json:"type"`
	Timestamp  string    `json:"timestamp"`
	Step       int       `json:"step,omitzero"`
	TotalSteps int       `json:"total_steps,omitzero"`
	StepName   string    `json:"step_name,omitempty"`
	Message    string    `json:"message,omitempty"`
	Percent    int       `json:"percent,omitzero"`
	Details    any       `json:"details,omitempty"`
}
```

**Step 2: Verify it compiles**

```bash
cd /home/bjk/projects/frostyard/std && go build ./reporter/
```

Expected: no errors

**Step 3: Commit**

```bash
git add reporter/event.go
git commit -m "feat: add ProgressEvent and EventType types"
```

---

### Task 3: Create Reporter interface

**Files:**
- Create: `reporter/reporter.go`

**Step 1: Write reporter.go**

```go
// Package reporter provides a progress reporting interface with text,
// JSON Lines, and no-op implementations.
package reporter

// Reporter is the interface for reporting progress and messages.
// It has three implementations:
//   - TextReporter: human-readable text output
//   - JSONReporter: machine-readable JSON Lines output
//   - NoopReporter: silently discards all output
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

**Step 2: Verify it compiles**

```bash
cd /home/bjk/projects/frostyard/std && go build ./reporter/
```

Expected: no errors

**Step 3: Commit**

```bash
git add reporter/reporter.go
git commit -m "feat: add Reporter interface"
```

---

### Task 4: Implement TextReporter with tests (TDD)

**Files:**
- Create: `reporter/text_test.go`
- Create: `reporter/text.go`

**Step 1: Write the tests in text_test.go**

```go
package reporter

import (
	"bytes"
	"errors"
	"testing"
)

func TestTextReporter_Step(t *testing.T) {
	var buf bytes.Buffer
	r := NewTextReporter(&buf)

	r.Step(1, 3, "Partitioning disk")

	got := buf.String()
	want := "Step 1/3: Partitioning disk...\n"
	if got != want {
		t.Errorf("Step output = %q, want %q", got, want)
	}
}

func TestTextReporter_StepAddsNewlineAfterFirst(t *testing.T) {
	var buf bytes.Buffer
	r := NewTextReporter(&buf)

	r.Step(1, 3, "First step")
	r.Step(2, 3, "Second step")
	r.Step(3, 3, "Third step")

	got := buf.String()
	want := "Step 1/3: First step...\n\nStep 2/3: Second step...\n\nStep 3/3: Third step...\n"
	if got != want {
		t.Errorf("Step output = %q, want %q", got, want)
	}
}

func TestTextReporter_Progress(t *testing.T) {
	t.Run("non-empty message", func(t *testing.T) {
		var buf bytes.Buffer
		r := NewTextReporter(&buf)

		r.Progress(50, "Halfway there")

		got := buf.String()
		want := "  Halfway there\n"
		if got != want {
			t.Errorf("Progress output = %q, want %q", got, want)
		}
	})

	t.Run("empty message prints nothing", func(t *testing.T) {
		var buf bytes.Buffer
		r := NewTextReporter(&buf)

		r.Progress(50, "")

		got := buf.String()
		if got != "" {
			t.Errorf("Progress with empty message should produce no output, got %q", got)
		}
	})
}

func TestTextReporter_Message(t *testing.T) {
	var buf bytes.Buffer
	r := NewTextReporter(&buf)

	r.Message("Installing %s version %d", "GRUB", 2)

	got := buf.String()
	want := "  Installing GRUB version 2\n"
	if got != want {
		t.Errorf("Message output = %q, want %q", got, want)
	}
}

func TestTextReporter_MessagePlain(t *testing.T) {
	var buf bytes.Buffer
	r := NewTextReporter(&buf)

	r.MessagePlain("No indentation %s", "here")

	got := buf.String()
	want := "No indentation here\n"
	if got != want {
		t.Errorf("MessagePlain output = %q, want %q", got, want)
	}
}

func TestTextReporter_Warning(t *testing.T) {
	var buf bytes.Buffer
	r := NewTextReporter(&buf)

	r.Warning("disk %s is small", "/dev/sda")

	got := buf.String()
	want := "Warning: disk /dev/sda is small\n"
	if got != want {
		t.Errorf("Warning output = %q, want %q", got, want)
	}
}

func TestTextReporter_Error(t *testing.T) {
	var buf bytes.Buffer
	r := NewTextReporter(&buf)

	r.Error(errors.New("permission denied"), "failed to write")

	got := buf.String()
	want := "Error: failed to write: permission denied\n"
	if got != want {
		t.Errorf("Error output = %q, want %q", got, want)
	}
}

func TestTextReporter_Complete(t *testing.T) {
	var buf bytes.Buffer
	r := NewTextReporter(&buf)

	r.Complete("Installation complete!", nil)

	got := buf.String()
	sep := "================================================================="
	want := "\n" + sep + "\n" + "Installation complete!" + "\n" + sep + "\n"
	if got != want {
		t.Errorf("Complete output = %q, want %q", got, want)
	}
}

func TestTextReporter_IsJSON(t *testing.T) {
	var buf bytes.Buffer
	r := NewTextReporter(&buf)

	if r.IsJSON() {
		t.Error("TextReporter.IsJSON() = true, want false")
	}
}
```

**Step 2: Run tests to verify they fail**

```bash
cd /home/bjk/projects/frostyard/std && go test ./reporter/ -run TestTextReporter -v
```

Expected: FAIL — `NewTextReporter` not defined

**Step 3: Write text.go**

```go
package reporter

import (
	"fmt"
	"io"
)

// TextReporter writes human-readable progress text to an io.Writer.
type TextReporter struct {
	w       io.Writer
	stepped bool // true after the first Step call
}

// NewTextReporter returns a TextReporter that writes to w.
func NewTextReporter(w io.Writer) *TextReporter {
	return &TextReporter{w: w}
}

func (r *TextReporter) Step(step, total int, name string) {
	if r.stepped {
		_, _ = fmt.Fprintln(r.w)
	}
	r.stepped = true
	_, _ = fmt.Fprintf(r.w, "Step %d/%d: %s...\n", step, total, name)
}

func (r *TextReporter) Progress(_ int, message string) {
	if message != "" {
		_, _ = fmt.Fprintf(r.w, "  %s\n", message)
	}
}

func (r *TextReporter) Message(format string, args ...any) {
	_, _ = fmt.Fprintf(r.w, "  %s\n", fmt.Sprintf(format, args...))
}

func (r *TextReporter) MessagePlain(format string, args ...any) {
	_, _ = fmt.Fprintln(r.w, fmt.Sprintf(format, args...))
}

func (r *TextReporter) Warning(format string, args ...any) {
	_, _ = fmt.Fprintf(r.w, "Warning: %s\n", fmt.Sprintf(format, args...))
}

func (r *TextReporter) Error(err error, message string) {
	_, _ = fmt.Fprintf(r.w, "Error: %s: %v\n", message, err)
}

func (r *TextReporter) Complete(message string, _ any) {
	_, _ = fmt.Fprintln(r.w)
	_, _ = fmt.Fprintln(r.w, "=================================================================")
	_, _ = fmt.Fprintln(r.w, message)
	_, _ = fmt.Fprintln(r.w, "=================================================================")
}

func (r *TextReporter) IsJSON() bool { return false }
```

**Step 4: Run tests to verify they pass**

```bash
cd /home/bjk/projects/frostyard/std && go test ./reporter/ -run TestTextReporter -v
```

Expected: all PASS

**Step 5: Commit**

```bash
git add reporter/text.go reporter/text_test.go
git commit -m "feat: add TextReporter implementation with tests"
```

---

### Task 5: Implement JSONReporter with tests (TDD)

**Files:**
- Create: `reporter/json_test.go`
- Create: `reporter/json.go`

**Step 1: Write the tests in json_test.go**

```go
package reporter

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestJSONReporter_Step(t *testing.T) {
	var buf bytes.Buffer
	r := NewJSONReporter(&buf)

	r.Step(2, 5, "Formatting partitions")

	var event ProgressEvent
	if err := json.Unmarshal(buf.Bytes(), &event); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	if event.Type != EventTypeStep {
		t.Errorf("event.Type = %q, want %q", event.Type, EventTypeStep)
	}
	if event.Step != 2 {
		t.Errorf("event.Step = %d, want 2", event.Step)
	}
	if event.TotalSteps != 5 {
		t.Errorf("event.TotalSteps = %d, want 5", event.TotalSteps)
	}
	if event.StepName != "Formatting partitions" {
		t.Errorf("event.StepName = %q, want %q", event.StepName, "Formatting partitions")
	}
	if event.Timestamp == "" {
		t.Error("event.Timestamp should not be empty")
	}
}

func TestJSONReporter_Message(t *testing.T) {
	var buf bytes.Buffer
	r := NewJSONReporter(&buf)

	r.Message("hello %s", "world")

	var event ProgressEvent
	if err := json.Unmarshal(buf.Bytes(), &event); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	if event.Type != EventTypeMessage {
		t.Errorf("event.Type = %q, want %q", event.Type, EventTypeMessage)
	}
	if event.Message != "hello world" {
		t.Errorf("event.Message = %q, want %q", event.Message, "hello world")
	}
}

func TestJSONReporter_Warning(t *testing.T) {
	var buf bytes.Buffer
	r := NewJSONReporter(&buf)

	r.Warning("low disk space on %s", "/dev/sda")

	var event ProgressEvent
	if err := json.Unmarshal(buf.Bytes(), &event); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	if event.Type != EventTypeWarning {
		t.Errorf("event.Type = %q, want %q", event.Type, EventTypeWarning)
	}
	if event.Message != "low disk space on /dev/sda" {
		t.Errorf("event.Message = %q, want %q", event.Message, "low disk space on /dev/sda")
	}
}

func TestJSONReporter_Progress(t *testing.T) {
	var buf bytes.Buffer
	r := NewJSONReporter(&buf)

	r.Progress(75, "extracting layers")

	var event ProgressEvent
	if err := json.Unmarshal(buf.Bytes(), &event); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	if event.Type != EventTypeProgress {
		t.Errorf("event.Type = %q, want %q", event.Type, EventTypeProgress)
	}
	if event.Percent != 75 {
		t.Errorf("event.Percent = %d, want 75", event.Percent)
	}
	if event.Message != "extracting layers" {
		t.Errorf("event.Message = %q, want %q", event.Message, "extracting layers")
	}
}

func TestJSONReporter_Error(t *testing.T) {
	var buf bytes.Buffer
	r := NewJSONReporter(&buf)

	r.Error(errors.New("disk full"), "write failed")

	var event ProgressEvent
	if err := json.Unmarshal(buf.Bytes(), &event); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	if event.Type != EventTypeError {
		t.Errorf("event.Type = %q, want %q", event.Type, EventTypeError)
	}
	if event.Message != "write failed" {
		t.Errorf("event.Message = %q, want %q", event.Message, "write failed")
	}

	details, ok := event.Details.(map[string]any)
	if !ok {
		t.Fatalf("event.Details is %T, want map[string]any", event.Details)
	}
	if details["error"] != "disk full" {
		t.Errorf("event.Details[error] = %q, want %q", details["error"], "disk full")
	}
}

func TestJSONReporter_Complete(t *testing.T) {
	var buf bytes.Buffer
	r := NewJSONReporter(&buf)

	r.Complete("done", map[string]string{"device": "/dev/sda"})

	var event ProgressEvent
	if err := json.Unmarshal(buf.Bytes(), &event); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	if event.Type != EventTypeComplete {
		t.Errorf("event.Type = %q, want %q", event.Type, EventTypeComplete)
	}
	if event.Message != "done" {
		t.Errorf("event.Message = %q, want %q", event.Message, "done")
	}
}

func TestJSONReporter_MultipleEvents(t *testing.T) {
	var buf bytes.Buffer
	r := NewJSONReporter(&buf)

	r.Step(1, 2, "First")
	r.Message("info")

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 JSON lines, got %d: %q", len(lines), buf.String())
	}

	var event1 ProgressEvent
	if err := json.Unmarshal([]byte(lines[0]), &event1); err != nil {
		t.Fatalf("failed to parse first JSON line: %v", err)
	}
	if event1.Type != EventTypeStep {
		t.Errorf("first event type = %q, want %q", event1.Type, EventTypeStep)
	}

	var event2 ProgressEvent
	if err := json.Unmarshal([]byte(lines[1]), &event2); err != nil {
		t.Fatalf("failed to parse second JSON line: %v", err)
	}
	if event2.Type != EventTypeMessage {
		t.Errorf("second event type = %q, want %q", event2.Type, EventTypeMessage)
	}
}

func TestJSONReporter_IsJSON(t *testing.T) {
	var buf bytes.Buffer
	r := NewJSONReporter(&buf)

	if !r.IsJSON() {
		t.Error("JSONReporter.IsJSON() = false, want true")
	}
}
```

**Step 2: Run tests to verify they fail**

```bash
cd /home/bjk/projects/frostyard/std && go test ./reporter/ -run TestJSONReporter -v
```

Expected: FAIL — `NewJSONReporter` not defined

**Step 3: Write json.go**

```go
package reporter

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"
)

// JSONReporter writes JSON Lines (one ProgressEvent per line) to an
// io.Writer. All writes are serialized with a mutex for thread safety.
type JSONReporter struct {
	mu      sync.Mutex
	encoder *json.Encoder
}

// NewJSONReporter returns a JSONReporter that writes to w.
func NewJSONReporter(w io.Writer) *JSONReporter {
	return &JSONReporter{encoder: json.NewEncoder(w)}
}

func (r *JSONReporter) emit(event ProgressEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	event.Timestamp = time.Now().UTC().Format(time.RFC3339)
	_ = r.encoder.Encode(event)
}

func (r *JSONReporter) Step(step, total int, name string) {
	r.emit(ProgressEvent{
		Type:       EventTypeStep,
		Step:       step,
		TotalSteps: total,
		StepName:   name,
	})
}

func (r *JSONReporter) Progress(percent int, message string) {
	r.emit(ProgressEvent{
		Type:    EventTypeProgress,
		Percent: percent,
		Message: message,
	})
}

func (r *JSONReporter) Message(format string, args ...any) {
	r.emit(ProgressEvent{
		Type:    EventTypeMessage,
		Message: fmt.Sprintf(format, args...),
	})
}

func (r *JSONReporter) MessagePlain(format string, args ...any) {
	r.emit(ProgressEvent{
		Type:    EventTypeMessage,
		Message: fmt.Sprintf(format, args...),
	})
}

func (r *JSONReporter) Warning(format string, args ...any) {
	r.emit(ProgressEvent{
		Type:    EventTypeWarning,
		Message: fmt.Sprintf(format, args...),
	})
}

func (r *JSONReporter) Error(err error, message string) {
	r.emit(ProgressEvent{
		Type:    EventTypeError,
		Message: message,
		Details: map[string]string{"error": err.Error()},
	})
}

func (r *JSONReporter) Complete(message string, details any) {
	r.emit(ProgressEvent{
		Type:    EventTypeComplete,
		Message: message,
		Details: details,
	})
}

func (r *JSONReporter) IsJSON() bool { return true }
```

**Step 4: Run tests to verify they pass**

```bash
cd /home/bjk/projects/frostyard/std && go test ./reporter/ -run TestJSONReporter -v
```

Expected: all PASS

**Step 5: Commit**

```bash
git add reporter/json.go reporter/json_test.go
git commit -m "feat: add JSONReporter implementation with tests"
```

---

### Task 6: Implement NoopReporter with tests (TDD)

**Files:**
- Create: `reporter/noop_test.go`
- Create: `reporter/noop.go`

**Step 1: Write the tests in noop_test.go**

```go
package reporter

import (
	"errors"
	"testing"
)

func TestNoopReporter(t *testing.T) {
	r := NoopReporter{}

	r.Step(1, 3, "test")
	r.Progress(50, "test")
	r.Message("hello %s", "world")
	r.MessagePlain("hello %s", "world")
	r.Warning("careful %s", "now")
	r.Error(errors.New("boom"), "oops")
	r.Complete("done", nil)

	if r.IsJSON() {
		t.Error("NoopReporter.IsJSON() = true, want false")
	}
}
```

**Step 2: Run tests to verify they fail**

```bash
cd /home/bjk/projects/frostyard/std && go test ./reporter/ -run TestNoopReporter -v
```

Expected: FAIL — `NoopReporter` not defined

**Step 3: Write noop.go**

```go
package reporter

// NoopReporter silently discards all output. Useful for tests and contexts
// where no progress reporting is needed.
type NoopReporter struct{}

func (NoopReporter) Step(int, int, string)       {}
func (NoopReporter) Progress(int, string)        {}
func (NoopReporter) Message(string, ...any)      {}
func (NoopReporter) MessagePlain(string, ...any) {}
func (NoopReporter) Warning(string, ...any)      {}
func (NoopReporter) Error(error, string)         {}
func (NoopReporter) Complete(string, any)        {}
func (NoopReporter) IsJSON() bool                { return false }
```

**Step 4: Run tests to verify they pass**

```bash
cd /home/bjk/projects/frostyard/std && go test ./reporter/ -run TestNoopReporter -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add reporter/noop.go reporter/noop_test.go
git commit -m "feat: add NoopReporter implementation with tests"
```

---

### Task 7: Run full test suite and verify

**Step 1: Run all tests**

```bash
cd /home/bjk/projects/frostyard/std && go test ./... -v
```

Expected: all tests PASS

**Step 2: Run go vet**

```bash
cd /home/bjk/projects/frostyard/std && go vet ./...
```

Expected: no issues

**Step 3: Verify interface compliance**

All three types should satisfy `Reporter`. This is implicitly verified by the tests (constructors return concrete types used through the interface), but the compiler will catch any missing methods.
