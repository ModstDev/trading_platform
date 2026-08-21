package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ModstDev/trading_platform/internal/config"
	"github.com/ModstDev/trading_platform/internal/database"
	"github.com/ModstDev/trading_platform/internal/marketdata"
	"github.com/ModstDev/trading_platform/internal/matching"
	"github.com/ModstDev/trading_platform/internal/pubsub"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("warning: .env file not found")
	}

	cfg := config.Load()

	db, err := database.Connect(cfg.Database)
	if err != nil {
		log.Fatalf("connecting to database: %v", err)
	}
	defer db.Close()

	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}

	natsClient, err := pubsub.NewNATS(natsURL)
	if err != nil {
		log.Fatalf("failed to connect to NATS: %v", err)
	}
	defer natsClient.Close()

	priceStore := marketdata.NewPriceStore()

	priceSubscription, err := marketdata.StartConsumer(
		natsClient.Conn(),
		priceStore,
	)
	if err != nil {
		log.Fatalf("starting market data consumer: %v", err)
	}
	defer priceSubscription.Unsubscribe()

	matchingService := matching.NewService(db, priceStore)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := natsClient.StartMatchingConsumer(ctx, matchingService); err != nil {
		log.Fatal(err)
	}

	log.Println("matching worker started")

	<-ctx.Done()

	log.Println("matching worker stopped")
}
