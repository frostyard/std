// Package reporter provides a progress reporting interface with text,
// JSON Lines, and no-op implementations.
package reporter

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
