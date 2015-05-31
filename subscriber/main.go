package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/andrewwebber/cqrs"
	"github.com/couchbaselabs/go-couchbase"

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
	if err := bus.ReceiveCommands(cqrs.CommandReceiverOptions{commandTypeCache, closeChannel, errorChannel, receiveCommandChannel, false}); err != nil {
		failOnError(err, "Receive")
	}

	for {
		// Wait on multiple channels using the select control flow.
		select {
		// Version event received channel receives a result with a channel to respond to, signifying successful processing of the message.
		// This should eventually call an event handler. See cqrs.NewVersionedEventDispatcher()
		case command := <-receiveCommandChannel:
			sampleCommand := command.Command.Body.(scaleout.SampleCommand)
			err := updateValueInCouchbaseV2(sampleCommand.Message)
			if err == nil {
				command.ProcessedSuccessfully <- true
			} else {
				log.Println(err)
				command.ProcessedSuccessfully <- false
			}

			// Receiving on this channel signifys an error has occured work processor side
		case err := <-errorChannel:
			failOnError(err, "Error")
		}
	}
}

type DataObject struct {
	Message string
	Count   int
}

func updateValueInCouchbase(message string) error {
	client, err := couchbase.Connect("http://localhost:8091/")
	if err != nil {
		return err
	}

	pool, err := client.GetPool("default")
	if err != nil {
		return err
	}

	bucket, err := pool.GetBucket("cqrs")
	if err != nil {
		return err
	}

	var dataObject DataObject
	dataObjectKey := "cqrs-scaleout:dataobject"
	err = bucket.Get(dataObjectKey, &dataObject)
	if err != nil {
		if !IsNotFoundError(err) {
			return err
		}

		dataObject = DataObject{Message: message, Count: 0}
	}

	dataObject.Count = dataObject.Count + 1
	log.Println(dataObject.Count)
	return bucket.Set(dataObjectKey, 0, &dataObject)
}

func updateValueInCouchbaseV2(message string) error {
	client, err := couchbase.Connect("http://localhost:8091/")
	if err != nil {
		return err
	}

	pool, err := client.GetPool("default")
	if err != nil {
		return err
	}

	bucket, err := pool.GetBucket("cqrs")
	if err != nil {
		return err
	}

	var dataObject DataObject
	dataObjectKey := "cqrs-scaleout:dataobject"
	var cas uint64
	err = bucket.Gets(dataObjectKey, &dataObject, &cas)
	if err != nil {
		if !IsNotFoundError(err) {
			return err
		}

		dataObject = DataObject{Message: message, Count: 0}
	}

	dataObject.Count = dataObject.Count + 1
	log.Println(dataObject.Count)
	log.Println(cas)
	return bucket.Cas(dataObjectKey, 0, cas, &dataObject)
}

func IsNotFoundError(err error) bool {
	// No error?
	if err == nil {
		return false
	}

	return strings.Contains(err.Error(), "Not found")
}
