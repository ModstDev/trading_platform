package user

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"database/sql"

	"github.com/google/uuid"

	"github.com/ModstDev/trading_platform/internal/auth"
	"github.com/ModstDev/trading_platform/internal/database"
)

const minPasswordLength = 8

type Service struct {
	db      *sql.DB
	queries *database.Queries
}

type RegisterInput struct {
	Email    string
	Password string
}

type LoginInput struct {
	Email    string
	Password string
}

func NewService(db *sql.DB, queries *database.Queries) *Service {
	return &Service{
		db:      db,
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

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	queries := database.New(tx)

	userID := uuid.New()

	err = queries.CreateUser(ctx, database.CreateUserParams{
		ID:           userID,
		Email:        input.Email,
		PasswordHash: passwordHash,
	})
	if err != nil {
		return nil, fmt.Errorf("creating user: %w", err)
	}

	err = queries.CreateAccount(ctx, database.CreateAccountParams{
		ID:              uuid.New().String(),
		UserID:          userID.String(),
		Balance:         "1000.0000",
		ReservedBalance: "0.0000",
		Currency:        "EUR",
	})
	if err != nil {
		return nil, fmt.Errorf("creating account: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("committing transaction: %w", err)
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

func (s *Service) Login(
	ctx context.Context,
	input LoginInput,
) (*database.User, error) {
	user, err := s.queries.GetUserByEmail(ctx, input.Email)
	if err != nil {
		return nil, err
	}

	valid, err := auth.CheckPassword(input.Password, user.PasswordHash)
	if err != nil {
		return nil, err
	}
	if !valid {
		return nil, errors.New("invalid credentials")
	}

	return &database.User{
		ID:        user.ID,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	}, nil
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*database.User, error) {
	user, err := s.queries.GetUserByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting user by ID: %w", err)
	}

	return &database.User{
		ID:           user.ID,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		CreatedAt:    user.CreatedAt,
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
