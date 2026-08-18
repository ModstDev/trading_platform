package main

import (
	"context"
	"log"

	"github.com/ModstDev/trading_platform/internal/config"
	"github.com/ModstDev/trading_platform/internal/database"
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

	matchingService := matching.NewService(db)

	natsClient, err := pubsub.NewNATS("nats://localhost:4222")
	if err != nil {
		log.Fatalf("failed to connect to NATS: %v", err)
	}
	defer natsClient.Close()

	ctx := context.Background()
	if err := natsClient.StartMatchingConsumer(ctx, matchingService); err != nil {
		log.Fatal(err)
	}

	log.Println("matching worker started")

	select {}
}
