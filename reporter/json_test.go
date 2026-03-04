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
