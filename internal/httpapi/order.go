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
	MaxCost      *decimal.Decimal `json:"max_cost"`
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
		return
	}

	account, err := s.accountService.GetByUserID(r.Context(), userID)
	if err != nil {
		http.Error(w, "account not found", http.StatusNotFound)
		return
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
		MaxCost:      request.MaxCost,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.nats.PublishOrderCreated(r.Context(), createdOrder.ID); err != nil {
		http.Error(w, "failed to publish order created event", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(createdOrder)
}

func (s *Server) listOrders(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		http.Error(w, "user ID not found", http.StatusUnauthorized)
		return
	}

	account, err := s.accountService.GetByUserID(r.Context(), userID)
	if err != nil {
		http.Error(w, "account not found", http.StatusNotFound)
		return
	}

	accountID, err := uuid.Parse(account.ID)
	if err != nil {
		http.Error(w, "account not found", http.StatusNotFound)
		return
	}
	orders, err := s.orderService.ListByAccountID(r.Context(), accountID)
	if err != nil {
		http.Error(w, "failed to get orders", http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(orders); err != nil {
		return
	}
}

func (s *Server) cancelOrder(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		http.Error(w, "user ID not found", http.StatusUnauthorized)
		return
	}

	account, err := s.accountService.GetByUserID(r.Context(), userID)
	if err != nil {
		http.Error(w, "account not found", http.StatusNotFound)
		return
	}

	orderID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid order ID", http.StatusBadRequest)
	}
	accountID, err := uuid.Parse(account.ID)
	if err != nil {
		http.Error(w, "invalid account ID", http.StatusBadRequest)
	}

	if err := s.orderService.Cancel(r.Context(), orderID, accountID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) executeOrder(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		http.Error(w, "user ID not found", http.StatusUnauthorized)
		return
	}

	account, err := s.accountService.GetByUserID(r.Context(), userID)
	if err != nil {
		http.Error(w, "account not found", http.StatusNotFound)
		return
	}

	orderID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid order ID", http.StatusBadRequest)
		return
	}

	accountID, err := uuid.Parse(account.ID)
	if err != nil {
		http.Error(w, "invalid account ID", http.StatusBadRequest)
		return
	}

	if err := s.orderService.Execute(r.Context(), orderID, accountID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
