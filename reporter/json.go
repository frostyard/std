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
}

// NewJSONReporter returns a JSONReporter that writes to w.
func NewJSONReporter(w io.Writer) *JSONReporter {
	return &JSONReporter{encoder: json.NewEncoder(w)}
}

// emit stamps the event and writes it as one JSON line. When the event
// cannot be encoded because the caller-supplied Details value is not
// JSON-encodable (a func, a channel, NaN/Inf, or a Marshaler that fails), the
// same event is re-emitted with Details replaced by
// {"encoding_error": "<reason>"} so the line — in particular the terminal
// `complete` event — still reaches the stream as a valid ProgressEvent.
// Writer (I/O) errors are discarded: a progress stream has no channel to
// report its own transport failure and must never abort the caller.
func (r *JSONReporter) emit(event ProgressEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	event.Timestamp = time.Now().UTC().Format(time.RFC3339)
	err := r.encoder.Encode(event)
	if err == nil || !isEncodingError(err) {
		return
	}
	event.Details = map[string]any{"encoding_error": err.Error()}
	_ = r.encoder.Encode(event)
}

// isEncodingError reports whether err is a value-encoding failure from
// encoding/json rather than a writer error. json.Encoder marshals the whole
// value before writing, so an encoding failure leaves nothing on the wire.
func isEncodingError(err error) bool {
	var typeErr *json.UnsupportedTypeError
	var valueErr *json.UnsupportedValueError
	var marshalerErr *json.MarshalerError
	return errors.As(err, &typeErr) || errors.As(err, &valueErr) || errors.As(err, &marshalerErr)
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
