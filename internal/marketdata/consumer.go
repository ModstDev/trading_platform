package marketdata

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/shopspring/decimal"
)

type priceEvent struct {
	Event     string  `json:"event"`
	Symbol    string  `json:"symbol"`
	Price     float64 `json:"price"`
	Timestamp int64   `json:"timestamp"`
}

func StartConsumer(nc *nats.Conn, store *PriceStore) (*nats.Subscription, error) {
	sub, err := nc.Subscribe("market.price.*", func(msg *nats.Msg) {
		var event priceEvent

		if err := json.Unmarshal(msg.Data, &event); err != nil {
			log.Printf("NATS: failed to decode price: %v", err)
			return
		}

		value := decimal.NewFromFloat(event.Price)

		symbol := strings.ToUpper(event.Symbol)

		store.Set(Price{
			Symbol:    symbol,
			Value:     value,
			Timestamp: time.Unix(event.Timestamp, 0),
		})

		log.Printf("NATS: updated market price %s = %s", symbol, value.String())
	})
	if err != nil {
		return nil, fmt.Errorf("subscribing to market prices: %w", err)
	}

	return sub, nil
}
