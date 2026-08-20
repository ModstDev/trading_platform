package outbox

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ModstDev/trading_platform/internal/database"
	"github.com/ModstDev/trading_platform/internal/pubsub"
)

type Service struct {
	queries *database.Queries
	nats    *pubsub.NATS
}

func NewService(queries *database.Queries, nats *pubsub.NATS) *Service {
	return &Service{
		queries: queries,
		nats:    nats,
	}
}

func (s *Service) Process(ctx context.Context) error {
	events, err := s.queries.GetUnpublishedOutboxEvents(ctx, 100)
	if err != nil {
		return fmt.Errorf("getting unpublished events: %w", err)
	}

	for _, event := range events {
		log.Printf("outbox: publishing event %s (%s)", event.ID, event.EventType)

		_, err := s.nats.Publish(ctx, event.Subject, []byte(event.Payload))
		if err != nil {
			log.Printf("outbox: failed o publish event %s: %v", event.ID, err)
			continue
		}

		err = s.queries.MarkOutboxEventPublished(ctx, event.ID)
		if err != nil {
			return fmt.Errorf("marking event %s as published: %w", event.ID, err)
		}

		log.Printf("outbox: event %s published", event.ID)
	}

	return nil
}

func (s *Service) Run(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("outbox: worker stopped")
			return
		case <-ticker.C:
			if err := s.Process(ctx); err != nil {
				log.Printf("outbox: processing failed: %v", err)
			}
		}
	}

}
