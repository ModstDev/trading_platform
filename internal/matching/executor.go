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

func (s *Service) executeMarketOrder(ctx context.Context, queries *database.Queries, order database.Order, quantity decimal.Decimal,
) error {
	instrument, err := queries.GetInstrumentByID(ctx, order.InstrumentID)
	if err != nil {
		return fmt.Errorf("getting instrument: %w", err)
	}

	marketPrice, ok := s.priceStore.Get(instrument.Symbol)
	if !ok {
		return fmt.Errorf("no market price available for %s", instrument.Symbol)
	}

	price := marketPrice.Value

	switch order.Side {
	case "BUY":

		maxCost, err := decimal.NewFromString(order.MaxCost.String)
		if err != nil {
			return fmt.Errorf("parsing market max cost: %w", err)
		}

		actualCost := quantity.Mul(price)

		if actualCost.GreaterThan(maxCost) {
			// The order can never execute at this price.
			if _, err := queries.ReleaseFunds(
				ctx,
				database.ReleaseFundsParams{
					ReservedBalance:   maxCost.String(),
					ID:                order.AccountID,
					ReservedBalance_2: maxCost.String(),
				},
			); err != nil {
				return fmt.Errorf("releasing market order funds: %w", err)
			}

			if err := queries.CancelUnfilledMarketOrder(
				ctx,
				order.ID,
			); err != nil {
				return fmt.Errorf("canceling market order: %w", err)
			}

			return nil
		}

		if err := s.executeBuy(ctx, queries, order, quantity, price); err != nil {
			return fmt.Errorf("executing market buy: %w", err)
		}
		unusedFunds := maxCost.Sub(actualCost)

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

	case "SELL":
		if err := s.executeSell(ctx, queries, order, quantity, price); err != nil {
			return fmt.Errorf("executing market sell: %w", err)
		}

	default:
		return errors.New("invalid market order side")
	}

	_, err = queries.UpdateFilledQuantity(
		ctx,
		database.UpdateFilledQuantityParams{
			FilledQuantity:   quantity.String(),
			FilledQuantity_2: quantity.String(),
			ID:               order.ID,
			Quantity:         quantity.String(),
		},
	)

	if err != nil {
		return fmt.Errorf("updating market order fill: %w", err)
	}

	err = queries.CreateExecution(ctx, database.CreateExecutionParams{
		ID:           uuid.New().String(),
		OrderID:      order.ID,
		AccountID:    order.AccountID,
		InstrumentID: order.InstrumentID,
		Quantity:     quantity.String(),
		Price:        price.String(),
	})
	if err != nil {
		return fmt.Errorf("creating market execution: %w", err)
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
	var reservedCost decimal.Decimal

	switch buy.Type {
	case "LIMIT":
		limitPrice, err := decimal.NewFromString(buy.Price.String)
		if err != nil {
			return fmt.Errorf("parsing buyer limit price: %w", err)
		}

		reservedCost = quantity.Mul(limitPrice)

	case "MARKET":
		if !buy.MaxCost.Valid {
			return errors.New("market buy has no max cost")
		}

		var err error
		reservedCost, err = decimal.NewFromString(buy.MaxCost.String)
		if err != nil {
			return fmt.Errorf("parsing market max cost: %w", err)
		}

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
