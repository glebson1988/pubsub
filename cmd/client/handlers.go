package main

import (
	"fmt"
	"time"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func handlerMove(gs *gamelogic.GameState, ch *amqp.Channel, username string) func(gamelogic.ArmyMove) pubsub.AckType {
	return func(move gamelogic.ArmyMove) pubsub.AckType {
		defer fmt.Printf("> ")
		outcome := gs.HandleMove(move)

		switch outcome {
		case gamelogic.MoveOutComeSafe:
			return pubsub.Ack
		case gamelogic.MoveOutcomeMakeWar:
			routingKey := routing.WarRecognitionsPrefix + "." + username
			err := pubsub.PublishJSON(
				ch,
				routing.ExchangePerilTopic,
				routingKey,
				gamelogic.RecognitionOfWar{
					Attacker: move.Player,
					Defender: gs.GetPlayerSnap(),
				},
			)
			if err != nil {
				fmt.Printf("Error publishing war recognition: %v\n", err)
				return pubsub.NackRequeue
			}
			return pubsub.Ack
		case gamelogic.MoveOutcomeSamePlayer:
			return pubsub.NackDiscard
		default:
			return pubsub.NackDiscard
		}
	}
}

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) pubsub.AckType {
	return func(ps routing.PlayingState) pubsub.AckType {
		defer fmt.Print("> ")
		gs.HandlePause(ps)
		return pubsub.Ack
	}
}

func publishGameLog(ch *amqp.Channel, initiator string, log routing.GameLog) pubsub.AckType {
	routingKey := routing.GameLogSlug + "." + initiator
	if err := pubsub.PublishGob(ch, routing.ExchangePerilTopic, routingKey, log); err != nil {
		fmt.Printf("Error publishing game log: %v\n", err)
		return pubsub.NackRequeue
	}
	return pubsub.Ack
}

func handlerWar(gs *gamelogic.GameState, ch *amqp.Channel) func(gamelogic.RecognitionOfWar) pubsub.AckType {
	return func(rw gamelogic.RecognitionOfWar) pubsub.AckType {
		defer fmt.Print("> ")
		outcome, winner, loser := gs.HandleWar(rw)

		switch outcome {
		case gamelogic.WarOutcomeNotInvolved:
			return pubsub.NackRequeue
		case gamelogic.WarOutcomeNoUnits:
			return pubsub.NackDiscard
		case gamelogic.WarOutcomeOpponentWon, gamelogic.WarOutcomeYouWon:
			message := fmt.Sprintf("%s won a war against %s", winner, loser)
			initiator := rw.Attacker.Username
			log := routing.GameLog{
				CurrentTime: time.Now(),
				Message:     message,
				Username:    initiator,
			}
			return publishGameLog(ch, initiator, log)
		case gamelogic.WarOutcomeDraw:
			message := fmt.Sprintf("A war between %s and %s resulted in a draw", winner, loser)
			initiator := rw.Attacker.Username
			log := routing.GameLog{
				CurrentTime: time.Now(),
				Message:     message,
				Username:    initiator,
			}
			return publishGameLog(ch, initiator, log)
		default:
			fmt.Printf("Unknown war outcome: %v\n", outcome)
			return pubsub.NackDiscard
		}
	}
}
