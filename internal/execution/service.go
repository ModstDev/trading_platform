package execution

import (
	"context"
	"fmt"

	"github.com/ModstDev/trading_platform/internal/database"
	"github.com/google/uuid"
)

type Service struct {
	queries *database.Queries
}

func NewService(queries *database.Queries) *Service {
	return &Service{
		queries: queries,
	}
}

func (s *Service) ListByAccountID(ctx context.Context, accountID uuid.UUID) ([]database.Execution, error) {
	executions, err := s.queries.ListExecutionsByAccountID(ctx, accountID.String())
	if err != nil {
		return nil, fmt.Errorf("listing executions: %w", err)
	}

	return executions, nil
}
