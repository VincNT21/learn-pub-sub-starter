package main

import (
	"fmt"
	"log"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	const connectionString = "amqp://guest:guest@localhost:5672/"

	connection, err := amqp.Dial(connectionString)
	if err != nil {
		log.Fatalf("could not connect to RabbitMQ: %v", err)
	}
	defer connection.Close()
	fmt.Println("Peril game server connected to RabbitMQ!")

	// Create a channel on the connection
	publishCh, err := connection.Channel()
	if err != nil {
		log.Fatalf("could not create connection channel: %v", err)
	}

	// Show server's user available commands
	gamelogic.PrintServerHelp()

	// Start the REPL loop
	for {
		// Get inputs from the user
		inputs := gamelogic.GetInput()

		// If no input, continue to next iteration
		if len(inputs) == 0 {
			continue
		}

		// Check for inputs first word
		switch inputs[0] {
		// If input is pause, pause the game
		case "pause":
			log.Println("Publishing paused game state")
			// Publish a pause message
			err = pubsub.PublishJson(
				publishCh,
				routing.ExchangePerilDirect,
				routing.PauseKey,
				routing.PlayingState{
					IsPaused: true,
				},
			)
			if err != nil {
				log.Printf("could not publish time: %v", err)
			}
		// If input is resume, resume the game
		case "resume":
			log.Println("Publishing resumes game state")
			// Publish a resume message
			err = pubsub.PublishJson(
				publishCh,
				routing.ExchangePerilDirect,
				routing.PauseKey,
				routing.PlayingState{
					IsPaused: false,
				},
			)
			if err != nil {
				log.Printf("could not publish time: %v", err)
			}
		// If input is quit, exit
		case "quit":
			log.Println("goodbye")
			return
		// If input is anything else
		default:
			fmt.Println("unknown command")
			gamelogic.PrintServerHelp()
		}
	}
}
