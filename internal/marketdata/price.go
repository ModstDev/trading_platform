package marketdata

import (
	"sync"
	"time"

	"github.com/shopspring/decimal"
)

type Price struct {
	Symbol    string
	Value     decimal.Decimal
	Timestamp time.Time
}

type PriceStore struct {
	mu     sync.RWMutex
	prices map[string]Price
}

func NewPriceStore() *PriceStore {
	return &PriceStore{
		prices: make(map[string]Price),
	}
}

func (s *PriceStore) Set(price Price) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.prices[price.Symbol] = price
}

func (s *PriceStore) Get(symbol string) (Price, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	price, ok := s.prices[symbol]
	return price, ok
}

func (s *PriceStore) GetAll() []Price {
	s.mu.RLock()
	defer s.mu.RUnlock()

	prices := make([]Price, 0, len(s.prices))

	for _, price := range s.prices {
		prices = append(prices, price)
	}

	return prices
}
