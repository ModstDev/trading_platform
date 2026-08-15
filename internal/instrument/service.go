package instrument

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/ModstDev/trading_platform/internal/database"
)

type Service struct {
	queries *database.Queries
}

func NewService(queries *database.Queries) *Service {
	return &Service{
		queries: queries,
	}
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*database.Instrument, error) {
	instrument, err := s.queries.GetInstrumentByID(ctx, id.String())
	if err != nil {
		return nil, fmt.Errorf("getting instrument: %w", err)
	}

	return &instrument, nil
}

func (s *Service) GetBySymbol(ctx context.Context, symbol string) (*database.Instrument, error) {
	instrument, err := s.queries.GetInstrumentBySymbol(ctx, symbol)
	if err != nil {
		return nil, fmt.Errorf("getting instrument: %w", err)
	}

	return &instrument, nil
}

func (s *Service) List(ctx context.Context) ([]database.Instrument, error) {
	instruments, err := s.queries.ListInstruments(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing instruments: %w", err)
	}

	return instruments, nil
}
