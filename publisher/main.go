package main

import (
	"fmt"
	"log"
	"reflect"
	"time"

	"github.com/andrewwebber/cqrs"

	"github.com/andrewwebber/cqrs/rabbit"

	"github.com/andrewwebber/cqrs-scaleout"
)

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
		panic(fmt.Sprintf("%s: %s", msg, err))
	}
}

func main() {
	// Create a new event bus
	bus := rabbit.NewCommandBus("amqp://guest:guest@localhost:5672/", "scaleout4", "scaleout4")

	// Register types
	commandType := reflect.TypeOf(scaleout.SampleCommand{})
	commandTypeCache := cqrs.NewTypeRegistry()
	commandTypeCache.RegisterType(scaleout.SampleCommand{})

	// Publish a simple event to the exchange http://www.rabbitmq.com/tutorials/tutorial-three-python.html
	log.Println("Publishing Commands")

	for {
		time.Sleep(1 * time.Second)
		for i := 0; i < 10; i++ {
			if err := bus.PublishCommands([]cqrs.Command{cqrs.Command{
				CommandType: commandType.String(),
				Body:        scaleout.SampleCommand{"rabbit_TestCommandBus"}}}); err != nil {
				failOnError(err, "publish commands")
			}
		}
	}
}
