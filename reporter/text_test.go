package reporter

import (
	"bytes"
	"errors"
	"io"
	"sync/atomic"
	"testing"
)

func TestTextReporter_NilWriter(t *testing.T) {
	r := NewTextReporter(nil)
	if r.w != io.Discard {
		t.Fatalf("NewTextReporter(nil) writer = %T, want io.Discard", r.w)
	}

	tests := []struct {
		name string
		call func()
	}{
		{"first step", func() { r.Step(1, 2, "first") }},
		{"repeated step", func() { r.Step(2, 2, "second") }},
		{"progress", func() { r.Progress(50, "halfway") }},
		{"message", func() { r.Message("item %d", 1) }},
		{"plain message", func() { r.MessagePlain("item %d", 2) }},
		{"warning", func() { r.Warning("item %d", 3) }},
		{"error", func() { r.Error(errors.New("failed"), "item") }},
		{"complete", func() { r.Complete("done", map[string]bool{"ok": true}) }},
		{"repeated complete", func() { r.Complete("still done", nil) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.call()
		})
	}
}

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

func TestTextReporter_Error_NilErr(t *testing.T) {
	var buf bytes.Buffer
	r := NewTextReporter(&buf)

	r.Error(nil, "write failed")

	got := buf.String()
	want := "Error: write failed\n"
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

// typedNilWrites counts Write calls that reached nilRecordingWriter through a
// typed-nil receiver. A constructor that normalized the writer correctly
// never lets such a call happen, so any increment is observable output from a
// writer the caller meant to be absent.
var typedNilWrites atomic.Int64

// nilRecordingWriter is an io.Writer whose Write survives a nil receiver
// instead of panicking, so a test can distinguish "the constructor discarded
// the typed-nil writer" from "the write happened to be harmless".
type nilRecordingWriter struct {
	buf bytes.Buffer
}

func (w *nilRecordingWriter) Write(p []byte) (int, error) {
	if w == nil {
		typedNilWrites.Add(1)
		return len(p), nil
	}
	return w.buf.Write(p)
}

// exerciseReporter calls every Reporter method once.
func exerciseReporter(r Reporter) {
	r.Step(1, 2, "first")
	r.Step(2, 2, "second")
	r.Progress(50, "halfway")
	r.Message("item %d", 1)
	r.MessagePlain("item %d", 2)
	r.Warning("item %d", 3)
	r.Error(errors.New("failed"), "item")
	r.Complete("done", map[string]bool{"ok": true})
}

func TestTextReporter_TypedNilWriter(t *testing.T) {
	before := typedNilWrites.Load()

	// A nil *nilRecordingWriter stored in an io.Writer is a typed nil: the
	// interface is non-nil (staticcheck's SA4023 rejects comparing it to nil
	// as never true), so the constructor cannot catch it with w == nil.
	var w *nilRecordingWriter
	var iface io.Writer = w

	r := NewTextReporter(iface)
	if r.w != io.Discard {
		t.Fatalf("NewTextReporter(typed nil) writer = %T, want io.Discard", r.w)
	}

	exerciseReporter(r)

	if got := typedNilWrites.Load() - before; got != 0 {
		t.Errorf("typed-nil writer received %d writes, want 0", got)
	}
}

func TestTextReporter_NonNilPointerWriterUnchanged(t *testing.T) {
	w := &nilRecordingWriter{}

	r := NewTextReporter(w)
	if r.w != io.Writer(w) {
		t.Fatalf("NewTextReporter(non-nil) writer = %T, want the supplied writer", r.w)
	}

	r.MessagePlain("hello")

	if got, want := w.buf.String(), "hello\n"; got != want {
		t.Errorf("non-nil writer got %q, want %q", got, want)
	}
}

func TestDiscardIfNil_NonNilableKindUnchanged(t *testing.T) {
	// A struct value implementing io.Writer has no nil form; discardIfNil
	// must return it untouched rather than inspect it.
	var w structWriter
	if got := discardIfNil(w); got != io.Writer(w) {
		t.Errorf("discardIfNil(struct writer) = %T, want the supplied writer", got)
	}
}

// structWriter is a non-pointer io.Writer implementation: its reflect.Kind is
// Struct, which reflect.Value.IsNil would panic on.
type structWriter struct{}

func (structWriter) Write(p []byte) (int, error) { return len(p), nil }

// partialFailWriter writes half of the first call's bytes and reports a
// write error, then errors outright (with no bytes written) on any later
// call, so a test can tell whether the reporter invoked it again.
type partialFailWriter struct {
	buf    bytes.Buffer
	writes int
}

func (w *partialFailWriter) Write(p []byte) (int, error) {
	w.writes++
	if w.writes > 1 {
		return 0, errors.New("write called after latch")
	}
	half := len(p) / 2
	n, _ := w.buf.Write(p[:half])
	return n, errors.New("short write")
}

func TestTextReporter_LatchesAfterWriterError(t *testing.T) {
	w := &partialFailWriter{}
	r := NewTextReporter(w)

	r.Message("first message")
	if w.writes != 1 {
		t.Fatalf("writer invoked %d times after first call, want 1", w.writes)
	}
	before := w.buf.String()

	r.Message("second message")
	r.Step(1, 2, "ignored")
	r.Warning("ignored")
	r.Error(errors.New("ignored"), "ignored")
	r.Complete("ignored", nil)

	if w.writes != 1 {
		t.Errorf("writer invoked %d times after latch, want 1", w.writes)
	}
	if got := w.buf.String(); got != before {
		t.Errorf("buffer changed after latch: got %q, want %q", got, before)
	}
}
