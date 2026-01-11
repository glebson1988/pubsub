package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	url := "amqp://guest:guest@localhost:5672/"
	conn, err := amqp.Dial(url)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatal(err)
	}
	defer ch.Close()
	username, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatal(err)
	}

	gs := gamelogic.NewGameState(username)
	pauseQueueName := routing.PauseKey + "." + username
	moveQueueName := routing.ArmyMovesPrefix + "." + username
	warQueueName := routing.WarRecognitionsPrefix
	armyMovesRoutingKey := "army_moves.*"
	warRoutingKey := routing.WarRecognitionsPrefix + ".*"
	if strings.HasSuffix(routing.WarRecognitionsPrefix, ".") {
		warRoutingKey = routing.WarRecognitionsPrefix + "*"
	}

	err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilDirect,
		pauseQueueName,
		routing.PauseKey,
		pubsub.SimpleQueueTransient,
		handlerPause(gs),
	)

	if err != nil {
		log.Printf("could not subscribe to pause state: %v", err)
	}

	err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilTopic,
		moveQueueName,
		armyMovesRoutingKey,
		pubsub.SimpleQueueTransient,
		handlerMove(gs, ch, username),
	)

	if err != nil {
		log.Printf("could not subscribe to pause move: %v", err)
	}

	err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilTopic,
		warQueueName,
		warRoutingKey,
		pubsub.SimpleQueueDurable,
		handlerWar(gs, ch),
	)

	if err != nil {
		log.Printf("could not subscribe to war recognitions: %v", err)
	}

	fmt.Println("Starting Peril client... (Ctrl+C to exit)")

	for {
		words := gamelogic.GetInput()
		if len(words) == 0 {
			continue
		}
		switch words[0] {
		case "spawn":
			err := gs.CommandSpawn(words)
			if err != nil {
				log.Printf("Could not spawning: %v", err)
			}
		case "move":
			move, err := gs.CommandMove(words)
			if err != nil {
				log.Printf("Could not move: %v", err)
				break
			}

			routingKey := routing.ArmyMovesPrefix + "." + username

			err = pubsub.PublishJSON(
				ch,
				routing.ExchangePerilTopic,
				routingKey,
				move,
			)
			if err != nil {
				log.Printf("Could not publish move: %v", err)
			} else {
				log.Printf("published move to %s", routingKey)
			}
		case "status":
			gs.CommandStatus()
		case "help":
			gamelogic.PrintClientHelp()
		case "spam":
			if len(words) < 2 {
				log.Printf("usage: spam <n>")
				break
			}
			count, err := strconv.Atoi(words[1])
			if err != nil {
				log.Printf("invalid spam count: %v", err)
				break
			}
			routingKey := routing.GameLogSlug + "." + username
			for i := 0; i < count; i++ {
				message := gamelogic.GetMaliciousLog()
				logEntry := routing.GameLog{
					CurrentTime: time.Now(),
					Message:     message,
					Username:    username,
				}
				err := pubsub.PublishGob(
					ch,
					routing.ExchangePerilTopic,
					routingKey,
					logEntry,
				)
				if err != nil {
					log.Printf("could not publish log: %v", err)
					break
				}
			}
		case "quit":
			gamelogic.PrintQuit()
			return
		default:
			log.Printf("unknown command: %s", words[0])
		}
	}
}
