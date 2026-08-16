package position

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/ModstDev/trading_platform/internal/database"
)

type Service struct {
	queries *database.Queries
}

func NewService(queries *database.Queries) *Service {
	return &Service{
		queries: queries,
	}
}

func (s *Service) ListByAccountID(ctx context.Context, accountID uuid.UUID) ([]database.Position, error) {
	positions, err := s.queries.ListPositionsByAccountID(ctx, accountID.String())
	if err != nil {
		return nil, fmt.Errorf("listing positions: %w", err)
	}

	return positions, nil
}
