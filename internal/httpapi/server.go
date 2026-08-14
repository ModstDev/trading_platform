package httpapi

import (
	"net/http"

	"github.com/ModstDev/trading_platform/internal/user"
)

type Server struct {
	userService *user.Service
}

func NewServer(userService *user.Service) *Server {
	return &Server{
		userService: userService,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /users", s.registerUser)

	return mux
}
