package main

import (
	"brainloop/util/cqrs"
	"brainloop/util/cqrs/rabbit"
	"log"
	"reflect"

	scaleout "github.com/andrewwebber/cqrs-scaleout"
)

func main() {
	// Create a new event bus
	bus := rabbit.NewCommandBus("amqp://guest:guest@localhost:5672/", "rabbit_scaleout", "testing.scaleout")

	// Register types
	commandType := reflect.TypeOf(scaleout.SampleCommand{})
	commandTypeCache := cqrs.NewTypeRegistry()
	commandTypeCache.RegisterType(scaleout.SampleCommand{})

	// Publish a simple event to the exchange http://www.rabbitmq.com/tutorials/tutorial-three-python.html
	log.Println("Publishing Commands")

	if err := bus.PublishCommands([]cqrs.Command{cqrs.Command{
		CommandType: commandType.String(),
		Body:        scaleout.SampleCommand{"rabbit_TestCommandBus"}}}); err != nil {
		panic(err)
	}
}
