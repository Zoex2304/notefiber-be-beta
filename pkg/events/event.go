package events

import "time"

// Event defines the contract for all system events.
type Event interface {
	// EventType returns the unique code for this event (e.g., "USER_LOGIN").
	EventType() string

	// Payload returns the data associated with the event.
	Payload() map[string]interface{}

	// Timestamp returns when the event occurred.
	Timestamp() time.Time
}

// BaseEvent helps embed common logic if needed,
// strictly creating valid implementations is preferred though.
type BaseEvent struct {
	Type       string
	Data       map[string]interface{}
	OccurredAt time.Time
}

func (e BaseEvent) EventType() string {
	return e.Type
}

func (e BaseEvent) Payload() map[string]interface{} {
	return e.Data
}

func (e BaseEvent) Timestamp() time.Time {
	return e.OccurredAt
}
