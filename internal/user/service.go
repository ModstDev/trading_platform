package user

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ModstDev/trading_platform/internal/auth"
	"github.com/ModstDev/trading_platform/internal/database"
)

const minPasswordLength = 8

type Service struct {
	queries *database.Queries
}

type RegisterInput struct {
	Email    string
	Password string
}

func NewService(queries *database.Queries) *Service {
	return &Service{
		queries: queries,
	}
}

func (s *Service) Register(ctx context.Context, input RegisterInput) (*database.User, error) {
	if err := validateEmail(input.Email); err != nil {
		return nil, err
	}

	if err := validatePassword(input.Password); err != nil {
		return nil, err
	}

	passwordHash, err := auth.HashPassword(input.Password)
	if err != nil {
		return nil, fmt.Errorf("hashing password %w", err)
	}

	err = s.queries.CreateUser(ctx, database.CreateUserParams{
		Email:        input.Email,
		PasswordHash: passwordHash,
	})
	if err != nil {
		return nil, fmt.Errorf("creating user: %w", err)
	}

	user, err := s.queries.GetUserByEmail(ctx, input.Email)
	if err != nil {
		return nil, fmt.Errorf("getting created user: %w", err)
	}

	return &database.User{
		ID:        user.ID,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	}, nil
}

func validateEmail(email string) error {
	email = strings.TrimSpace(email)

	if email == "" {
		return errors.New("email is required")
	}

	if !strings.Contains(email, "@") {
		return errors.New("invalid email")
	}

	return nil
}

func validatePassword(password string) error {
	if len(password) < minPasswordLength {
		return errors.New("password must contain at least 8 characters")
	}

	return nil
}
