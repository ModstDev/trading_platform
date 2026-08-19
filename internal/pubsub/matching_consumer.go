package pubsub

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ModstDev/trading_platform/internal/matching"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go/jetstream"
)

func (n *NATS) StartMatchingConsumer(ctx context.Context, matchingService *matching.Service) error {
	consumer, err := n.js.CreateOrUpdateConsumer(ctx, OrderStream, jetstream.ConsumerConfig{
		Name:          MatchingConsumer,
		Durable:       MatchingConsumer,
		AckPolicy:     jetstream.AckExplicitPolicy,
		AckWait:       30 * time.Second,
		FilterSubject: OrderCreatedSubject,
	})
	if err != nil {
		return fmt.Errorf("subscribing to order created events: %w", err)
	}

	log.Printf("NATS: matching consumer started: %s", MatchingConsumer)

	cons, err := consumer.Consume(func(msg jetstream.Msg) {
		orderID, err := uuid.Parse(string(msg.Data()))
		if err != nil {
			log.Printf("NATS: invalid order ID: %v", err)

			// This message can never become valid,
			// so acknowledge it instead of retrying forever
			if err := msg.Ack(); err != nil {
				log.Printf("NATS: failed to ACK invalid message: %v", err)
			}
			return
		}

		log.Printf("NATS: matching order %s", orderID)

		if err := matchingService.MatchOrder(ctx, orderID); err != nil {
			log.Printf("NATS: matching order %s failed: %v", orderID, err)

			// We don't need ACK
			// This is an explicit ACK consumer
			// JetStream can redeliver the message
			return
		}

		if err := msg.Ack(); err != nil {
			log.Printf("NATS: failed to ACK order %s: %v", orderID, err)
			return
		}

		log.Printf("NATS: order %s matching completed", orderID)
	})

	if err != nil {
		return fmt.Errorf("starting matching consumer: %w", err)
	}

	go func() {
		<-ctx.Done()
		cons.Stop()
	}()

	return nil
}
