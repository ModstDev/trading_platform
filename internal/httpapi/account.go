package httpapi

import (
	"encoding/json"
	"net/http"
)

type AccountResponse struct {
	ID              string `json:"id"`
	UserID          string `json:"user_id"`
	Balance         string `json:"balance"`
	ReservedBalance string `json:"reserved_balance"`
	Currency        string `json:"currency"`
}

func (s *Server) getAccount(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		http.Error(w, "user ID not found", http.StatusUnauthorized)
	}

	account, err := s.accountService.GetByUserID(r.Context(), userID)
	if err != nil {
		http.Error(w, "account not fund", http.StatusNotFound)
	}

	response := AccountResponse{
		ID:              account.ID,
		UserID:          account.UserID,
		Balance:         account.Balance,
		ReservedBalance: account.ReservedBalance,
		Currency:        account.Currency,
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(response); err != nil {
		return
	}
}
