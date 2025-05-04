package main

import (
	"fmt"
	"log"
	"strconv"
	"time"

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

	// Create a publish channel
	publishCh, err := connection.Channel()
	if err != nil {
		log.Fatalf("could not create cannel: %v", err)
	}

	// Prompt the user for a username
	username, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatalf("could not get username: %v", err)
	}

	// Create a new game state
	gamestate := gamelogic.NewGameState(username)

	// Subscribe to pause queue
	err = pubsub.SubscribeJSON(
		connection,
		routing.ExchangePerilDirect,
		routing.PauseKey+"."+gamestate.GetUsername(),
		routing.PauseKey,
		pubsub.SimpleQueueTransient,
		handlerPause(gamestate),
	)
	if err != nil {
		log.Fatalf("could not suscribe to pause: %v", err)
	}

	// Subscribe to army_moves queue
	err = pubsub.SubscribeJSON(
		connection,
		routing.ExchangePerilTopic,
		routing.ArmyMovesPrefix+"."+gamestate.GetUsername(),
		routing.ArmyMovesPrefix+".*",
		pubsub.SimpleQueueTransient,
		handlerMove(gamestate, publishCh),
	)
	if err != nil {
		log.Fatalf("could not suscribe to army_moves: %v", err)
	}

	// Subscribe to war queue
	err = pubsub.SubscribeJSON(
		connection,
		routing.ExchangePerilTopic,
		routing.WarRecognitionsPrefix,
		routing.WarRecognitionsPrefix+".*",
		pubsub.SimpleQueueDurable,
		handlerWar(gamestate, publishCh),
	)
	if err != nil {
		log.Fatalf("could not suscribe to war: %v", err)
	}

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
				continue
			}
		// If input is move,
		case "move":
			move, err := gamestate.CommandMove(inputs)
			if err != nil {
				log.Println(err)
				continue
			}
			// Publish the move
			err = pubsub.PublishJson(
				publishCh,
				routing.ExchangePerilTopic,
				routing.ArmyMovesPrefix+"."+gamestate.GetUsername(),
				move,
			)
			if err != nil {
				fmt.Printf("error: %s\n", err)
				continue
			}

			fmt.Printf("Moved %v units to %s\n", len(move.Units), move.ToLocation)
		// If input is status,
		case "status":
			gamestate.CommandStatus()
		// If input is help, print all available commands
		case "help":
			gamelogic.PrintClientHelp()
		// If input is spam, go for fake spamming feature
		case "spam":
			if len(inputs) != 2 {
				fmt.Println("you must add a number after spam command")
				continue
			}
			n, err := strconv.Atoi(inputs[1])
			if err != nil {
				fmt.Println("error with second input type, not a number")
				continue
			}
			for range n {
				maliciousMsg := gamelogic.GetMaliciousLog()
				publishGameLog(
					publishCh,
					username,
					maliciousMsg,
				)
			}
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

func publishGameLog(publishCh *amqp.Channel, username, msg string) error {
	return pubsub.PublishGob(
		publishCh,
		routing.ExchangePerilTopic,
		routing.GameLogSlug+"."+username,
		routing.GameLog{
			CurrentTime: time.Now(),
			Message:     msg,
			Username:    username,
		},
	)
}
