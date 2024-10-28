package mbus

import (
	"context"
	"errors"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
)

// mockEvent is a mock implementation of Event
type mockEvent struct {
	eventType EventType
	payload   string
}

func (m *mockEvent) GetType() EventType {
	return m.eventType
}

// mockCommand is a mock implementation of Command
type mockCommand struct {
	name     string
	events   []Event
	execErr  error
	execWait time.Duration // Simulate delay in Execute
}

func (m *mockCommand) GetName() string {
	return m.name
}

func (m *mockCommand) Execute(_ context.Context) ([]Event, error) {
	if m.execWait > 0 {
		time.Sleep(m.execWait)
	}
	return m.events, m.execErr
}

// TestNewBus tests the NewBus function
func TestNewBus(t *testing.T) {
	t.Run("With custom logger", func(t *testing.T) {
		var logged []any
		customLogger := func(values ...any) {
			logged = append(logged, values...)
		}

		bus := NewBus(customLogger)
		require.NotNil(t, bus)
		require.NotNil(t, bus.log)

		// Test default fields
		require.Len(t, bus.handlers, 0)
		require.Len(t, bus.middlewares, 0)

		// Test logging
		bus.log("Test log")
		require.Len(t, logged, 1)
		require.Equal(t, "Test log", logged[0].(string))
	})

	t.Run("With nil logger", func(t *testing.T) {
		bus := NewBus(nil)
		require.NotNil(t, bus.log)

		// Ensure the default logger does not panic when called
		defer func() {
			require.Nil(t, recover())
		}()
		bus.log("Default logger test")
	})
}

// TestRegisterHandler tests the RegisterHandler method
func TestRegisterHandler(t *testing.T) {
	bus := NewBus(nil)
	eventType := EventType("test_event")

	handlerCalled := false
	handler := func(ctx context.Context, event Event) ([]Event, []Command, error) {
		handlerCalled = true
		return nil, nil, nil
	}

	bus.RegisterHandler(eventType, handler)

	// Verify handler registration
	bus.muh.RLock()
	defer bus.muh.RUnlock()
	handlers, exists := bus.handlers[eventType]
	require.True(t, exists, "Handler for event type %s not registered", eventType)
	require.Len(t, handlers, 1, "Expected 1 handler, got %d", len(handlers))

	// Dispatch event to trigger handler
	event := &mockEvent{eventType: eventType, payload: "payload"}
	err := bus.dispatch(context.Background(), event)
	require.NoError(t, err, "Unexpected error during dispatch")
	require.True(t, handlerCalled, "Expected handler to be called")
}

// TestUseMiddleware tests the UseMiddleware method
func TestUseMiddleware(t *testing.T) {
	bus := NewBus(nil)
	eventType := EventType("middleware_event")

	var middlewareOrder []string

	mw1 := func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, event Event) ([]Event, []Command, error) {
			middlewareOrder = append(middlewareOrder, "mw1_before")
			events, cmds, err := next(ctx, event)
			middlewareOrder = append(middlewareOrder, "mw1_after")
			return events, cmds, err
		}
	}

	mw2 := func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, event Event) ([]Event, []Command, error) {
			middlewareOrder = append(middlewareOrder, "mw2_before")
			events, cmds, err := next(ctx, event)
			middlewareOrder = append(middlewareOrder, "mw2_after")
			return events, cmds, err
		}
	}

	bus.UseMiddleware(mw1)
	bus.UseMiddleware(mw2)

	handler := func(ctx context.Context, event Event) ([]Event, []Command, error) {
		middlewareOrder = append(middlewareOrder, "handler")
		return nil, nil, nil
	}

	bus.RegisterHandler(eventType, handler)

	// Dispatch event to trigger handler with middleware
	event := &mockEvent{eventType: eventType, payload: "test"}
	err := bus.dispatch(context.Background(), event)
	require.NoError(t, err, "Unexpected error during dispatch")

	// Verify middleware order
	expectedOrder := []string{
		"mw1_before",
		"mw2_before",
		"handler",
		"mw2_after",
		"mw1_after",
	}
	require.EqualValues(t, expectedOrder, middlewareOrder)
	for i, v := range expectedOrder {
		require.Equal(t, v, middlewareOrder[i], "Expected middlewareOrder[%d] = %s, got %s", i, v, middlewareOrder[i])
	}
}

// TestExecuteCommand tests the ExecuteCommand method
func TestExecuteCommand(t *testing.T) {
	bus := NewBus(nil)
	eventType := EventType("command_event")

	handlerCalled := false
	handler := func(ctx context.Context, event Event) ([]Event, []Command, error) {
		handlerCalled = true
		return nil, nil, nil
	}

	bus.RegisterHandler(eventType, handler)

	// Define a command that emits one event
	cmd := &mockCommand{
		name:   "test_command",
		events: []Event{&mockEvent{eventType: eventType, payload: "cmd_payload"}},
	}

	err := bus.ExecuteCommand(context.Background(), cmd)
	require.NoError(t, err, "Unexpected error during ExecuteCommand")
	require.True(t, handlerCalled, "Expected handler to be called by ExecuteCommand")
}

// TestDispatchNoHandlers tests dispatching an event with no handlers
func TestDispatchNoHandlers(t *testing.T) {
	bus := NewBus(nil)
	eventType := EventType("no_handler_event")
	event := &mockEvent{eventType: eventType, payload: "test"}

	err := bus.dispatch(context.Background(), event)
	require.Error(t, err, "Expected error when dispatching event with no handlers")
	expectedErrMsg := "no handlers registered for event: no_handler_event"
	require.Equal(t, expectedErrMsg, err.Error(), "Expected error message '%s', got '%s'", expectedErrMsg, err.Error())
}

// TestHandlerError tests a handler returning an error
func TestHandlerError(t *testing.T) {
	bus := NewBus(nil)
	eventType := EventType("error_event")

	handler := func(ctx context.Context, event Event) ([]Event, []Command, error) {
		return nil, nil, errors.New("handler error")
	}

	bus.RegisterHandler(eventType, handler)

	event := &mockEvent{eventType: eventType, payload: "test"}

	err := bus.dispatch(context.Background(), event)
	require.Error(t, err, "Expected error from handler")
	require.Equal(t, "handler error", err.Error(), "Expected 'handler error', got '%s'", err.Error())
}

// TestMiddlewareError tests middleware handling when next handler returns an error
func TestMiddlewareError(t *testing.T) {
	bus := NewBus(nil)
	eventType := EventType("middleware_error_event")

	mw := func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, event Event) ([]Event, []Command, error) {
			return next(ctx, event)
		}
	}

	bus.UseMiddleware(mw)

	handler := func(ctx context.Context, event Event) ([]Event, []Command, error) {
		return nil, nil, errors.New("handler error")
	}

	bus.RegisterHandler(eventType, handler)

	event := &mockEvent{eventType: eventType, payload: "test"}

	err := bus.dispatch(context.Background(), event)
	require.Error(t, err, "Expected error from handler through middleware")
	require.Equal(t, "handler error", err.Error(), "Expected 'handler error', got '%s'", err.Error())
}

// TestConcurrentDispatch tests dispatching events concurrently
func TestConcurrentDispatch(t *testing.T) {
	bus := NewBus(nil)
	eventType := EventType("concurrent_event")

	var (
		mu          sync.Mutex
		handlerCall int
	)

	handler := func(ctx context.Context, event Event) ([]Event, []Command, error) {
		mu.Lock()
		handlerCall++
		mu.Unlock()
		return nil, nil, nil
	}

	// Register multiple handlers
	numHandlers := 10
	for i := 0; i < numHandlers; i++ {
		bus.RegisterHandler(eventType, handler)
	}

	// Dispatch the event
	event := &mockEvent{eventType: eventType, payload: "test"}

	err := bus.dispatch(context.Background(), event)
	require.NoError(t, err, "Unexpected error during dispatch")

	// Verify all handlers were called
	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, numHandlers, handlerCall, "Expected %d handler calls, got %d", numHandlers, handlerCall)
}

// TestExecuteCommandWithExecutionError tests ExecuteCommand when command execution returns an error
func TestExecuteCommandWithExecutionError(t *testing.T) {
	bus := NewBus(nil)
	cmd := &mockCommand{
		name:    "failing_command",
		execErr: errors.New("execution failed"),
	}

	err := bus.ExecuteCommand(context.Background(), cmd)
	require.Error(t, err, "Expected error from ExecuteCommand")
	require.Equal(t, "execution failed", err.Error(), "Expected 'execution failed', got '%s'", err.Error())
}

// TestMiddlewareOrdering tests that middleware are applied in the correct order
func TestMiddlewareOrdering(t *testing.T) {
	bus := NewBus(nil)
	eventType := EventType("ordering_event")

	var order []string
	var mu sync.Mutex

	mw1 := func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, event Event) ([]Event, []Command, error) {
			mu.Lock()
			order = append(order, "mw1_before")
			mu.Unlock()

			events, cmds, err := next(ctx, event)

			mu.Lock()
			order = append(order, "mw1_after")
			mu.Unlock()
			return events, cmds, err
		}
	}

	mw2 := func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, event Event) ([]Event, []Command, error) {
			mu.Lock()
			order = append(order, "mw2_before")
			mu.Unlock()

			events, cmds, err := next(ctx, event)

			mu.Lock()
			order = append(order, "mw2_after")
			mu.Unlock()
			return events, cmds, err
		}
	}

	bus.UseMiddleware(mw1)
	bus.UseMiddleware(mw2)

	handler := func(ctx context.Context, event Event) ([]Event, []Command, error) {
		mu.Lock()
		order = append(order, "handler")
		mu.Unlock()
		return nil, nil, nil
	}

	bus.RegisterHandler(eventType, handler)

	// Dispatch event
	event := &mockEvent{eventType: eventType, payload: "test"}
	err := bus.dispatch(context.Background(), event)
	require.NoError(t, err, "Unexpected error during dispatch")

	// Verify order
	expectedOrder := []string{
		"mw1_before",
		"mw2_before",
		"handler",
		"mw2_after",
		"mw1_after",
	}

	mu.Lock()
	defer mu.Unlock()
	require.EqualValues(t, expectedOrder, order)
	for i, v := range expectedOrder {
		require.Equal(t, v, order[i], "Expected order[%d] = %s, got %s", i, v, order[i])
	}
}

// TestExecuteComplexCommand tests processing of a command that emits events and commands
func TestExecuteComplexCommand(t *testing.T) {
	bus := NewBus(nil)
	event1 := &mockEvent{eventType: EventType("event1")}
	event2 := &mockEvent{eventType: EventType("event2")}
	event3 := &mockEvent{eventType: EventType("event3")}

	cmd1 := &mockCommand{
		name:   "command1",
		events: []Event{event1},
	}
	cmd2 := &mockCommand{
		name:   "command2",
		events: []Event{event3},
	}

	handler1 := func(ctx context.Context, event Event) ([]Event, []Command, error) {
		require.Equal(t, event1, event, "Expected event1 to be processed by handler1")
		return []Event{event2}, []Command{cmd2}, nil
	}
	receivedEvents := make(map[Event]struct{})
	handler2 := func(ctx context.Context, event Event) ([]Event, []Command, error) {
		receivedEvents[event] = struct{}{}
		return nil, nil, nil
	}

	bus.RegisterHandler(event1.eventType, handler1)
	bus.RegisterHandler(event2.eventType, handler2)
	bus.RegisterHandler(event3.eventType, handler2)

	err := bus.ExecuteCommand(context.Background(), cmd1)
	require.NoError(t, err, "Unexpected error during ExecuteCommand")
	require.Len(t, receivedEvents, 2, "Expected 2 events to be processed")
	_, ok := receivedEvents[event2]
	require.True(t, ok, "Expected event2 to be processed")
	_, ok = receivedEvents[event3]
	require.True(t, ok, "Expected event3 to be processed")
}

// TestAggregatesErrors tests that ExecuteCommand aggregates errors from all handlers of an event
func TestExecuteCommandAggregatesErrors(t *testing.T) {
	bus := NewBus(nil)
	event1 := &mockEvent{eventType: EventType("event1")}
	event2 := &mockEvent{eventType: EventType("event2")}

	err1 := errors.New("error1")
	err2 := errors.New("error2")
	err3 := errors.New("error3")

	cmd1 := &mockCommand{
		name:   "command1",
		events: []Event{event1},
	}
	cmd2 := &mockCommand{
		name:    "command2",
		execErr: err3,
	}

	handler0 := func(ctx context.Context, event Event) ([]Event, []Command, error) {
		return nil, nil, err1
	}
	handler1 := func(ctx context.Context, event Event) ([]Event, []Command, error) {
		return []Event{event2}, nil, nil
	}
	handler2 := func(ctx context.Context, event Event) ([]Event, []Command, error) {
		return nil, []Command{cmd2}, nil
	}
	handler3 := func(ctx context.Context, event Event) ([]Event, []Command, error) {
		return nil, nil, err2
	}

	bus.RegisterHandler(event1.eventType, handler0)
	bus.RegisterHandler(event1.eventType, handler1)
	bus.RegisterHandler(event1.eventType, handler2)
	bus.RegisterHandler(event1.eventType, handler3)

	err := bus.ExecuteCommand(context.Background(), cmd1)
	require.Error(t, err, "Expected error from ExecuteCommand")
	require.ErrorIs(t, err, err1, "Expected error1")
	require.ErrorIs(t, err, err2, "Expected error2")
	require.ErrorIs(t, err, err3, "Expected error3")
}
