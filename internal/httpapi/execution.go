package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
)

func (s *Server) listExecutions(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		http.Error(w, "user ID not found", http.StatusUnauthorized)
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

	executions, err := s.executionService.ListByAccountID(r.Context(), accountID)
	if err != nil {
		http.Error(w, "failed to get executions", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(executions); err != nil {
		return
	}
}
