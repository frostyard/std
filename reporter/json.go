package reporter

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"
)

// JSONReporter writes JSON Lines (one ProgressEvent per line) to an
// io.Writer. All writes are serialized with a mutex for thread safety.
type JSONReporter struct {
	mu      sync.Mutex
	encoder *json.Encoder
	failed  bool
}

// NewJSONReporter returns a JSONReporter that writes to w. A nil writer is
// treated as io.Discard so reporting remains silent and non-panicking.
func NewJSONReporter(w io.Writer) *JSONReporter {
	if w == nil {
		w = io.Discard
	}
	return &JSONReporter{encoder: json.NewEncoder(w)}
}

// emit stamps the event and writes it as one JSON line. When the event
// cannot be encoded because the caller-supplied Details value is not
// JSON-encodable (a func, a channel, NaN/Inf, or a Marshaler that fails or
// panics), the same event is re-emitted with Details replaced by
// {"encoding_error": "<reason>"} so the line — in particular the terminal
// `complete` event — still reaches the stream as a valid ProgressEvent.
// Writer (I/O) errors are discarded: a progress stream has no channel to
// report its own transport failure and must never abort the caller. After the
// first writer error, later events are dropped so they cannot follow a partial
// JSON record on the same stream.
func (r *JSONReporter) emit(event ProgressEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.failed {
		return
	}
	event.Timestamp = time.Now().UTC().Format(time.RFC3339)
	err := r.safeEncode(event)
	if err == nil {
		return
	}
	if !isEncodingError(err) {
		r.failed = true
		return
	}
	event.Details = map[string]any{"encoding_error": err.Error()}
	// The replacement details and fixed event fields are always encodable, so
	// any fallback error is a writer failure.
	if err := r.safeEncode(event); err != nil {
		r.failed = true
	}
}

// safeEncode encodes event, recovering a panic raised while marshaling it —
// for example from a caller-supplied Details value's MarshalJSON panicking
// instead of returning an error — so a hostile or buggy Details value cannot
// crash the caller. A recovered panic is reported as an encodingPanicError so
// isEncodingError routes it through the same fallback path as a Marshaler
// that returns an error.
func (r *JSONReporter) safeEncode(event ProgressEvent) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = encodingPanicError{value: p}
		}
	}()
	return r.encoder.Encode(event)
}

// encodingPanicError wraps a panic recovered while encoding a caller-supplied
// value.
type encodingPanicError struct {
	value any
}

func (e encodingPanicError) Error() string {
	return fmt.Sprintf("panic encoding progress event: %v", e.value)
}

// isEncodingError reports whether err is a value-encoding failure from
// encoding/json rather than a writer error. json.Encoder marshals the whole
// value before writing, so an encoding failure leaves nothing on the wire.
func isEncodingError(err error) bool {
	var typeErr *json.UnsupportedTypeError
	var valueErr *json.UnsupportedValueError
	var marshalerErr *json.MarshalerError
	var panicErr encodingPanicError
	return errors.As(err, &typeErr) || errors.As(err, &valueErr) || errors.As(err, &marshalerErr) || errors.As(err, &panicErr)
}

func (r *JSONReporter) Step(step, total int, name string) {
	r.emit(ProgressEvent{
		Type:       EventTypeStep,
		Step:       step,
		TotalSteps: total,
		StepName:   name,
	})
}

func (r *JSONReporter) Progress(percent int, message string) {
	r.emit(ProgressEvent{
		Type:    EventTypeProgress,
		Percent: &percent,
		Message: message,
	})
}

func (r *JSONReporter) Message(format string, args ...any) {
	r.emit(ProgressEvent{
		Type:    EventTypeMessage,
		Message: fmt.Sprintf(format, args...),
	})
}

func (r *JSONReporter) MessagePlain(format string, args ...any) {
	r.emit(ProgressEvent{
		Type:    EventTypeMessage,
		Message: fmt.Sprintf(format, args...),
	})
}

func (r *JSONReporter) Warning(format string, args ...any) {
	r.emit(ProgressEvent{
		Type:    EventTypeWarning,
		Message: fmt.Sprintf(format, args...),
	})
}

func (r *JSONReporter) Error(err error, message string) {
	var errVal any
	if err != nil {
		errVal = err.Error()
	}
	r.emit(ProgressEvent{
		Type:    EventTypeError,
		Message: message,
		Details: map[string]any{"error": errVal},
	})
}

func (r *JSONReporter) Complete(message string, details any) {
	r.emit(ProgressEvent{
		Type:    EventTypeComplete,
		Message: message,
		Details: details,
	})
}

func (r *JSONReporter) IsJSON() bool { return true }
