package httpapi

import (
	"encoding/json"
	"net/http"
)

type marketPriceResponse struct {
	Symbol    string `json:"symbol"`
	Price     string `json:"price"`
	Timestamp string `json:"timestamp"`
}

func (s *Server) getMarketPrices(w http.ResponseWriter, r *http.Request) {
	prices := s.priceStore.GetAll()

	response := make([]marketPriceResponse, 0, len(prices))

	for _, price := range prices {
		response = append(response, marketPriceResponse{
			Symbol:    price.Symbol,
			Price:     price.Value.String(),
			Timestamp: price.Timestamp.UTC().Format("2006-01-02T15:04:05Z"),
		})
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(response); err != nil {
		return
	}
}
