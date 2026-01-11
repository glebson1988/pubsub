package pubsub

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type AckType int
type SimpleQueueType int

const deadLetterExchangeName = "peril_dlx"

const (
	SimpleQueueDurable SimpleQueueType = iota
	SimpleQueueTransient
)

const (
	Ack AckType = iota
	NackRequeue
	NackDiscard
)

func DeclareAndBind(
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
) (*amqp.Channel, amqp.Queue, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("could not create channel: %v", err)
	}

	queue, err := ch.QueueDeclare(
		queueName,
		queueType == SimpleQueueDurable,
		queueType != SimpleQueueDurable,
		queueType != SimpleQueueDurable,
		false,
		amqp.Table{
			"x-dead-letter-exchange": deadLetterExchangeName,
		},
	)
	if err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("could not declare queue: %v", err)
	}

	err = ch.QueueBind(
		queue.Name,
		key,
		exchange,
		false,
		nil,
	)
	if err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("could not bind queue: %v", err)
	}

	return ch, queue, nil
}

func subscribe[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
	handler func(T) AckType,
	unmarshaller func([]byte) (T, error),
) error {
	ch, q, err := DeclareAndBind(
		conn,
		exchange,
		queueName,
		key,
		queueType,
	)
	if err != nil {
		return fmt.Errorf("could not declare and bind queue: %w", err)
	}

	msgs, err := ch.Consume(
		q.Name,
		"",
		false,
		false,
		false,
		false,
		nil,
	)

	if err != nil {
		return fmt.Errorf("could not start consuming messages: %w", err)
	}

	go func() {
		for msg := range msgs {
			target, err := unmarshaller(msg.Body)
			if err != nil {
				fmt.Printf("Error unmarshalling message: %v\n", err)
				if err := msg.Nack(false, false); err != nil {
					fmt.Printf("Error nacking message: %v\n", err)
				}
				continue
			}

			ackType := handler(target)

			switch ackType {
			case Ack:
				fmt.Println("Ack: message processed successfully")
				if err := msg.Ack(false); err != nil {
					fmt.Printf("Error acknowledging message: %v\n", err)
				}
			case NackRequeue:
				fmt.Println("NackRequeue: message will be requeued")
				if err := msg.Nack(false, true); err != nil {
					fmt.Printf("Error nacking (requeue) message: %v\n", err)
				}
			case NackDiscard:
				fmt.Println("NackDiscard: message will be discarded")
				if err := msg.Nack(false, false); err != nil {
					fmt.Printf("Error nacking (discard) message: %v\n", err)
				}
			}
		}
	}()

	return nil
}

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
	handler func(T) AckType,
) error {
	return subscribe(conn, exchange, queueName, key, queueType, handler, func(body []byte) (T, error) {
		var target T
		if err := json.Unmarshal(body, &target); err != nil {
			return target, err
		}
		return target, nil
	})
}

func SubscribeGob[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
	handler func(T) AckType,
) error {
	return subscribe(conn, exchange, queueName, key, queueType, handler, func(body []byte) (T, error) {
		var target T
		dec := gob.NewDecoder(bytes.NewReader(body))
		if err := dec.Decode(&target); err != nil {
			return target, err
		}
		return target, nil
	})
}
