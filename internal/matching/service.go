package matching

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ModstDev/trading_platform/internal/database"
	"github.com/google/uuid"
)

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{
		db: db,
	}
}

func (s *Service) MatchOrder(ctx context.Context, orderID uuid.UUID) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	queries := database.New(tx)

	order, err := queries.GetOrderByID(ctx, orderID.String())
	if err != nil {
		return fmt.Errorf("getting order: %w", err)
	}

	if order.Status != "PENDING" {
		return nil
	}

	if !order.Price.Valid {
		return errors.New("order has no price")
	}

	var match database.Order

	if order.Side == "BUY" {
		match, err = queries.FindMatchingSellOrder(ctx, database.FindMatchingSellOrderParams{
			InstrumentID: order.InstrumentID,
			Price:        order.Price,
		})
	} else {
		match, err = queries.FindMatchingBuyOrder(ctx, database.FindMatchingBuyOrderParams{
			InstrumentID: order.InstrumentID,
			Price:        order.Price,
		})
	}
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Nothing matches yet.
			return tx.Commit()
		}

		return fmt.Errorf("finding matching order: %w", err)
	}
	//TODO:
	_ = match

	return nil
}
