package httpapi

import (
	"net/http"

	"github.com/google/uuid"
)

func (s *Server) matchOrder(w http.ResponseWriter, r *http.Request,
) {
	orderID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid order ID", http.StatusBadRequest)
		return
	}

	if err := s.matchingService.MatchOrder(
		r.Context(),
		orderID,
	); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
