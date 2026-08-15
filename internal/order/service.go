package order

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ModstDev/trading_platform/internal/database"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Service struct {
	db      *sql.DB
	queries *database.Queries
}

func NewService(db *sql.DB, queries *database.Queries) *Service {
	return &Service{
		queries: queries,
		db:      db,
	}
}

type Side string

const (
	SideBuy  Side = "BUY"
	SideSell Side = "SELL"
)

type Type string

const (
	TypeMarket Type = "MARKET"
	TypeLimit  Type = "LIMIT"
)

type Status string

const (
	StatusPending   Status = "PENDING"
	StatusExecuted  Status = "EXECUTED"
	StatusCancelled Status = "CANCELLED"
	StatusRejected  Status = "REJECTED"
)

type CreateInput struct {
	AccountID    uuid.UUID
	InstrumentID uuid.UUID
	Side         Side
	Type         Type
	Quantity     *decimal.Decimal
	Price        *decimal.Decimal
}

func (s *Service) Create(ctx context.Context, input CreateInput) (*database.Order, error) {
	if input.Type != TypeLimit {
		return nil, errors.New("only limit orders are currently supported")
	}

	if input.Side != SideBuy {
		return nil, errors.New("only buy orders are currently supported")
	}

	if input.Price == nil {
		return nil, errors.New("price is required for limit orders")
	}

	if input.Quantity.LessThanOrEqual(decimal.Zero) {
		return nil, errors.New("quantity must be greater than zero")
	}

	if input.Price.LessThanOrEqual(decimal.Zero) {
		return nil, errors.New("price must be greater than zero")
	}

	requiredFunds := input.Quantity.Mul(*input.Price)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	queries := database.New(tx)

	result, err := queries.ReserveFunds(ctx, database.ReserveFundsParams{
		ReservedBalance: requiredFunds.String(),
		ID:              input.AccountID.String(),
		Balance:         requiredFunds.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("reserving funds: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("checking reserved funds: %w", err)
	}

	if rowsAffected == 0 {
		return nil, errors.New("insufficient available funds")
	}

	orderID := uuid.New()

	var price sql.NullString

	if input.Price != nil {
		price = sql.NullString{
			String: input.Price.String(),
			Valid:  true,
		}
	}

	err = queries.CreateOrder(ctx, database.CreateOrderParams{
		ID:           orderID.String(),
		AccountID:    input.AccountID.String(),
		InstrumentID: input.InstrumentID.String(),
		Side:         string(input.Side),
		Type:         string(input.Type),
		Quantity:     input.Quantity.String(),
		Price:        price,
		Status:       string(StatusPending),
	})

	if err != nil {
		return nil, fmt.Errorf("creating order: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("committing transaction: %w", err)
	}

	order, err := s.queries.GetOrderByID(ctx, orderID.String())
	if err != nil {
		return nil, fmt.Errorf("getting created order: %w", err)
	}

	return &order, nil
}

func (s *Service) ListByAccountID(ctx context.Context, accountID uuid.UUID) ([]database.Order, error) {
	orders, err := s.queries.ListOrdersByAccountID(ctx, accountID.String())
	if err != nil {
		return nil, fmt.Errorf("listing orders: %w", err)
	}

	return orders, nil
}
