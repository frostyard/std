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
