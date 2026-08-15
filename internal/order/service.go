package order

import (
	"context"
	"database/sql"
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
	Quantity     string
	Price        sql.NullString
}

func (s *Service) Create(ctx context.Context, input CreateInput) (*database.Order, error) {
	orderID := uuid.New()

	err := s.queries.CreateOrder(ctx, database.CreateOrderParams{
		ID:           orderID.String(),
		AccountID:    input.AccountID.String(),
		InstrumentID: input.InstrumentID.String(),
		Side:         string(input.Side),
		Type:         string(input.Type),
		Quantity:     input.Quantity,
		Price:        input.Price,
		Status:       string(StatusPending),
	})

	if err != nil {
		return nil, fmt.Errorf("creating order: %w", err)
	}

	order, err := s.queries.GetOrderByID(ctx, orderID.String())
	if err != nil {
		return nil, fmt.Errorf("getting created order: %w", err)
	}

	return &order, nil
}
