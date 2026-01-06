package main

import (
	"log"

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

	ch, _, err := pubsub.DeclareAndBind(
		conn,
		routing.ExchangePerilTopic,
		routing.GameLogSlug,
		routing.GameLogSlug+".*",
		pubsub.SimpleQueueDurable,
	)
	if err != nil {
		log.Fatalf("Failed to declare/bind queue: %v", err)
	}
	defer ch.Close()

	gamelogic.PrintServerHelp()

loop:
	for {
		words := gamelogic.GetInput()
		if len(words) == 0 {
			continue
		}
		switch words[0] {
		case "pause":
			log.Printf("sending pause message")
			playingState := routing.PlayingState{IsPaused: true}
			err = pubsub.PublishJSON(
				ch,
				routing.ExchangePerilDirect,
				routing.PauseKey,
				playingState,
			)
			if err != nil {
				log.Printf("could not publish pause state: %v", err)
			}
		case "resume":
			log.Printf("sending resume message")
			playingState := routing.PlayingState{IsPaused: false}
			err = pubsub.PublishJSON(
				ch,
				routing.ExchangePerilDirect,
				routing.PauseKey,
				playingState,
			)
			if err != nil {
				log.Printf("could not publish resume state: %v", err)
			}
		case "quit":
			log.Printf("exiting")
			break loop
		default:
			log.Printf("unknown command: %s", words[0])
		}
	}
}
