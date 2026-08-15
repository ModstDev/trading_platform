package httpapi

import (
	"context"
	"net/http"

	"github.com/ModstDev/trading_platform/internal/database"
	"github.com/ModstDev/trading_platform/internal/user"
	"github.com/google/uuid"
)

type UserService interface {
	Register(ctx context.Context, input user.RegisterInput) (*database.User, error)
	Login(ctx context.Context, input user.LoginInput) (*database.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*database.User, error)
}

type Server struct {
	userService UserService
	jwtSecret   string
}

func NewServer(userService UserService, jwtSecret string) *Server {
	return &Server{
		userService: userService,
		jwtSecret:   jwtSecret,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /users", s.registerUser)
	mux.HandleFunc("POST /login", s.loginUser)
	mux.Handle("GET /me", s.requireAuth(http.HandlerFunc(s.getMe)))
	return mux
}
