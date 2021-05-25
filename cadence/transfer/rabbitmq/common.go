package rabbitmq

import (
	"context"
	"encoding/json"
	"log"

	"github.com/streadway/amqp"
)

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

type AmqpConnection struct {
	queue   amqp.Queue
	channel *amqp.Channel
}

func GetConnection() AmqpConnection {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/avenue")
	failOnError(err, "Failed to connect to RabbitMQ")

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")

	q, err := ch.QueueDeclare(
		"hello", // name
		false,   // durable
		false,   // delete when unused
		false,   // exclusive
		false,   // no-wait
		nil,     // arguments
	)
	failOnError(err, "Failed to declare a queue")

	return AmqpConnection{
		channel: ch,
		queue:   q,
	}
}

func (a AmqpConnection) Produce(ctx context.Context, message interface{}) error {

	msg, err := json.Marshal(message)
	failOnError(err, "Faailed to marshal a message")

	// The message content is a byte array, so you can encode whatever you like there.
	err = a.channel.Publish(
		"",           // exchange
		a.queue.Name, // routing key
		false,        // mandatory
		false,        // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        msg,
		})
	log.Printf(" [x] Sent %s", message)
	failOnError(err, "Faailed to publish a message")

	return nil
}

func (a AmqpConnection) GetChannel() *amqp.Channel {
	return a.channel
}

func (a AmqpConnection) GetQueue() string {
	return a.queue.Name
}
