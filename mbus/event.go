package mbus

// EventType represents the type of event.
type EventType string

func (e EventType) String() string {
	return string(e)
}

// Event represents a generic event in the system.
type Event interface {
	GetType() EventType
}
