package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/streadway/amqp"
)

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
		panic(fmt.Sprintf("%s: %s", msg, err))
	}
}

func main() {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	err = ch.ExchangeDeclare("test_scaleout2", "topic", true, false, false, false, nil)
	failOnError(err, "exchange")

	for {
		time.Sleep(2 * time.Second)
		for i := 0; i < 10; i++ {
			err = ch.Publish(
				"test_scaleout2", // exchange
				"scaleout_queue", // routing key
				false,            // mandatory
				false,
				amqp.Publishing{
					DeliveryMode: amqp.Persistent,
					ContentType:  "text/plain",
					Body:         []byte(fmt.Sprintf("scaleout_queue message %d", i)),
				})
			failOnError(err, "Failed to publish a message")
		}
	}
}

func bodyFrom(args []string) string {
	var s string
	if (len(args) < 2) || os.Args[1] == "" {
		s = "hello"
	} else {
		s = strings.Join(args[1:], " ")
	}
	return s
}
