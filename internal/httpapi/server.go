package httpapi

import (
	"context"
	"net/http"

	"github.com/ModstDev/trading_platform/internal/database"
	"github.com/ModstDev/trading_platform/internal/order"
	"github.com/ModstDev/trading_platform/internal/user"
	"github.com/google/uuid"
)

type UserService interface {
	Register(ctx context.Context, input user.RegisterInput) (*database.User, error)
	Login(ctx context.Context, input user.LoginInput) (*database.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*database.User, error)
}

type AccountService interface {
	GetByUserID(ctx context.Context, userID uuid.UUID) (*database.Account, error)
}

type InstrumentService interface {
	GetByID(ctx context.Context, id uuid.UUID) (*database.Instrument, error)
	GetBySymbol(ctx context.Context, symbol string) (*database.Instrument, error)
	List(ctx context.Context) ([]database.Instrument, error)
}

type OrderService interface {
	Create(ctx context.Context, input order.CreateInput) (*database.Order, error)
	ListByAccountID(ctx context.Context, accountID uuid.UUID) ([]database.Order, error)
	Cancel(ctx context.Context, orderID uuid.UUID, accountID uuid.UUID) error
}

type Server struct {
	userService       UserService
	accountService    AccountService
	instrumentService InstrumentService
	orderService      OrderService
	jwtSecret         string
}

func NewServer(userService UserService, accountService AccountService, instrumentService InstrumentService, orderService OrderService, jwtSecret string) *Server {
	return &Server{
		userService:       userService,
		accountService:    accountService,
		instrumentService: instrumentService,
		orderService:      orderService,
		jwtSecret:         jwtSecret,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /users", s.registerUser)
	mux.HandleFunc("POST /login", s.loginUser)
	mux.Handle("POST /orders", s.requireAuth(http.HandlerFunc(s.createOrder)))
	mux.Handle("DELETE /orders/{id}", s.requireAuth(http.HandlerFunc(s.cancelOrder)))

	mux.Handle("GET /me", s.requireAuth(http.HandlerFunc(s.getMe)))
	mux.Handle("GET /account", s.requireAuth(http.HandlerFunc(s.getAccount)))
	mux.HandleFunc("GET /instruments", s.listInstruments)
	mux.Handle("GET /orders", s.requireAuth(http.HandlerFunc(s.listOrders)))
	return mux
}
