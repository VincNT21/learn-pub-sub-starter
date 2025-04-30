package pubsub

import (
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
