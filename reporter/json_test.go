package reporter

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"math"
	"strconv"
	"strings"
	"sync"
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
	if event.Percent == nil {
		t.Fatal("event.Percent is nil, want 75")
	}
	if *event.Percent != 75 {
		t.Errorf("event.Percent = %d, want 75", *event.Percent)
	}
	if event.Message != "extracting layers" {
		t.Errorf("event.Message = %q, want %q", event.Message, "extracting layers")
	}
}

func TestJSONReporter_ProgressZeroPercent(t *testing.T) {
	var buf bytes.Buffer
	r := NewJSONReporter(&buf)

	r.Progress(0, "starting")

	var event ProgressEvent
	if err := json.Unmarshal(buf.Bytes(), &event); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	if event.Percent == nil {
		t.Fatal("event.Percent is nil, want 0 — zero percent should be present in JSON output")
	}
	if *event.Percent != 0 {
		t.Errorf("event.Percent = %d, want 0", *event.Percent)
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

func TestJSONReporter_Error_NilErr(t *testing.T) {
	var buf bytes.Buffer
	r := NewJSONReporter(&buf)

	r.Error(nil, "write failed")

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
	if details["error"] != nil {
		t.Errorf("event.Details[error] = %v, want nil", details["error"])
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

func TestJSONReporter_Complete_UnencodableDetails(t *testing.T) {
	cases := []struct {
		name    string
		details any
	}{
		{name: "func", details: func() {}},
		{name: "NaN", details: math.NaN()},
		{name: "channel", details: make(chan int)},
		{name: "failing marshaler", details: failingMarshaler{}},
		{name: "panicking marshaler", details: panickingMarshaler{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			r := NewJSONReporter(&buf)

			r.Complete("done", tc.details)

			lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
			if len(lines) != 1 {
				t.Fatalf("got %d lines, want exactly 1:\n%s", len(lines), buf.String())
			}
			var event ProgressEvent
			if err := json.Unmarshal([]byte(lines[0]), &event); err != nil {
				t.Fatalf("failed to parse JSON output: %v", err)
			}
			if event.Type != EventTypeComplete {
				t.Errorf("event.Type = %q, want %q", event.Type, EventTypeComplete)
			}
			if event.Message != "done" {
				t.Errorf("event.Message = %q, want %q", event.Message, "done")
			}
			if event.Timestamp == "" {
				t.Error("event.Timestamp should not be empty")
			}
			details, ok := event.Details.(map[string]any)
			if !ok {
				t.Fatalf("event.Details = %T, want map[string]any", event.Details)
			}
			reason, ok := details["encoding_error"].(string)
			if !ok || reason == "" {
				t.Errorf("details[\"encoding_error\"] = %#v, want non-empty string", details["encoding_error"])
			}
		})
	}
}

// failingMarshaler is a json.Marshaler whose MarshalJSON always fails, so the
// encoder surfaces a *json.MarshalerError.
type failingMarshaler struct{}

func (failingMarshaler) MarshalJSON() ([]byte, error) {
	return nil, errors.New("marshal refused")
}

// panickingMarshaler is a json.Marshaler whose MarshalJSON always panics, so
// emit must recover it rather than let it crash the caller.
type panickingMarshaler struct{}

func (panickingMarshaler) MarshalJSON() ([]byte, error) {
	panic("marshal panicked")
}

func TestJSONReporter_WriterErrorIsSilent(t *testing.T) {
	r := NewJSONReporter(failingWriter{})

	// Neither a normal event nor an unencodable one may panic or block when
	// the writer itself fails; the reporter has no channel to report it.
	r.Complete("done", map[string]string{"device": "/dev/sda"})
	r.Complete("done", func() {})
}

func TestJSONReporter_WriterErrorStopsLaterEvents(t *testing.T) {
	w := &partialThenSuccessWriter{}
	r := NewJSONReporter(w)

	r.Message("partial")
	// Do not let json.Encoder's own sticky writer error mask a missing reporter guard.
	r.encoder = json.NewEncoder(w)
	r.Complete("must not be appended", nil)

	if !r.failed {
		t.Error("reporter did not latch primary transport failure")
	}
	if w.writes != 1 {
		t.Fatalf("writer called %d times, want 1 after transport failure", w.writes)
	}
	if strings.Contains(w.String(), "must not be appended") {
		t.Errorf("output contains event written after transport failure: %q", w.String())
	}
}

func TestJSONReporter_FallbackWriterErrorStopsLaterEvents(t *testing.T) {
	w := &partialThenSuccessWriter{}
	r := NewJSONReporter(w)

	r.Complete("partial fallback", func() {})
	// Do not let json.Encoder's own sticky writer error mask a missing reporter guard.
	r.encoder = json.NewEncoder(w)
	r.Message("must not be appended")

	if !r.failed {
		t.Error("reporter did not latch fallback transport failure")
	}
	if w.writes != 1 {
		t.Fatalf("writer called %d times, want 1 after fallback transport failure", w.writes)
	}
	if strings.Contains(w.String(), "must not be appended") {
		t.Errorf("output contains event written after fallback transport failure: %q", w.String())
	}
}

func TestJSONReporter_WriterPanicIsSilent(t *testing.T) {
	w := &panickingWriter{}
	r := NewJSONReporter(w)

	// A normal event reaches Write directly during the primary encode.
	r.Complete("done", map[string]string{"device": "/dev/sda"})

	if !r.failed {
		t.Error("reporter did not latch primary writer panic")
	}
	if w.calls != 1 {
		t.Fatalf("writer called %d times, want 1", w.calls)
	}

	r.Message("must not be attempted")
	if w.calls != 1 {
		t.Errorf("writer called %d times after latch, want still 1", w.calls)
	}
}

func TestJSONReporter_FallbackWriterPanicIsSilent(t *testing.T) {
	w := &panickingWriter{}
	r := NewJSONReporter(w)

	// Marshaling an unencodable Details value fails entirely in memory, so
	// the primary encode never calls Write; the panic can only be observed
	// in the fallback encode after Details is replaced.
	r.Complete("done", func() {})

	if !r.failed {
		t.Error("reporter did not latch fallback writer panic")
	}
	if w.calls != 1 {
		t.Fatalf("writer called %d times, want 1", w.calls)
	}
}

// panickingWriter is an io.Writer whose Write always panics, simulating a
// transport failure that surfaces as a panic (for example a closed pipe or
// broken connection) rather than a returned error.
type panickingWriter struct {
	calls int
}

func (w *panickingWriter) Write([]byte) (int, error) {
	w.calls++
	panic("writer panicked")
}

func TestJSONReporter_NilWriterIsSilent(t *testing.T) {
	r := NewJSONReporter(nil)

	r.Message("starting %s", "work")
	r.Progress(50, "halfway")
	r.Complete("done", map[string]string{"result": "ok"})
}

// failingWriter is an io.Writer whose Write always fails.
type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("write refused")
}

type partialThenSuccessWriter struct {
	bytes.Buffer
	writes int
}

func (w *partialThenSuccessWriter) Write(p []byte) (int, error) {
	w.writes++
	if w.writes == 1 {
		n := len(p) / 2
		_, _ = w.Buffer.Write(p[:n])
		return n, errors.New("partial write refused")
	}
	return w.Buffer.Write(p)
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

func TestJSONReporter_ConcurrentMessages(t *testing.T) {
	const messageCount = 64

	var buf bytes.Buffer
	r := NewJSONReporter(&buf)

	var wg sync.WaitGroup
	for i := range messageCount {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.Message("message %d", i)
		}()
	}
	wg.Wait()

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != messageCount {
		t.Fatalf("expected %d JSON lines, got %d: %q", messageCount, len(lines), buf.String())
	}

	messages := make(map[string]bool, messageCount)
	for _, line := range lines {
		var event ProgressEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatalf("failed to parse JSON line %q: %v", line, err)
		}
		if event.Type != EventTypeMessage {
			t.Errorf("event.Type = %q, want %q", event.Type, EventTypeMessage)
		}
		if event.Timestamp == "" {
			t.Error("event.Timestamp should not be empty")
		}
		messages[event.Message] = true
	}

	for i := range messageCount {
		message := "message " + strconv.Itoa(i)
		if !messages[message] {
			t.Errorf("missing message %q", message)
		}
	}
}

func TestJSONReporter_IsJSON(t *testing.T) {
	var buf bytes.Buffer
	r := NewJSONReporter(&buf)

	if !r.IsJSON() {
		t.Error("JSONReporter.IsJSON() = false, want true")
	}
}

func TestJSONReporter_TypedNilWriterIsSilent(t *testing.T) {
	before := typedNilWrites.Load()

	// A nil *nilRecordingWriter stored in an io.Writer is a typed nil: the
	// interface is non-nil (staticcheck's SA4023 rejects comparing it to nil
	// as never true), so the constructor cannot catch it with w == nil.
	var w *nilRecordingWriter
	var iface io.Writer = w

	exerciseReporter(NewJSONReporter(iface))

	if got := typedNilWrites.Load() - before; got != 0 {
		t.Errorf("typed-nil writer received %d writes, want 0", got)
	}
}

func TestJSONReporter_NonNilPointerWriterUnchanged(t *testing.T) {
	w := &nilRecordingWriter{}

	NewJSONReporter(w).Message("hello")

	var event ProgressEvent
	if err := json.Unmarshal(w.buf.Bytes(), &event); err != nil {
		t.Fatalf("unmarshal event from non-nil writer: %v", err)
	}
	if event.Message != "hello" {
		t.Errorf("event.Message = %q, want %q", event.Message, "hello")
	}
}
