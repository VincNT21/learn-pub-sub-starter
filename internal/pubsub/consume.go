package pubsub

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Acktype int

type SimpleQueueType int

const (
	SimpleQueueDurable SimpleQueueType = iota
	SimpleQueueTransient
)

const (
	Ack Acktype = iota
	NackRequeue
	NackDiscard
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
		amqp.Table{ // arguments
			"x-dead-letter-exchange": "peril_dlx",
		},
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
	handler func(T) Acktype,
) error {

	return subscribe(
		conn,
		exchange,
		queueName,
		key,
		simpleQueueType,
		handler,
		func(data []byte) (T, error) {
			var target T
			err := json.Unmarshal(data, target)
			return target, err
		},
	)
}

func SubscribeGob[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType SimpleQueueType,
	handler func(T) Acktype,
) error {
	return subscribe(
		conn,
		exchange,
		queueName,
		key,
		simpleQueueType,
		handler,
		func(data []byte) (T, error) {
			buffer := bytes.NewBuffer(data)
			decoder := gob.NewDecoder(buffer)
			var target T
			err := decoder.Decode(&target)
			return target, err
		},
	)
}

func subscribe[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType SimpleQueueType,
	handler func(T) Acktype,
	unmarshaller func([]byte) (T, error),
) error {
	// make sure that the given queue exists and is bound to the exchange
	ch, _, err := DelareAndBind(conn, exchange, queueName, key, simpleQueueType)
	if err != nil {
		return err
	}

	// Set the prefetch count
	err = ch.Qos(10, 0, false)

	// Get a new chan onf amqp.Delivery structs
	newChan, err := ch.Consume(
		queueName, // queue
		"",        // consumer
		false,     // auto-ack
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // args
	)
	if err != nil {
		return fmt.Errorf("could not consume messages: %v", err)
	}

	// Start a goroutine that ranges over the channel of deliveries
	go func() {
		for msg := range newChan {
			// Unmarshal the body of each message into the T type
			data, err := unmarshaller(msg.Body)
			if err != nil {
				log.Printf("error with unmarshal message: %v\n", err)
			}

			// Call the handler function
			acktype := handler(data)

			// Acknowledge the message depending of acktype
			switch acktype {
			case Ack:
				log.Println("Message Ack")
				msg.Ack(false)
			case NackRequeue:
				log.Println("Message NackRequeue")
				msg.Nack(false, true)
			case NackDiscard:
				log.Println("Message NackDiscard")
				msg.Nack(false, false)
			}
		}
	}()

	return nil
}
