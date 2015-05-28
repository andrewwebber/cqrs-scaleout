package main

import (
	"fmt"
	"log"

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
	if err := bus.ReceiveCommands(cqrs.CommandReceiverOptions{commandTypeCache, closeChannel, errorChannel, receiveCommandChannel, true}); err != nil {
		failOnError(err, "Receive")
	}

	for {
		// Wait on multiple channels using the select control flow.
		select {
		// Version event received channel receives a result with a channel to respond to, signifying successful processing of the message.
		// This should eventually call an event handler. See cqrs.NewVersionedEventDispatcher()
		case command := <-receiveCommandChannel:
			sampleCommand := command.Command.Body.(scaleout.SampleCommand)
			log.Println(sampleCommand.Message)
			command.ProcessedSuccessfully <- true
			// Receiving on this channel signifys an error has occured work processor side
		case err := <-errorChannel:
			failOnError(err, "Error")
		}
	}
}
