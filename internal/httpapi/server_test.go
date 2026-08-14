package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ModstDev/trading_platform/internal/database"
	"github.com/ModstDev/trading_platform/internal/user"
)

type fakeUserService struct {
	register func(context.Context, user.RegisterInput) (*database.User, error)
}

func (f *fakeUserService) Register(
	ctx context.Context,
	input user.RegisterInput,
) (*database.User, error) {
	return f.register(ctx, input)
}

func TestRegisterUser(t *testing.T) {
	var gotInput user.RegisterInput

	service := &fakeUserService{
		register: func(
			ctx context.Context,
			input user.RegisterInput,
		) (*database.User, error) {
			gotInput = input

			return &database.User{
				ID:    1,
				Email: input.Email,
			}, nil
		},
	}

	server := NewServer(service)

	req := httptest.NewRequest(
		http.MethodPost,
		"/users",
		strings.NewReader(`{
			"email": "test@example.com",
			"password": "password123"
		}`),
	)

	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}

	if gotInput.Email != "test@example.com" {
		t.Fatalf("expected email %q, got %q", "test@example.com", gotInput.Email)
	}

	if gotInput.Password != "password123" {
		t.Fatalf("expected password %q, got %q", "password123", gotInput.Password)
	}
}
