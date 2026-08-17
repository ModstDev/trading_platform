package matching

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
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{
		db: db,
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

	order, err := queries.GetOrderByID(
		ctx,
		orderID.String(),
	)
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

	switch order.Side {
	case "BUY":
		match, err = queries.FindMatchingSellOrder(
			ctx,
			database.FindMatchingSellOrderParams{
				InstrumentID: order.InstrumentID,
				Price:        order.Price,
			},
		)

	case "SELL":
		match, err = queries.FindMatchingBuyOrder(
			ctx,
			database.FindMatchingBuyOrderParams{
				InstrumentID: order.InstrumentID,
				Price:        order.Price,
			},
		)

	default:
		return errors.New("invalid order side")
	}

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// No opposite order yet.
			return tx.Commit()
		}

		return fmt.Errorf("finding matching order: %w", err)
	}

	quantity, err := decimal.NewFromString(order.Quantity)
	if err != nil {
		return fmt.Errorf("parsing order quantity: %w", err)
	}

	filledQuantity, err := decimal.NewFromString(order.FilledQuantity)
	if err != nil {
		return fmt.Errorf("parsing filled quantity: %w", err)
	}

	remaining := quantity.Sub(filledQuantity)

	matchQuantity, err := decimal.NewFromString(match.Quantity)
	if err != nil {
		return fmt.Errorf("parsing matching quantity: %w", err)
	}

	matchFilled, err := decimal.NewFromString(match.FilledQuantity)
	if err != nil {
		return fmt.Errorf("parsing matching filled quantity: %w", err)
	}

	matchRemaining := matchQuantity.Sub(matchFilled)

	if remaining.LessThanOrEqual(decimal.Zero) || matchRemaining.LessThanOrEqual(decimal.Zero) {
		return errors.New("no remaining quantity to execute")
	}

	executionQuantity := decimal.Min(remaining, matchRemaining)

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

	// Buyer.
	if err := s.executeBuy(
		ctx,
		queries,
		buy,
		executionQuantity,
		executionPrice,
	); err != nil {
		return err
	}

	// Seller.
	if err := s.executeSell(
		ctx,
		queries,
		sell,
		executionQuantity,
		executionPrice,
	); err != nil {
		return err
	}

	// Mark both orders as completely filled.
	result, err := queries.UpdateFilledQuantity(
		ctx,
		database.UpdateFilledQuantityParams{
			FilledQuantity:   executionQuantity.String(),
			FilledQuantity_2: executionQuantity.String(),
			ID:               buy.ID,
			Quantity:         executionQuantity.String(),
		},
	)
	if err != nil {
		return fmt.Errorf("updating buyer fill: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking buyer fill: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("failed to fill buyer order")
	}

	result, err = queries.UpdateFilledQuantity(
		ctx,
		database.UpdateFilledQuantityParams{
			FilledQuantity:   executionQuantity.String(),
			FilledQuantity_2: executionQuantity.String(),
			ID:               sell.ID,
			Quantity:         executionQuantity.String(),
		},
	)
	if err != nil {
		return fmt.Errorf("updating seller fill: %w", err)
	}

	rowsAffected, err = result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking seller fill: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("failed to fill seller order")
	}

	// Record the actual trade.
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

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing match: %w", err)
	}

	return nil
}

func (s *Service) executeBuy(
	ctx context.Context,
	queries *database.Queries,
	buy database.Order,
	quantity decimal.Decimal,
	price decimal.Decimal,
) error {
	buyAccountID := buy.AccountID

	totalCost := quantity.Mul(price)

	// The BUY order already served this money.
	// Move it from reserved to spent.
	result, err := queries.SpendReservedFunds(ctx, database.SpendReservedFundsParams{
		Balance:           totalCost.String(),
		ReservedBalance:   totalCost.String(),
		ID:                buyAccountID,
		ReservedBalance_2: totalCost.String(),
	})
	if err != nil {
		return fmt.Errorf("spending buyer funds: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking buyer funds: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("buyer has insufficient reserved funds")
	}

	position, err := queries.GetPosition(ctx, database.GetPositionParams{
		AccountID:    buy.AccountID,
		InstrumentID: buy.InstrumentID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		err = queries.CreatePosition(ctx, database.CreatePositionParams{
			ID:               uuid.New().String(),
			AccountID:        buy.AccountID,
			InstrumentID:     buy.InstrumentID,
			Quantity:         quantity.String(),
			ReservedQuantity: "0",
			AveragePrice:     price.String(),
		})
		if err != nil {
			return fmt.Errorf("creating buyer position: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("getting buyer positions: %w", err)
	} else {
		oldQuantity, err := decimal.NewFromString(position.Quantity)
		if err != nil {
			return fmt.Errorf("parsing buyer average price: %w", err)
		}

		oldAveragePrice, err := decimal.NewFromString(position.AveragePrice)
		if err != nil {
			return fmt.Errorf("parsing buyer average price: %w", err)
		}

		newQuantity := oldQuantity.Add(quantity)

		oldCost := oldQuantity.Mul(oldAveragePrice)
		newCost := quantity.Mul(price)

		newAveragePrice := oldCost.
			Add(newCost).
			Div(newQuantity)

		err = queries.UpdatePosition(
			ctx, database.UpdatePositionParams{
				Quantity:         newQuantity.String(),
				ReservedQuantity: position.ReservedQuantity,
				AveragePrice:     newAveragePrice.String(),
				ID:               position.ID,
			},
		)
		if err != nil {
			return fmt.Errorf("updating buyer position: %w", err)
		}
	}
	return nil
}

func (s *Service) executeSell(
	ctx context.Context,
	queries *database.Queries,
	sell database.Order,
	quantity decimal.Decimal,
	price decimal.Decimal,
) error {
	position, err := queries.GetPosition(
		ctx,
		database.GetPositionParams{
			AccountID:    sell.AccountID,
			InstrumentID: sell.InstrumentID,
		},
	)
	if err != nil {
		return fmt.Errorf("getting seller position: %w", err)
	}

	result, err := queries.ExecuteSellPosition(
		ctx,
		database.ExecuteSellPositionParams{
			Quantity:           quantity.String(),
			ReservedQuantity:   quantity.String(),
			ID:                 position.ID,
			Quantity_2:         quantity.String(),
			ReservedQuantity_2: quantity.String(),
		},
	)
	if err != nil {
		return fmt.Errorf("selling position: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking seller position: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("seller has insufficient reserved position")
	}

	proceeds := quantity.Mul(price)

	err = queries.ReceiveFunds(
		ctx,
		database.ReceiveFundsParams{
			Balance: proceeds.String(),
			ID:      sell.AccountID,
		},
	)
	if err != nil {
		return fmt.Errorf("paying seller: %w", err)
	}

	return nil
}
