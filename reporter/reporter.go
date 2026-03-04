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
