package mbus

import (
	"context"
	"errors"
	"reflect"
	"sync"
)

// Bus manages event handlers, middleware, and command execution.
type Bus struct {
	muh         sync.RWMutex
	handlers    map[EventType][]HandlerFunc
	mum         sync.RWMutex
	middlewares []MiddlewareFunc
	log         LoggerFunc
}

// NewBus creates and returns a new Bus instance.
// If logFunc is nil, it initializes a default empty logger.
func NewBus(logFunc LoggerFunc) *Bus {
	if logFunc == nil {
		logFunc = func(values ...any) {}
	}
	return &Bus{
		handlers:    make(map[EventType][]HandlerFunc),
		middlewares: []MiddlewareFunc{},
		log:         logFunc,
	}
}

// RegisterHandler registers a handler for a specific event type.
// eventType is the event type to register the handler for.
func (b *Bus) RegisterHandler(eventType EventType, handler HandlerFunc) *Bus {
	b.muh.Lock()
	defer b.muh.Unlock()

	b.handlers[eventType] = append(b.handlers[eventType], handler)
	b.log("Handler registered for event type:", eventType.String())
	return b
}

// UseMiddleware adds a middleware to the bus to wraps handlers.
// Middleware is applied in the order they are added.
func (b *Bus) UseMiddleware(mw MiddlewareFunc) {
	b.mum.Lock()
	defer b.mum.Unlock()
	b.middlewares = append(b.middlewares, mw)
	b.log("Middleware added:", reflect.TypeOf(mw).String())
}

// ExecuteCommand executes a command.
// It dispatches resulting events to their respective handlers.
func (b *Bus) ExecuteCommand(ctx context.Context, cmd Command) error {
	b.log("Executing command", "command", cmd.GetName())

	events, err := cmd.Execute(ctx)
	if err != nil {
		b.log("Command execution error", "error", err)
		return err
	}

	// Dispatch resulting events
	for _, event := range events {
		if err := b.dispatch(ctx, event); err != nil {
			b.log("Event dispatch error", "event", event.GetType(), "error", err)
			return err
		}
	}

	return nil
}

// dispatch sends an event to all registered handlers asynchronously.
func (b *Bus) dispatch(ctx context.Context, event Event) error {
	b.muh.RLock()
	handlers, exists := b.handlers[event.GetType()]
	b.muh.RUnlock()

	if !exists || len(handlers) == 0 {
		err := errors.New("no handlers registered for event: " + event.GetType().String())
		b.log("No handlers found", "event", event.GetType(), "error", err)
		return err
	}

	b.log("Dispatching event", "event_type", event.GetType(), "handler_count", len(handlers))

	var wg sync.WaitGroup
	handlersCount := len(handlers)
	errChan := make(chan error, handlersCount)
	wg.Add(handlersCount)

	for _, handler := range handlers {
		go func(h HandlerFunc) {
			defer wg.Done()

			// Apply middleware
			wrappedHandler := h
			for i := len(b.middlewares) - 1; i >= 0; i-- {
				wrappedHandler = b.middlewares[i](wrappedHandler)
			}

			newEvents, newCommands, err := wrappedHandler(ctx, event)
			if err != nil {
				b.log("Handler error", "event", event.GetType(), "error", err)
				errChan <- err
				return
			}

			// Dispatch new events
			for _, newEvent := range newEvents {
				if err := b.dispatch(ctx, newEvent); err != nil {
					errChan <- err
					return
				}
			}

			// Execute new commands
			for _, newCmd := range newCommands {
				if err := b.ExecuteCommand(ctx, newCmd); err != nil {
					errChan <- err
					return
				}
			}
		}(handler)
	}

	wg.Wait()
	close(errChan)

	// Aggregate errors
	var aggErr error
	for err := range errChan {
		if aggErr == nil {
			aggErr = err
		} else {
			aggErr = errors.Join(aggErr, err)
		}
	}

	return aggErr
}
