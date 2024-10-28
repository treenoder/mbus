package main

import (
	"context"
	"fmt"
	"github.com/treenoder/bus/v1/mbus"
)

func main() {
	bus := mbus.NewBus(nil)
	myCommand := &MyCommand{Name: "Command1"}
	err := bus.ExecuteCommand(context.Background(), myCommand)
	if err != nil {
		panic(err)
	}
}

type MyCommand struct {
	Name string
}

func (c *MyCommand) GetName() string {
	return c.Name
}

// Execute executes simple command with no events.
func (c *MyCommand) Execute(ctx context.Context) ([]mbus.Event, error) {
	fmt.Println("Executing command", c.GetName())
	return nil, nil
}
