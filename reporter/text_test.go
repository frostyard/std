package reporter

import (
	"bytes"
	"errors"
	"io"
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
