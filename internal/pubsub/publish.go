package pubsub

import (
	"context"
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"
)

func PublishJson[T any](ch *amqp.Channel, exchange, key string, val T) error {
	// Marshal the val to JSON Bytes
	dat, err := json.Marshal(val)
	if err != nil {
		return err
	}

	// Publish the message to the exchanges with the routing key
	return ch.PublishWithContext(
		context.Background(),
		exchange,
		key,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        dat,
		},
	)
}
