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

	// Connect to rabit
	connection, err := amqp.Dial(connectionString)
	if err != nil {
		log.Fatalf("could not connect to RabbitMQ: %v", err)
	}
	defer connection.Close()
	fmt.Println("Peril game client connected to RabbitMQ!")

	// Prompt the user for a username
	username, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatalf("could not get username: %v", err)
	}

	// Declare and bind a transient queue (using pubsub function)
	_, queue, err := pubsub.DelareAndBind(
		connection,
		routing.ExchangePerilDirect,
		routing.PauseKey+"."+username,
		routing.PauseKey,
		0, // transient queue
	)
	if err != nil {
		log.Fatalf("could not suscribe to pause: %v", err)
	}
	fmt.Printf("Queue %v declared and bound!\n", queue.Name)

	// Create a new game state
	gamestate := gamelogic.NewGameState(username)

	// Show client's user available commands
	gamelogic.PrintClientHelp()

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
		// If input is spawn,
		case "spawn":
			err = gamestate.CommandSpawn(inputs)
			if err != nil {
				log.Println(err)
			}
		// If input is move,
		case "move":
			_, err = gamestate.CommandMove(inputs)
			if err != nil {
				log.Println(err)
			}
			fmt.Println("move successful")
		// If input is status,
		case "status":
			gamestate.CommandStatus()
		// If input is help, print all available commands
		case "help":
			gamelogic.PrintClientHelp()
		// If input is spam, print that it's not allowed
		case "spam":
			fmt.Println("Spamming not allowed yet!")
		// If input is quit, exit the REPL client
		case "quit":
			gamelogic.PrintQuit()
			return
		// If input is anything else
		default:
			fmt.Println("unknown command")
			fmt.Println("type 'help' for a list of available commands")
		}
	}
}
