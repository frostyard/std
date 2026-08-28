// Package reporter provides a progress reporting interface with text,
// JSON Lines, and no-op implementations.
package reporter

import (
	"io"
	"reflect"
)

// Reporter is the interface for reporting progress and messages.
// It has three implementations:
//   - TextReporter: human-readable text output
//   - JSONReporter: machine-readable JSON Lines output
//   - NoopReporter: silently discards all output
//
// Message, MessagePlain, and Warning take fmt.Sprintf-style format strings.
// go vet's printf analyzer checks those calls only through the concrete
// types (*TextReporter, *JSONReporter); calls made through the Reporter
// interface are not statically checked, so a wrong verb or a missing
// argument surfaces at runtime as %!v(...) or %!s(MISSING) in the output.
// Keep format strings literal and covered by tests, or call through the
// concrete type where a test can pin the output.
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

// discardIfNil normalizes a writer a caller cannot have meant to write to
// into io.Discard, so both reporter constructors share one contract.
//
// Two distinct values mean "no writer". A literal nil io.Writer is nil at
// the interface level and is caught by the plain comparison. An io.Writer
// holding a nil concrete value — a typed nil such as (*bytes.Buffer)(nil) or
// a nil map/slice/func/chan-based writer — is *not* equal to nil, so the
// comparison alone lets it through and the first report panics inside the
// concrete Write method. Both arise from the same caller mistake (a
// zero-valued writer variable passed straight to the constructor), so both
// are normalized here and reporting stays silent and non-panicking.
//
// Only the kinds reflect.Value.IsNil accepts are inspected; any other kind
// (a struct value implementing io.Writer, for example) cannot be nil and is
// returned unchanged.
func discardIfNil(w io.Writer) io.Writer {
	if w == nil {
		return io.Discard
	}
	switch v := reflect.ValueOf(w); v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Pointer, reflect.Slice:
		if v.IsNil() {
			return io.Discard
		}
	}
	return w
}
