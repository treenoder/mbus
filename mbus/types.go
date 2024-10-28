package mbus

import "context"

// HandlerFunc is the function signature for event handlers.
type HandlerFunc func(ctx context.Context, event Event) ([]Event, []Command, error)

// MiddlewareFunc defines the function signature for middleware that wraps handlers.
type MiddlewareFunc func(next HandlerFunc) HandlerFunc

// LoggerFunc is the function signature for logging function
type LoggerFunc func(values ...any)
