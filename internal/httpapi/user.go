package httpapi

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/ModstDev/trading_platform/internal/user"
)

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UserResponse struct {
	ID        uint64    `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *Server) registerUser(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	createdUser, err := s.userService.Register(
		r.Context(),
		user.RegisterInput{
			Email:    req.Email,
			Password: req.Password,
		},
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response := UserResponse{
		ID:        createdUser.ID,
		Email:     createdUser.Email,
		CreatedAt: createdUser.CreatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		// At this point the response may already have been sent,
		// so there is no useful HTTP status we can change to.
		return
	}
}
