package matching

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ModstDev/trading_platform/internal/database"
	"github.com/ModstDev/trading_platform/internal/marketdata"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Service struct {
	db         *sql.DB
	priceStore *marketdata.PriceStore
}

func NewService(db *sql.DB, priceStore *marketdata.PriceStore) *Service {
	return &Service{
		db:         db,
		priceStore: priceStore,
	}
}

func (s *Service) MatchOrder(
	ctx context.Context,
	orderID uuid.UUID,
) error {
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

	if order.Type == "LIMIT" && !order.Price.Valid {
		return errors.New("limit order has no price")
	}

	orderQuantity, err := decimal.NewFromString(order.Quantity)
	if err != nil {
		return fmt.Errorf("parsing order quantity: %w", err)
	}

	orderFilled, err := decimal.NewFromString(order.FilledQuantity)
	if err != nil {
		return fmt.Errorf("parsing filled quantity: %w", err)
	}

	remaining := orderQuantity.Sub(orderFilled)

	if remaining.LessThanOrEqual(decimal.Zero) {
		return nil
	}

	if order.Type == "MARKET" {
		if err := s.executeMarketOrder(
			ctx,
			queries,
			order,
			remaining,
		); err != nil {
			return fmt.Errorf("executing market order: %w", err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("committing market order: %w", err)
		}

		return nil
	}

	for remaining.GreaterThan(decimal.Zero) {
		var match database.Order

		// Only LIMIT orders reach the matching loop.
		switch order.Side {
		case "BUY":
			match, err = queries.FindMatchingSellOrder(
				ctx,
				database.FindMatchingSellOrderParams{
					InstrumentID: order.InstrumentID,
					Price:        order.Price,
					ID:           order.ID,
				},
			)

		case "SELL":
			match, err = queries.FindMatchingBuyOrder(
				ctx,
				database.FindMatchingBuyOrderParams{
					InstrumentID: order.InstrumentID,
					Price:        order.Price,
					ID:           order.ID,
				},
			)

		default:
			return errors.New("invalid order side")
		}

		if errors.Is(err, sql.ErrNoRows) {
			break
		}

		if err != nil {
			return fmt.Errorf("finding matching order: %w", err)
		}

		matchQuantity, err := decimal.NewFromString(match.Quantity)
		if err != nil {
			return fmt.Errorf("parsing matching quantity: %w", err)
		}

		matchFilled, err := decimal.NewFromString(match.FilledQuantity)
		if err != nil {
			return fmt.Errorf("parsing matching filled quantity: %w", err)
		}

		matchRemaining := matchQuantity.Sub(matchFilled)

		if matchRemaining.LessThanOrEqual(decimal.Zero) {
			continue
		}

		executionQuantity := decimal.Min(
			remaining,
			matchRemaining,
		)

		executionPrice, err := decimal.NewFromString(match.Price.String)
		if err != nil {
			return fmt.Errorf("parsing execution price: %w", err)
		}
		var buy database.Order
		var sell database.Order

		if order.Side == "BUY" {
			buy = order
			sell = match
		} else {
			buy = match
			sell = order
		}

		// Execute buyer side.
		if err := s.executeBuy(
			ctx,
			queries,
			buy,
			executionQuantity,
			executionPrice,
		); err != nil {
			return fmt.Errorf("executing buy: %w", err)
		}

		// Execute seller side.
		if err := s.executeSell(
			ctx,
			queries,
			sell,
			executionQuantity,
			executionPrice,
		); err != nil {
			return fmt.Errorf("executing sell: %w", err)
		}

		// Update original order.
		result, err := queries.UpdateFilledQuantity(
			ctx,
			database.UpdateFilledQuantityParams{
				FilledQuantity:   executionQuantity.String(),
				FilledQuantity_2: executionQuantity.String(),
				ID:               order.ID,
				Quantity:         executionQuantity.String(),
			},
		)
		if err != nil {
			return fmt.Errorf("updating order fill: %w", err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("checking order fill: %w", err)
		}

		if rowsAffected != 1 {
			return errors.New("failed to update order fill")
		}

		// Update matching order.
		result, err = queries.UpdateFilledQuantity(
			ctx,
			database.UpdateFilledQuantityParams{
				FilledQuantity:   executionQuantity.String(),
				FilledQuantity_2: executionQuantity.String(),
				ID:               match.ID,
				Quantity:         executionQuantity.String(),
			},
		)
		if err != nil {
			return fmt.Errorf("updating matching order fill: %w", err)
		}

		rowsAffected, err = result.RowsAffected()
		if err != nil {
			return fmt.Errorf("checking order fill: %w", err)
		}

		if rowsAffected != 1 {
			return errors.New("failed to update order fill")
		}

		// Record buyer execution.
		err = queries.CreateExecution(
			ctx,
			database.CreateExecutionParams{
				ID:           uuid.New().String(),
				OrderID:      buy.ID,
				AccountID:    buy.AccountID,
				InstrumentID: buy.InstrumentID,
				Quantity:     executionQuantity.String(),
				Price:        executionPrice.String(),
			},
		)
		if err != nil {
			return fmt.Errorf("creating buyer execution: %w", err)
		}

		// Record seller execution.
		err = queries.CreateExecution(
			ctx,
			database.CreateExecutionParams{
				ID:           uuid.New().String(),
				OrderID:      sell.ID,
				AccountID:    sell.AccountID,
				InstrumentID: sell.InstrumentID,
				Quantity:     executionQuantity.String(),
				Price:        executionPrice.String(),
			},
		)
		if err != nil {
			return fmt.Errorf("creating seller execution: %w", err)
		}

		remaining = remaining.Sub(executionQuantity)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing match: %w", err)
	}

	return nil
}
