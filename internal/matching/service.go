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

	var totalBuyCost decimal.Decimal

	for remaining.GreaterThan(decimal.Zero) {
		var match database.Order

		switch order.Type {
		case "LIMIT":
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

		case "MARKET":
			switch order.Side {
			case "BUY":
				match, err = queries.FindBestSellOrder(
					ctx,
					database.FindBestSellOrderParams{
						InstrumentID: order.InstrumentID,
						ID:           order.ID,
					},
				)

			case "SELL":
				match, err = queries.FindBestBuyOrder(
					ctx,
					database.FindBestBuyOrderParams{
						InstrumentID: order.InstrumentID,
						ID:           order.ID,
					},
				)

			default:
				return errors.New("invalid order side")
			}
		default:
			return errors.New("invalid order type")
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

		if order.Side == "BUY" {
			totalBuyCost = totalBuyCost.Add(
				executionQuantity.Mul(executionPrice),
			)
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

	if order.Type == "MARKET" &&
		order.Side == "BUY" &&
		remaining.GreaterThanOrEqual(decimal.Zero) {

		maxCost, err := decimal.NewFromString(order.MaxCost.String)
		if err != nil {
			return fmt.Errorf("parsing market max cost: %w", err)
		}

		// totalBuyCost needs to contain the sum of all
		// actual execution costs.
		unusedFunds := maxCost.Sub(totalBuyCost)

		if unusedFunds.GreaterThan(decimal.Zero) {
			_, err = queries.ReleaseFunds(
				ctx,
				database.ReleaseFundsParams{
					ReservedBalance:   unusedFunds.String(),
					ID:                order.AccountID,
					ReservedBalance_2: unusedFunds.String(),
				},
			)
			if err != nil {
				return fmt.Errorf("releasing unused market funds: %w", err)
			}
		}
	}

	if order.Type == "MARKET" && remaining.GreaterThan(decimal.Zero) {
		err = queries.CancelUnfilledMarketOrder(
			ctx,
			order.ID,
		)
		if err != nil {
			return fmt.Errorf("canceling unfilled market order: %w", err)
		}
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

	actualCost := quantity.Mul(price)

	// LIMIT orders reserve based on their limit price.
	// MARKET orders reserve based on max_cost.

	var reservedCost decimal.Decimal

	switch buy.Type {
	case "LIMIT":
		limitPrice, err := decimal.NewFromString(buy.Price.String)
		if err != nil {
			return fmt.Errorf("parsing buyer limit price: %w", err)
		}

		reservedCost = quantity.Mul(limitPrice)

	case "MARKET":
		maxCost, err := decimal.NewFromString(buy.MaxCost.String)
		if err != nil {
			return fmt.Errorf("parsing buyer max cost: %w", err)
		}

		// The order's total reservation is max_cost, not
		// quantity * execution price.
		reservedCost = maxCost

	default:
		return errors.New("invalid buyer order type")
	}

	// The BUY order already served this money.
	// Move it from reserved to spent.
	result, err := queries.SpendReservedFunds(ctx, database.SpendReservedFundsParams{
		Balance:           actualCost.String(),
		ReservedBalance:   actualCost.String(),
		ID:                buyAccountID,
		ReservedBalance_2: actualCost.String(),
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

	// For LIMIT orders, release the difference between
	// the reserved limit price and the actual execution price.
	//
	// For MARKET orders, we DON'T release max_cost - actualCost
	// here because a single market order can execute against
	// multiple sellers. We handle its remaining reservation
	// after matching finishes.

	if buy.Type == "LIMIT" {
		priceDifference := reservedCost.Sub(actualCost)

		if priceDifference.GreaterThan(decimal.Zero) {
			_, err := queries.ReleaseFunds(
				ctx,
				database.ReleaseFundsParams{
					ReservedBalance:   priceDifference.String(),
					ID:                buyAccountID,
					ReservedBalance_2: priceDifference.String(),
				},
			)
			if err != nil {
				return fmt.Errorf("releasing unused buyer funds: %w", err)
			}
		}
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
