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

	if input.Side != SideBuy && input.Side != SideSell {
		return nil, errors.New("invalid order side")
	}

	if input.Price == nil {
		return nil, errors.New("price is required for limit orders")
	}

	if input.Quantity.LessThanOrEqual(decimal.Zero) {
		return nil, errors.New("quantity must be greater than zero")
	}

	if input.Quantity == nil || input.Quantity.LessThanOrEqual(decimal.Zero) {
		return nil, errors.New("quantity must be greater than zero")
	}

	if input.Price.LessThanOrEqual(decimal.Zero) {
		return nil, errors.New("price must be greater than zero")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	queries := database.New(tx)

	if input.Side == SideBuy {
		requiredFunds := input.Quantity.Mul(*input.Price)

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
	}

	if input.Side == SideSell {
		position, err := queries.GetPosition(
			ctx,
			database.GetPositionParams{
				AccountID:    input.AccountID.String(),
				InstrumentID: input.InstrumentID.String(),
			},
		)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, errors.New("position not found")
			}

			return nil, fmt.Errorf("getting position: %w", err)
		}

		result, err := queries.ReservePositionQuantity(
			ctx,
			database.ReservePositionQuantityParams{
				ReservedQuantity: input.Quantity.String(),
				ID:               position.ID,
				Quantity:         input.Quantity.String(),
			},
		)
		if err != nil {
			return nil, fmt.Errorf("reserving position: %w", err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return nil, fmt.Errorf("checking reserved position: %w", err)
		}

		if rowsAffected == 0 {
			return nil, errors.New("insufficient available position")
		}
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

func (s *Service) Cancel(ctx context.Context, orderID uuid.UUID, accountID uuid.UUID) error {
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

	if order.AccountID != accountID.String() {
		return errors.New("order does not belong to account")
	}

	if order.Status != string(StatusPending) {
		return errors.New("only pending orders can be cancelled")
	}

	// Convert the order values back to decimals.
	quantity, err := decimal.NewFromString(order.Quantity)
	if err != nil {
		return fmt.Errorf("parsing order quantity: %w", err)
	}

	if order.Side == string(SideBuy) {
		if !order.Price.Valid {
			return errors.New("order has no price")
		}

		price, err := decimal.NewFromString(order.Price.String)
		if err != nil {
			return fmt.Errorf("parsing order price: %w", err)
		}

		reservedAmount := quantity.Mul(price)

		result, err := queries.ReleaseFunds(ctx, database.ReleaseFundsParams{
			ReservedBalance:   reservedAmount.String(),
			ID:                accountID.String(),
			ReservedBalance_2: reservedAmount.String(),
		})
		if err != nil {
			return fmt.Errorf("releasing funds: %w", err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("checking released funds: %w", err)
		}

		if rowsAffected == 0 {
			return errors.New("failed to release reserved funds")
		}
	}

	if order.Side == string(SideSell) {
		position, err := queries.GetPosition(ctx, database.GetPositionParams{
			AccountID:    accountID.String(),
			InstrumentID: order.InstrumentID,
		})
		if err != nil {
			return fmt.Errorf("getting position: %w", err)
		}

		result, err := queries.ReleasePositionQuantity(ctx, database.ReleasePositionQuantityParams{
			ReservedQuantity:   quantity.String(),
			ID:                 position.ID,
			ReservedQuantity_2: quantity.String(),
		})
		if err != nil {
			return fmt.Errorf("releasing position: %w", err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("checking released position: %w", err)
		}

		if rowsAffected == 0 {
			return errors.New("failed to release reserved position")
		}
	}

	resultErr := queries.CancelOrder(ctx, database.CancelOrderParams{
		ID:        order.ID,
		AccountID: accountID.String(),
	})
	if resultErr != nil {
		return fmt.Errorf("cancelling order: %w", resultErr)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("comitting transacation: %w", err)
	}

	return nil
}

func (s *Service) Execute(ctx context.Context, orderID uuid.UUID, accountID uuid.UUID) error {
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

	if order.AccountID != accountID.String() {
		return errors.New("order does not belong to account")
	}

	if order.Status != string(StatusPending) {
		return errors.New("only pending orders can be executed")
	}

	if !order.Price.Valid {
		return errors.New("order has no price")
	}

	quantity, err := decimal.NewFromString(order.Quantity)
	if err != nil {
		return fmt.Errorf("parsing order quantity: %w", err)
	}

	price, err := decimal.NewFromString(order.Price.String)
	if err != nil {
		return fmt.Errorf("parsing order price: %w", err)
	}

	if order.Side == string(SideBuy) {
		totalCost := quantity.Mul(price)

		// Spend the money that was previously reserved.
		result, err := queries.SpendReservedFunds(ctx, database.SpendReservedFundsParams{
			Balance:           totalCost.String(),
			ReservedBalance:   totalCost.String(),
			ID:                accountID.String(),
			ReservedBalance_2: totalCost.String(),
		})
		if err != nil {
			return fmt.Errorf("spending reserved funs: %w", err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("checking spent funds: %w", err)
		}

		if rowsAffected == 0 {
			return errors.New("failed to spend reserved funds")
		}

		// Find the existing position.
		position, err := queries.GetPosition(ctx, database.GetPositionParams{
			AccountID:    accountID.String(),
			InstrumentID: order.InstrumentID,
		})
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("getting position: %w", err)
			}

			// User doesn't own this instrument yet
			err = queries.CreatePosition(ctx, database.CreatePositionParams{
				ID:               uuid.New().String(),
				AccountID:        accountID.String(),
				InstrumentID:     order.InstrumentID,
				Quantity:         quantity.String(),
				ReservedQuantity: "0",
				AveragePrice:     price.String(),
			},
			)
			if err != nil {
				return fmt.Errorf("creating position: %w", err)
			}
		} else {
			// Existing position.
			oldQuantity, err := decimal.NewFromString(position.Quantity)
			if err != nil {
				return fmt.Errorf("parsing existing quantity: %w", err)
			}

			oldAveragePrice, err := decimal.NewFromString(position.AveragePrice)
			if err != nil {
				return fmt.Errorf("parsing existing average price: %w", err)
			}

			newQuantity := oldQuantity.Add(quantity)

			oldCost := oldQuantity.Mul(oldAveragePrice)
			newCost := quantity.Mul(price)

			newAveragePrice := oldCost.Add(newCost).Div(newQuantity)

			err = queries.UpdatePosition(ctx, database.UpdatePositionParams{
				Quantity:     newQuantity.String(),
				AveragePrice: newAveragePrice.String(),
				ID:           position.ID,
			})
			if err != nil {
				return fmt.Errorf("updating position: %w", err)
			}
		}
	}

	if order.Side == string(SideSell) {
		position, err := queries.GetPosition(ctx, database.GetPositionParams{
			AccountID:    accountID.String(),
			InstrumentID: order.InstrumentID,
		})
		if err != nil {
			return fmt.Errorf("getting position: %w", err)
		}

		result, err := queries.ExecuteSellPosition(ctx, database.ExecuteSellPositionParams{
			Quantity:           quantity.String(),
			ReservedQuantity:   quantity.String(),
			ID:                 position.ID,
			Quantity_2:         quantity.String(),
			ReservedQuantity_2: quantity.String(),
		})
		if err != nil {
			return fmt.Errorf("executing sell position: %w", err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("checking sold position: %w", err)
		}

		if rowsAffected == 0 {
			return errors.New("failed to sell position")
		}

		proceeds := quantity.Mul(price)

		err = queries.ReceiveFunds(ctx, database.ReceiveFundsParams{
			Balance: proceeds.String(),
			ID:      accountID.String(),
		})
		if err != nil {
			return fmt.Errorf("receiving funds: %w", err)
		}
	}

	// Mark the order as executed
	result, err := queries.ExecuteOrder(ctx, database.ExecuteOrderParams{
		ID:        orderID.String(),
		AccountID: accountID.String(),
	})
	if err != nil {
		return fmt.Errorf("executing order: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking executed order: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("order was not executed")
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("comitting transaction: %w", err)
	}

	return nil
}
