package mbus

import "context"

// Command represents a generic command in the system.
type Command interface {
	GetName() string
	Execute(ctx context.Context) ([]Event, error)
}
