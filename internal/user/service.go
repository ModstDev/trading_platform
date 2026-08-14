package user

import (
	"context"

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

func (s *Service) Create(ctx context.Context, email, passwordHash string) error {
	_, err := s.queries.CreateUser(
		ctx,
		database.CreateUserParams{
			Email:        email,
			PasswordHash: passwordHash,
		},
	)
	return err
}
