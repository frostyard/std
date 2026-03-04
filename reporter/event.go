package reporter

// EventType represents the type of progress event.
type EventType string

const (
	EventTypeStep     EventType = "step"
	EventTypeProgress EventType = "progress"
	EventTypeMessage  EventType = "message"
	EventTypeWarning  EventType = "warning"
	EventTypeError    EventType = "error"
	EventTypeComplete EventType = "complete"
)

// ProgressEvent represents a single line of JSON Lines output for streaming progress.
type ProgressEvent struct {
	Type       EventType `json:"type"`
	Timestamp  string    `json:"timestamp"`
	Step       int       `json:"step,omitzero"`
	TotalSteps int       `json:"total_steps,omitzero"`
	StepName   string    `json:"step_name,omitempty"`
	Message    string    `json:"message,omitempty"`
	Percent    int       `json:"percent,omitzero"`
	Details    any       `json:"details,omitempty"`
}
