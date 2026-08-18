package pubsub

import (
	"context"
	"fmt"
	"log"

	"github.com/ModstDev/trading_platform/internal/matching"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

func (n *NATS) StartMatchingConsumer(ctx context.Context, matchingService *matching.Service) error {
	_, err := n.conn.QueueSubscribe(
		OrderCreatedSubject,
		"matching",
		func(msg *nats.Msg) {
			orderID, err := uuid.Parse(string(msg.Data))
			if err != nil {
				log.Printf("invalid order ID from NATS: %v", err)
				return
			}

			log.Printf("NATS: matching order %s", orderID)

			if err := matchingService.MatchOrder(ctx, orderID); err != nil {
				log.Printf("NATS: matching order %s failed: %v", orderID, err)
				return
			}

			log.Printf("NATS: order %s matched successfully", orderID)
		},
	)
	if err != nil {
		return fmt.Errorf("subscribing to order created events: %w", err)
	}

	return nil
}
