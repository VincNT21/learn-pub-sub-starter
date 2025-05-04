package pubsub

import (
	"bytes"
	"context"
	"encoding/gob"
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

func PublishGob[T any](ch *amqp.Channel, exchange, key string, val T) error {
	// Encode the val to Bytes using gob
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	err := encoder.Encode(val)
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
			ContentType: "application/gob",
			Body:        buffer.Bytes(),
		},
	)
}
