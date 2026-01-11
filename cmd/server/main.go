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
	url := "amqp://guest:guest@localhost:5672/"
	conn, err := amqp.Dial(url)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	publishCh, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to create publish channel: %v", err)
	}
	defer publishCh.Close()

	err = pubsub.SubscribeGob(
		conn,
		routing.ExchangePerilTopic,
		routing.GameLogSlug,
		routing.GameLogSlug+".*",
		pubsub.SimpleQueueDurable,
		func(gameLog routing.GameLog) pubsub.AckType {
			defer fmt.Print("> ")
			if err := gamelogic.WriteLog(gameLog); err != nil {
				log.Printf("could not write log: %v", err)
				return pubsub.NackRequeue
			}
			return pubsub.Ack
		},
	)
	if err != nil {
		log.Fatalf("Failed to subscribe to game logs: %v", err)
	}

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
				publishCh,
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
				publishCh,
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
