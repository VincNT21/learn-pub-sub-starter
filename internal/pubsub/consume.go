package pubsub

import (
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Acktype int

type SimpleQueueType int

const (
	SimpleQueueDurable SimpleQueueType = iota
	SimpleQueueTransient
)

func DelareAndBind(
	conn *amqp.Connection,
	exchange, queueName, key string,
	simpleQueueType SimpleQueueType,
) (*amqp.Channel, amqp.Queue, error) {
	// Create a new Channel on the connection
	ch, err := conn.Channel()
	if err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("could not create channel: %v", err)
	}

	// Declare a new queue
	newQueue, err := ch.QueueDeclare(
		queueName,
		simpleQueueType == SimpleQueueDurable, // durable should only be true if simpleQueueType durable
		simpleQueueType != SimpleQueueDurable, // autoDelete should only be true if simpleQueueType is transient
		simpleQueueType != SimpleQueueDurable, // exclusive should only be true if simpleQueueType is transient
		false,                                 // no-wait
		nil,                                   // arguments
	)
	if err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("could not declare queue: %v", err)
	}

	// Bind the queue to the exchange
	err = ch.QueueBind(
		newQueue.Name, // queue name
		key,           // routing key
		exchange,      // exchange
		false,         // no-wait
		nil,           // args
	)
	if err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("could not bind queue: %v", err)
	}

	// Return the channel and queue
	return ch, newQueue, nil
}

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType SimpleQueueType,
	handler func(T),
) error {
	// make sure that the given queue exists and is bound to the exchange
	ch, _, err := DelareAndBind(conn, exchange, queueName, key, simpleQueueType)
	if err != nil {
		return err
	}
	// Get a new chan onf amqp.Delivery structs
	newChan, err := ch.Consume(queueName, "", false, false, false, false, nil)
	if err != nil {
		return err
	}

	// Start a goroutine that ranges over the channel of deliveries
	go func() {
		for delivery := range newChan {
			// Unmarshal the body of each message into the T type
			var data T
			json.Unmarshal(delivery.Body, &data)

			// Call the handler function
			handler(data)

			// Acknowledge the message to remove it from the queue
			delivery.Ack(false)
		}
	}()

	return nil
}
