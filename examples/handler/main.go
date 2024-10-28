package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/treenoder/bus/v1/mbus"
)

var h2Error = errors.New("handler2 error")
var h3Error = errors.New("handler3 error")

func main() {
	bus := mbus.NewBus(nil)

	// Register a handler for the event
	bus.RegisterHandler("Event1", handler1)
	bus.RegisterHandler("Event1", handler2)
	bus.RegisterHandler("Event1", handler3)

	myCommand := &MyCommand{Name: "Command1"}
	err := bus.ExecuteCommand(context.Background(), myCommand)
	if err != nil {
		// ExecuteCommand aggregates errors from all handlers
		fmt.Println("Command execution error:", err)
		fmt.Println("Is handler2 error:", errors.Is(err, h2Error))
		fmt.Println("Is handler3 error:", errors.Is(err, h3Error))
	}
}

func handler1(ctx context.Context, event mbus.Event) ([]mbus.Event, []mbus.Command, error) {
	fmt.Println("handler1 called for event", event.GetType())
	return nil, nil, nil
}

func handler2(ctx context.Context, event mbus.Event) ([]mbus.Event, []mbus.Command, error) {
	fmt.Println("handler2 called for event", event.GetType())
	return nil, nil, h2Error
}

func handler3(ctx context.Context, event mbus.Event) ([]mbus.Event, []mbus.Command, error) {
	fmt.Println("handler3 called for event", event.GetType())
	return nil, nil, h3Error
}

type Event1 struct {
}

func (e Event1) GetType() mbus.EventType {
	return "Event1"
}

type MyCommand struct {
	Name string
}

func (c *MyCommand) GetName() string {
	return c.Name
}

// Execute executes simple command with one event
func (c *MyCommand) Execute(ctx context.Context) ([]mbus.Event, error) {
	fmt.Println("Executing command", c.GetName())
	return []mbus.Event{Event1{}}, nil
}
