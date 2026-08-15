package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/ModstDev/trading_platform/internal/order"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type CreateOrderRequest struct {
	InstrumentID uuid.UUID        `json:"instrument_id"`
	Side         order.Side       `json:"side"`
	Type         order.Type       `json:"type"`
	Quantity     *decimal.Decimal `json:"quantity"`
	Price        *decimal.Decimal `json:"price"`
}

func (s *Server) createOrder(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		http.Error(w, "user ID not found", http.StatusUnauthorized)
		return
	}

	var request CreateOrderRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
	}

	account, err := s.accountService.GetByUserID(r.Context(), userID)
	if err != nil {
		http.Error(w, "account not found", http.StatusNotFound)
	}

	accountID, err := uuid.Parse(account.ID)
	if err != nil {
		return
	}

	createdOrder, err := s.orderService.Create(r.Context(), order.CreateInput{
		AccountID:    accountID,
		InstrumentID: request.InstrumentID,
		Side:         request.Side,
		Type:         request.Type,
		Quantity:     request.Quantity,
		Price:        request.Price,
	})

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(createdOrder)
}
