package account

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

func (s *Service) GetByUserID(ctx context.Context, userID uuid.UUID) (*database.Account, error) {
	account, err := s.queries.GetAccountByUserID(ctx, userID.String())
	if err != nil {
		return nil, fmt.Errorf("getting account: %w", err)
	}

	return &database.Account{
		ID:              account.ID,
		UserID:          account.UserID,
		Balance:         account.Balance,
		ReservedBalance: account.ReservedBalance,
		CreatedAt:       account.CreatedAt,
	}, nil
}
