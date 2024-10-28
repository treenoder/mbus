# mbus – Event Bus Library

This Go library provides an event bus mechanism to manage event handlers, middleware, and command execution. It supports registering handlers for specific events, applying middleware, and executing commands while dispatching the resulting events to the appropriate handlers.

## Features

- **Event Registration**: Register handlers for specific event types.
- **Middleware Support**: Add middleware to modify or enhance handler behavior.
- **Command Execution**: Execute commands that generate events, which are then dispatched to their respective handlers.
- **Concurrent Event Handling**: Supports concurrent handling of events.
- **Error Handling**: Propagates and aggregates errors from handlers and commands for a single event.

## Table of Contents

- [Installation](#installation)
- [Usage](#usage)
    - [Creating a Bus](#creating-a-bus)
    - [Registering Handlers](#registering-handlers)
    - [Using Middleware](#using-middleware)
    - [Executing Commands](#executing-commands)
- [Testing](#testing)
- [Makefile Commands](#makefile-commands)
- [License](#license)

## Installation

To install `mbus`, you can use the following command:
```bash
go get github.com/treenoder/mbus
```

Import `mbus` into your Go project:
```go
import "github.com/treenoder/mbus/mbus"
```

## Usage

### Creating a Bus

To create a new instance of the event bus, call the `NewBus` function:

```go
logFunc := func(values ...any) {
    fmt.Println(values...)
}
bus := mbus.NewBus(logFunc)
```

You can pass a custom logging function, or if `logFunc` is set to `nil`, the library initializes a default empty logger.

### Registering Handlers

To register a handler for a specific event type:

```go
bus.RegisterHandler("UserRegistered", func(ctx context.Context, event mbus.Event) ([]mbus.Event, []mbus.Command, error) {
    // Handle the event
    return nil, nil, nil
})
```

### Using Middleware

Middleware can be used to wrap event handlers for additional logic before and after the handler executes:

```go
bus.UseMiddleware(func(next mbus.HandlerFunc) mbus.HandlerFunc {
    return func(ctx context.Context, event mbus.Event) ([]mbus.Event, []mbus.Command, error) {
        // Before handler
        events, commands, err := next(ctx, event)
        // After handler
        return events, commands, err
    }
})
```

### Executing Commands

Commands execute logic and can generate events that are then dispatched to their registered handlers:

```go
cmd := &MyCommand{
    // Command details
}
err := bus.ExecuteCommand(context.Background(), cmd)
if err != nil {
    log.Println("Error executing command:", err)
}
```

A command must implement the `Command` interface:

```go
type MyCommand struct {}

func (c *MyCommand) GetName() string {
    return "MyCommand"
}

func (c *MyCommand) Execute(ctx context.Context) ([]mbus.Event, error) {
    // Execute the command logic
    return []mbus.Event{&MyEvent{}}, nil
}
```

### Event Definition

An event should implement the `Event` interface:

```go
type MyEvent struct {}

func (e *MyEvent) GetType() mbus.EventType {
    return "MyEvent"
}
```

## Testing

The library includes comprehensive tests to cover the main functionalities. To run the tests:

```bash
go test -v ./mbus/...
```

## Makefile Commands

The repository includes a `Makefile` to help automate testing and code coverage:

- **Run all tests**:
  ```bash
  make test
  ```

- **Generate and display code coverage**:
  ```bash
  make cover
  ```

- **Clean up coverage files**:
  ```bash
  make clean
  ```

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for more details.
