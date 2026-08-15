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

func (s *Service) Create(ctx context.Context, userID uuid.UUID) (*database.Account, error) {
	accountID := uuid.New()

	err := s.queries.CreateAccount(ctx, database.CreateAccountParams{
		ID:       accountID.String(),
		UserID:   userID.String(),
		Balance:  "0.0000",
		Currency: "EUR",
	})
	if err != nil {
		return nil, fmt.Errorf("creating account: %w", err)
	}

	account, err := s.queries.GetAccountByUserID(ctx, userID.String())
	if err != nil {
		return nil, fmt.Errorf("getting created account: %w", err)
	}

	return &account, nil
}
