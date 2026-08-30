package reporter

import (
	"fmt"
	"io"
)

// TextReporter writes human-readable progress text to an io.Writer.
type TextReporter struct {
	w       io.Writer
	stepped bool // true after the first Step call
	failed  bool // true after the first writer error; latches every later call to a no-op
}

// NewTextReporter returns a TextReporter that writes to w. A nil writer —
// either a literal nil or an io.Writer holding a nil concrete value — is
// treated as io.Discard so reporting remains silent and non-panicking.
func NewTextReporter(w io.Writer) *TextReporter {
	return &TextReporter{w: discardIfNil(w)}
}

// write sends s to the underlying writer in a single call, unless a previous
// write already failed. On the first writer error — or a short write, which
// io.Writer's contract treats as a failure — it silently latches failure so
// no later call can write to a partial or failed stream, matching the
// package's existing no-error-return convention.
func (r *TextReporter) write(s string) {
	if r.failed {
		return
	}
	n, err := io.WriteString(r.w, s)
	if err != nil || n < len(s) {
		r.failed = true
	}
}

func (r *TextReporter) Step(step, total int, name string) {
	if r.failed {
		return
	}
	prefix := ""
	if r.stepped {
		prefix = "\n"
	}
	r.stepped = true
	r.write(fmt.Sprintf("%sStep %d/%d: %s...\n", prefix, step, total, name))
}

func (r *TextReporter) Progress(_ int, message string) {
	if r.failed || message == "" {
		return
	}
	r.write(fmt.Sprintf("  %s\n", message))
}

func (r *TextReporter) Message(format string, args ...any) {
	if r.failed {
		return
	}
	r.write(fmt.Sprintf("  %s\n", fmt.Sprintf(format, args...)))
}

func (r *TextReporter) MessagePlain(format string, args ...any) {
	if r.failed {
		return
	}
	r.write(fmt.Sprintf("%s\n", fmt.Sprintf(format, args...)))
}

func (r *TextReporter) Warning(format string, args ...any) {
	if r.failed {
		return
	}
	r.write(fmt.Sprintf("Warning: %s\n", fmt.Sprintf(format, args...)))
}

func (r *TextReporter) Error(err error, message string) {
	if r.failed {
		return
	}
	if err != nil {
		r.write(fmt.Sprintf("Error: %s: %v\n", message, err))
	} else {
		r.write(fmt.Sprintf("Error: %s\n", message))
	}
}

func (r *TextReporter) Complete(message string, _ any) {
	if r.failed {
		return
	}
	const sep = "================================================================="
	r.write(fmt.Sprintf("\n%s\n%s\n%s\n", sep, message, sep))
}

func (r *TextReporter) IsJSON() bool { return false }
