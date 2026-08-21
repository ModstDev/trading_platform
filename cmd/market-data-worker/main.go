package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	"github.com/nats-io/nats.go"
)

const (
	twelveDataURL = "wss://ws.twelvedata.com/v1/quotes/price"
	priceSubject  = "market.price"
)

type PriceEvent struct {
	Event     string  `json:"event"`
	Symbol    string  `json:"symbol"`
	Price     float64 `json:"price"`
	Timestamp int64   `json:"timestamp"`
	DayVolume float64 `json:"day_volume,omitempty"`
}

type SubscribeMessage struct {
	Action string `json:"action"`
	Params struct {
		Symbols string `json:"symbols"`
	} `json:"params"`
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := godotenv.Load(); err != nil {
		log.Printf("could not load .env: %v", err)
	}

	apiKey := os.Getenv("TWELVE_DATA_API_KEY")
	if apiKey == "" {
		log.Fatal("TWELVE_DATA_API_KEY is not set")
	}

	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}

	symbols := os.Getenv("MARKET_DATA_SYMBOLS")
	if symbols == "" {
		symbols = "AAPL,MSFT,TSLA"
	}

	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Fatalf("connecting to NATS: %v", err)
	}
	defer nc.Drain()

	log.Printf("NATS connected: %s", natsURL)

	wsURL := fmt.Sprintf("%s?apikey=%s", twelveDataURL, apiKey)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		log.Fatalf("connecting to Twelve Data: %v", err)
	}
	defer conn.Close()

	log.Printf("connected to Twelve Data")

	subscribe := SubscribeMessage{
		Action: "subscribe",
	}

	subscribe.Params.Symbols = cleanSymbols(symbols)

	if err := conn.WriteJSON(subscribe); err != nil {
		log.Fatalf("subscribing to symbols: %v", err)
	}

	log.Printf("subscribed to: %s", subscribe.Params.Symbols)

	go func() {
		<-ctx.Done()

		log.Println("closing Twelve Data WebSocket")

		if err := conn.Close(); err != nil {
			log.Printf("closing Twelve Data WebSocket: %v", err)
		}
	}()

	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Println("stopping Twelve Data heartbeat")
				return

			case <-ticker.C:
				heartbeat := struct {
					Action string `json:"action"`
				}{
					Action: "heartbeat",
				}

				if err := conn.WriteJSON(heartbeat); err != nil {
					if ctx.Err() != nil {
						return
					}

					log.Printf("sending heartbeat: %v", err)
					return
				}

				log.Println("Twelve Data heartbeat sent")
			}
		}
	}()

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			if ctx.Err() != nil {
				log.Println("market-data worker stopped")
				return
			}

			log.Fatalf("reading Twelve Data message: %v", err)
		}

		if err := handleMessage(nc, data); err != nil {
			log.Printf("handling market data: %v", err)
		}
	}
}

func handleMessage(nc *nats.Conn, data []byte) error {
	var event PriceEvent

	if err := json.Unmarshal(data, &event); err != nil {
		return fmt.Errorf("decoding message: %w", err)
	}

	switch event.Event {
	case "price":
		return publishPrice(nc, event)

	case "subscribe-status":
		log.Printf("Twelve Data subscription status: %s", string(data))
		return nil

	default:
		log.Printf("Twelve Data event: %s", string(data))
		return nil
	}
}

func publishPrice(nc *nats.Conn, event PriceEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("encoding price event: %w", err)
	}

	subject := fmt.Sprintf("%s.%s", priceSubject, event.Symbol)

	if err := nc.Publish(subject, payload); err != nil {
		return fmt.Errorf("publishing %s: %w", subject, err)
	}

	log.Printf("market price: %s = %.4f", event.Symbol, event.Price)

	return nil
}

func cleanSymbols(value string) string {
	parts := strings.Split(value, ",")

	var symbols []string

	for _, part := range parts {
		symbol := strings.TrimSpace(strings.ToUpper(part))

		if symbol != "" {
			symbols = append(symbols, symbol)
		}
	}

	return strings.Join(symbols, ",")
}

func init() {
	log.SetFlags(log.Ldate | log.Ltime)
}
