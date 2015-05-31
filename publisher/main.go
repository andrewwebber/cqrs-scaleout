package main

import (
	"fmt"
	"log"
	"reflect"

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
	bus := rabbit.NewEventBus("amqp://guest:guest@localhost:5672/", "scaleout_events", "scaleout_events")

	// Register types
	eventType := reflect.TypeOf(scaleout.SampleEvent{})
	eventTypeCache := cqrs.NewTypeRegistry()
	eventTypeCache.RegisterType(scaleout.SampleEvent{})

	// Publish a simple event to the exchange http://www.rabbitmq.com/tutorials/tutorial-three-python.html
	log.Println("Publishing Commands")

	//for {
	//time.Sleep(1 * time.Second)
	for i := 0; i < 10; i++ {
		if err := bus.PublishEvents([]cqrs.VersionedEvent{cqrs.VersionedEvent{
			Version:   i,
			EventType: eventType.String(),
			Event:     scaleout.SampleEvent{"rabbit_TestEvent"}}}); err != nil {
			failOnError(err, "publish event")
		}
	}
	//}
}
