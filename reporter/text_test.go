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
