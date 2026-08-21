package main

import (
	"log"
	"os"

	"github.com/ModstDev/trading_platform/internal/marketdata"
	"github.com/nats-io/nats.go"
)

func main() {
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}

	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Fatalf("connecting to NATS: %v", err)
	}
	defer nc.Close()

	log.Printf("NATS connected: %s", natsURL)

	priceStore := marketdata.NewPriceStore()

	priceSubscription, err := marketdata.StartConsumer(nc, priceStore)
	if err != nil {
		log.Fatalf("starting market data consumer: %v", err)
	}
	defer priceSubscription.Unsubscribe()

	log.Println("subscribed to market.price.*")

	select {}
}
