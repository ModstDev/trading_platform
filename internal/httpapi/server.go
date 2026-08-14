package httpapi

import (
	"context"
	"net/http"

	"github.com/ModstDev/trading_platform/internal/database"
	"github.com/ModstDev/trading_platform/internal/user"
)

type UserService interface {
	Register(ctx context.Context, input user.RegisterInput) (*database.User, error)
}

type Server struct {
	userService UserService
}

func NewServer(userService UserService) *Server {
	return &Server{
		userService: userService,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /users", s.registerUser)

	return mux
}
