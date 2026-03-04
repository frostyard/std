package reporter

import (
	"encoding/json"
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

func (r *JSONReporter) emit(event ProgressEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	event.Timestamp = time.Now().UTC().Format(time.RFC3339)
	_ = r.encoder.Encode(event)
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
		Percent: percent,
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
	r.emit(ProgressEvent{
		Type:    EventTypeError,
		Message: message,
		Details: map[string]string{"error": err.Error()},
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
