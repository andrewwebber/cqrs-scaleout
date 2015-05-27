package main

import (
	"brainloop/util/cqrs"
	"brainloop/util/cqrs/rabbit"
	"log"

	"github.com/andrewwebber/cqrs-scaleout"
)

func main() {
	// Create a new event bus
	bus := rabbit.NewCommandBus("amqp://guest:guest@localhost:5672/", "rabbit_scaleout", "testing.scaleout")

	// Register types
	commandTypeCache := cqrs.NewTypeRegistry()
	commandTypeCache.RegisterType(scaleout.SampleCommand{})

	// Create communication channels
	//
	// for closing the queue listener,
	closeChannel := make(chan chan error)
	// receiving errors from the listener thread (go routine)
	errorChannel := make(chan error)
	// and receiving commands from the queue
	receiveCommandChannel := make(chan cqrs.CommandTransactedAccept)
	// Start receiving events by passing these channels to the worker thread (go routine)
	if err := bus.ReceiveCommands(cqrs.CommandReceiverOptions{commandTypeCache, closeChannel, errorChannel, receiveCommandChannel}); err != nil {
		log.Fatal(err)
	}

	// Wait on multiple channels using the select control flow.
	for {
		select {
		case command := <-receiveCommandChannel:
			sampleCommand := command.Command.Body.(scaleout.SampleCommand)
			log.Println(sampleCommand.Message)
			command.ProcessedSuccessfully <- true
			// Receiving on this channel signifys an error has occured work processor side
		case err := <-errorChannel:
			panic(err)
		}
	}
}
