package httpapi

import (
	"encoding/json"
	"net/http"
)

func (s *Server) listInstruments(w http.ResponseWriter, r *http.Request) {
	instruments, err := s.instrumentService.List(r.Context())
	if err != nil {
		http.Error(w, "failed to get instruments", http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(instruments); err != nil {
		return
	}
}
